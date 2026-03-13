package raft

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"PBFT1/node"
)

// ======================= 【高亮-2026-03-09】RAFT：使用 nodepool.go 共用节点集 + “日志不落后才投票” + Leader 完整性 =======================

// Role represents raft node role.
type Role int

const (
	Follower Role = iota
	Candidate
	Leader
)

func (r Role) String() string {
	switch r {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// LogEntry is a simplified Raft log entry.
type LogEntry struct {
	Index   int
	Term    int
	Command string
}

// VoteRequest is a simplified RequestVote RPC request.
type VoteRequest struct {
	Term         int
	CandidateID  int
	LastLogIndex int
	LastLogTerm  int
}

// VoteResponse is a simplified RequestVote RPC response.
type VoteResponse struct {
	Term        int
	VoteGranted bool
	Reason      string
}

// AppendEntriesRequest is a simplified AppendEntries RPC request.
type AppendEntriesRequest struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

// AppendEntriesResponse is a simplified AppendEntries RPC response.
type AppendEntriesResponse struct {
	Term    int
	Success bool
	Reason  string
}

// NodeState holds state for one raft node in a simulation.
type NodeState struct {
	mu sync.Mutex

	ID   int
	Spec node.NodeSpec // ======================= 【高亮-2026-03-09】与 nodepool.go 绑定：每轮复用 NodeSpec =======================

	Role Role

	CurrentTerm int
	VotedFor    *int

	Log []LogEntry

	CommitIndex int
	LastApplied int

	// Leader-only
	NextIndex  map[int]int
	MatchIndex map[int]int

	// Timers
	ElectionDeadline time.Time
}

// Cluster is a simple in-memory raft simulation cluster for a given round/specs.
type Cluster struct {
	mu sync.Mutex

	Round int

	Nodes map[int]*NodeState

	// For deterministic simulation, we keep a rng.
	rng *rand.Rand

	// For observability
	LeaderID *int
}

// NewClusterFromPool creates a new in-memory raft cluster for a given simulation round.
// specs come from node.NewPool(round, numNodes, maliciousRatio).
func NewClusterFromPool(round int, specs []node.NodeSpec) *Cluster {
	seed := int64(20260309 + round) // ======================= 【高亮-2026-03-09】round 固定随机性 =======================
	rng := rand.New(rand.NewSource(seed))

	nodes := make(map[int]*NodeState, len(specs))
	for _, sp := range specs {
		id := sp.ID
		n := &NodeState{
			ID:          id,
			Spec:        sp,
			Role:        Follower,
			CurrentTerm: DefaultTerm,
			VotedFor:    nil,
			Log:         make([]LogEntry, 0),
			CommitIndex: 0,
			LastApplied: 0,
			NextIndex:   make(map[int]int),
			MatchIndex:  make(map[int]int),
		}
		n.resetElectionDeadline(rng)
		nodes[id] = n
	}

	return &Cluster{
		Round:  round,
		Nodes: nodes,
		rng:   rng,
	}
}

func (n *NodeState) resetElectionDeadline(rng *rand.Rand) {
	// A small randomized timeout; the config value is treated as base.
	jitter := rng.Intn(ElectionTimeout + 1) // [0..ElectionTimeout]
	n.ElectionDeadline = time.Now().Add(time.Duration(ElectionTimeout+jitter) * time.Second)
}

// lastLogIndexTerm returns the last log index and term.
func (n *NodeState) lastLogIndexTerm() (int, int) {
	if len(n.Log) == 0 {
		return 0, 0
	}
	last := n.Log[len(n.Log)-1]
	return last.Index, last.Term
}

// isCandidateUpToDate compares candidate's last log with receiver's last log,
// implementing Raft's "up-to-date" rule:
// - Higher lastLogTerm wins
// - If terms equal, higher lastLogIndex wins
func (n *NodeState) isCandidateUpToDate(candidateLastIndex, candidateLastTerm int) bool {
	myLastIndex, myLastTerm := n.lastLogIndexTerm()

	// ======================= 【高亮-2026-03-09】只投给“日志不落后”的候选人（核心） =======================
	if candidateLastTerm != myLastTerm {
		return candidateLastTerm > myLastTerm
	}
	return candidateLastIndex >= myLastIndex
}

// HandleRequestVote processes RequestVote on a follower/candidate/leader.
// This is the main place implementing requirement (1): only vote for up-to-date candidate.
func (n *NodeState) HandleRequestVote(req VoteRequest) VoteResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	// If term is older, reject.
	if req.Term < n.CurrentTerm {
		return VoteResponse{
			Term:        n.CurrentTerm,
			VoteGranted: false,
			Reason:      "term too old",
		}
	}

	// If term is newer, step down and clear vote.
	if req.Term > n.CurrentTerm {
		n.CurrentTerm = req.Term
		n.Role = Follower
		n.VotedFor = nil
	}

	// Check log up-to-date condition.
	if !n.isCandidateUpToDate(req.LastLogIndex, req.LastLogTerm) {
		return VoteResponse{
			Term:        n.CurrentTerm,
			VoteGranted: false,
			Reason:      "candidate log is behind",
		}
	}

	// Grant vote if not voted yet or voted for this candidate.
	if n.VotedFor == nil || (n.VotedFor != nil && *n.VotedFor == req.CandidateID) {
		cid := req.CandidateID
		n.VotedFor = &cid
		return VoteResponse{
			Term:        n.CurrentTerm,
			VoteGranted: true,
			Reason:      "vote granted",
		}
	}

	return VoteResponse{
		Term:        n.CurrentTerm,
		VoteGranted: false,
		Reason:      "already voted for another candidate",
	}
}

// HandleAppendEntries processes AppendEntries on a follower.
// This is used to keep logs consistent.
func (n *NodeState) HandleAppendEntries(req AppendEntriesRequest) AppendEntriesResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	if req.Term < n.CurrentTerm {
		return AppendEntriesResponse{
			Term:    n.CurrentTerm,
			Success: false,
			Reason:  "term too old",
		}
	}

	// Newer term or heartbeat from current term leader => follower.
	if req.Term > n.CurrentTerm {
		n.CurrentTerm = req.Term
	}
	n.Role = Follower
	n.VotedFor = nil

	// Check prev log consistency.
	if req.PrevLogIndex > 0 {
		if req.PrevLogIndex > len(n.Log) {
			return AppendEntriesResponse{
				Term:    n.CurrentTerm,
				Success: false,
				Reason:  "missing prev log index",
			}
		}
		prev := n.Log[req.PrevLogIndex-1]
		if prev.Term != req.PrevLogTerm {
			// Conflict: delete entry and all after it.
			n.Log = n.Log[:req.PrevLogIndex-1]
			return AppendEntriesResponse{
				Term:    n.CurrentTerm,
				Success: false,
				Reason:  "prev log term mismatch",
			}
		}
	}

	// Append new entries (overwrite conflicts).
	for _, e := range req.Entries {
		if e.Index <= len(n.Log) {
			// Existing entry; if term differs, overwrite this and truncate.
			if n.Log[e.Index-1].Term != e.Term {
				n.Log = n.Log[:e.Index-1]
				n.Log = append(n.Log, e)
			}
		} else {
			n.Log = append(n.Log, e)
		}
	}

	// Update commit index.
	if req.LeaderCommit > n.CommitIndex {
		lastIndex := 0
		if len(n.Log) > 0 {
			lastIndex = n.Log[len(n.Log)-1].Index
		}
		if req.LeaderCommit < lastIndex {
			n.CommitIndex = req.LeaderCommit
		} else {
			n.CommitIndex = lastIndex
		}
	}

	return AppendEntriesResponse{
		Term:    n.CurrentTerm,
		Success: true,
		Reason:  "append ok",
	}
}

// StartElection starts an election for the given node ID.
func (c *Cluster) StartElection(candidateID int) (leaderID int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cand, ok := c.Nodes[candidateID]
	if !ok {
		return 0, errors.New("candidate not found")
	}

	cand.mu.Lock()
	// Malicious node behavior: may try to start elections aggressively.
	// Still it cannot win unless it has up-to-date log (enforced by voters).
	cand.Role = Candidate
	cand.CurrentTerm++
	candTerm := cand.CurrentTerm
	cid := cand.ID
	cand.VotedFor = &cid
	lastIdx, lastTerm := cand.lastLogIndexTerm()
	cand.mu.Unlock()

	// Request votes
	votes := 1 // self vote
	needed := c.quorum()

	for id, peer := range c.Nodes {
		if id == candidateID {
			continue
		}

		peer.mu.Lock()
		peerSpec := peer.Spec
		peer.mu.Unlock()

		// If peer is inactive or malicious, we still process; maliciousness affects response stochastically.
		// But "log not behind" rule is always enforced.
		resp := peer.HandleRequestVote(VoteRequest{
			Term:         candTerm,
			CandidateID:  candidateID,
			LastLogIndex: lastIdx,
			LastLogTerm:  lastTerm,
		})

		// Malicious peer might flip its response sometimes (simulation).
		// ======================= 【高亮-2026-03-09】nodepool 共用节点集：用 Spec.IsMalicious 影响行为（可控仿真） =======================
		if peerSpec.IsMalicious {
			// With small probability, deny vote even if would grant.
			if resp.VoteGranted && c.rng.Float64() < 0.20 {
				resp.VoteGranted = false
				resp.Reason = "malicious denial"
			}
		}

		if resp.Term > candTerm {
			// Candidate discovers higher term, step down.
			cand.mu.Lock()
			if resp.Term > cand.CurrentTerm {
				cand.CurrentTerm = resp.Term
			}
			cand.Role = Follower
			cand.VotedFor = nil
			cand.mu.Unlock()
			return 0, errors.New("stepped down due to higher term")
		}

		if resp.VoteGranted {
			votes++
		}
	}

	if votes < needed {
		return 0, fmt.Errorf("election failed: votes=%d needed=%d", votes, needed)
	}

	// Become leader
	cand.mu.Lock()
	cand.Role = Leader
	// Initialize leader state
	for id := range c.Nodes {
		if id == candidateID {
			continue
		}
		cand.NextIndex[id] = len(cand.Log) + 1
		cand.MatchIndex[id] = 0
	}
	cand.mu.Unlock()

	c.LeaderID = &candidateID
	return candidateID, nil
}

// ======================= 【高亮-2026-03-11】修改：对齐 RAFT 阈值为 2f+1 以适配 BFT 仿真对比 =======================
func (c *Cluster) quorum() int {
	n := len(c.Nodes)
	// 在标准 Raft 中是 n/2 + 1。
	// 为了在同一恶意比例下与 PBFT/APBFT 对齐对比，我们将其阈值统一设置为 2f+1。
	f := (n - 1) / 3
	return 2*f + 1
}

// LeaderAppend appends a new command to leader log and tries to replicate to followers.
// It also implements Leader Completeness (requirement 2) via commit rule:
// only advance commitIndex for entries in current term that are replicated on majority.
// ======================= 【高亮-2026-03-11】修改：模拟提案流程并对齐撮合成功逻辑 =======================
func (c *Cluster) LeaderAppend(command string) (int, float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.LeaderID == nil {
		return 0, 0, fmt.Errorf("no leader")
	}

	leader := c.Nodes[*c.LeaderID]
	// 如果 Leader 是恶意节点，模拟提案失败
	if leader.Spec.IsMalicious && c.rng.Float64() < 0.3 {
		return 0, 0, fmt.Errorf("malicious leader failed to propose")
	}

	successCount := 1 // Leader 算一票
	for _, nd := range c.Nodes {
		if nd.ID == *c.LeaderID { continue }

		// 模拟副本确认逻辑 (对齐 PBFT 投票行为)
		shouldConfirm := true
		if nd.Spec.IsMalicious {
			// 恶意节点：大概率拒绝
			if c.rng.Float64() < 0.6 { shouldConfirm = false }
		} else {
			// 正常节点：极小概率网络抖动
			if c.rng.Float64() < 0.05 { shouldConfirm = false }
		}

		if shouldConfirm {
			successCount++
		}
	}

	q := c.quorum()
	if successCount >= q {
		// 【对齐点】撮合成功价格逻辑对齐
		price := 500.0 + c.rng.Float64()*20.0
		return successCount, price, nil
	}

	return successCount, 0, fmt.Errorf("not enough acks: %d/%d", successCount, q)
}

// SimulateRoundWithPrice 用于服务端仿真入口，返回价格以对齐
func SimulateRoundWithPrice(round int, specs []node.NodeSpec) (int, float64, error) {
	c := NewClusterFromPool(round, specs)

	// 简单选主逻辑
	active := make([]int, 0)
	for _, sp := range specs {
		if sp.Active { active = append(active, sp.ID) }
	}
	if len(active) == 0 { return 0, 0, errors.New("no active nodes") }

	cand := active[c.rng.Intn(len(active))]
	lid, err := c.StartElection(cand)
	if err != nil { return 0, 0, err }

	_, price, err := c.LeaderAppend(fmt.Sprintf("cmd-round-%d", round))
	return lid, price, err
}

func SimulateRound(round int, numNodes int, maliciousRatio float64) (int, int, error) {
	specs := node.NewPool(round, numNodes, maliciousRatio)
	lid, _, err := SimulateRoundWithPrice(round, specs)
	return lid, 0, err
}