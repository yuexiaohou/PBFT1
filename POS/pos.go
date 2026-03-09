package pos

import (
	"fmt"
	"math"
	"math/rand"
	"time"
	// ======================= 【高亮-2026-03-09】新增：接入共用节点池 NodeSpec =======================
    "PBFT1/node"
)

// ======================= 2026-03-06 高亮新增：POS结果结构（含委员会与投票） BEGIN =======================
type Vote struct {
	ID   string `json:"id"`
	Vote string `json:"vote"` // commit / reject / offline / malicious
}

type POSResult struct {
	TxId         string
	Status       string
	Consensus    string
	BlockHeight  int
	Timestamp    time.Time
	Leader       string   // leader节点
	Committee    []string // 委员会成员（含leader可选，这里不包含leader）
	Votes        []Vote   // 委员会投票详情
	FailedReason string
	Price        float64
	SellNode     string // 为了兼容你之前字段，这里让 SellNode = Leader
}

var posHeight = 1

// ======================= 2026-03-06 高亮新增：POS仿真参数 BEGIN =======================
type SimConfig struct {
	ValidatorNum  int
	CommitteeSize int

	StakeMin float64
	StakeMax float64

	// 节点行为概率（简化）
	OfflineProb   float64 // 节点离线概率
	MaliciousProb float64 // 节点作恶概率（投 reject 或乱投）
	DoubleSignProb float64 // “双签/矛盾投票”概率（简化为更重惩罚）

	// 共识阈值：>= threshold 才算成功（例如 2/3）
	CommitThreshold float64

	// 奖励惩罚参数
	LeaderReward     float64
	LeaderPenalty    float64
	VoterReward      float64
	OfflinePenalty   float64
	MaliciousPenalty float64
	DoubleSignSlash  float64

	// stake 边界
	MinActiveStake float64 // stake 低于此值视为失效
}
// ======================= 2026-03-06 高亮新增：POS仿真参数 END =======================

// 默认配置（你可以按需调整）
func DefaultSimConfig() SimConfig {
	return SimConfig{
		ValidatorNum:     100,
		CommitteeSize:    21,
		StakeMin:         10,
		StakeMax:         100,
		OfflineProb:      0.05,
		MaliciousProb:    0.20,
		DoubleSignProb:   0.01,
		CommitThreshold:  2.0 / 3.0,
		LeaderReward:     0.8,
		LeaderPenalty:    1.2,
		VoterReward:      0.2,
		OfflinePenalty:   0.5,
		MaliciousPenalty: 1.0,
		DoubleSignSlash:  3.0,
		MinActiveStake:   1.0,
	}
}

// ======================= 2026-03-06 高亮新增：节点集合（含stake与激活状态）（节点池，由于pos算法的节点需要累积stake值，因此需要构建simNode，而不直接使用node.Node，从而实现对nodepool.go的初次调用） BEGIN =======================
type SimNode struct {
	ID        int
	Stake     float64
	Active    bool
	Malicious bool
}

func (n *SimNode) Name() string {
	return fmt.Sprintf("node-%d", n.ID)
}

// ======================= 【高亮-2026-03-09】新增：从共用节点池初始化 POS 节点（stake/active 来自 NodeSpec） BEGIN =======================
func NewNodesFromSpecs(specs []node.NodeSpec) []*SimNode {
	nodes := make([]*SimNode, 0, len(specs))
	for _, sp := range specs {
		nodes = append(nodes, &SimNode{
			ID:     sp.ID,
			Stake:  sp.Stake,
			Active: sp.Active,
		})
	}
	return nodes
}

// 同步 specs -> nodes：只同步 Active；并可选同步 stake（默认不覆盖，保证奖惩可累计）
// stakeOverride=false：只同步 Active，不覆盖 Stake（推荐）
// stakeOverride=true ：同步 stake（会破坏累计奖惩）
func SyncNodesFromSpecs(nodes []*SimNode, specs []node.NodeSpec, stakeOverride bool) {
	// 建立索引：specsByID
	specsByID := make(map[int]node.NodeSpec, len(specs))
	for _, sp := range specs {
		specsByID[sp.ID] = sp
	}

	for _, n := range nodes {
		sp, ok := specsByID[n.ID]
		if !ok {
			continue
		}
		n.Active = sp.Active
		if stakeOverride {
			n.Stake = sp.Stake
		}
	}
}

// ======================= 【高亮-2026-03-09】新增：可注入 RNG 的加权抽取（stake 权重）BEGIN =======================
func weightedPickOneWithRNG(nodes []*SimNode, rng *rand.Rand) *SimNode {
	total := 0.0
	for _, n := range nodes {
		if n.Active && n.Stake > 0 {
			total += n.Stake
		}
	}
	if total <= 0 {
		return nil
	}

	r := rng.Float64() * total
	acc := 0.0
	for _, n := range nodes {
		if !n.Active || n.Stake <= 0 {
			continue
		}
		acc += n.Stake
		if acc >= r {
			return n
		}
	}
	return nil
}

func weightedPickKWithRNG(nodes []*SimNode, k int, excludeID int, rng *rand.Rand) []*SimNode {
	picked := make([]*SimNode, 0, k)
	used := map[int]bool{excludeID: true}

	// 防止死循环：最多尝试次数
	maxTry := k * 50
	for len(picked) < k && maxTry > 0 {
		maxTry--
		n := weightedPickOneWithRNG(nodes, rng)
		if n == nil {
			break
		}
		if used[n.ID] {
			continue
		}
		used[n.ID] = true
		picked = append(picked, n)
	}
	return picked
}

// 兼容旧函数名（保留，但内部用一个临时 rng，避免直接依赖全局 rand）
func weightedPickOne(nodes []*SimNode) *SimNode {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return weightedPickOneWithRNG(nodes, rng)
}

// ======================= 【高亮-2026-03-09】新增：可注入 RNG 的加权抽取 END =======================
func weightedPickK(nodes []*SimNode, k int, excludeID int) []*SimNode {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return weightedPickKWithRNG(nodes, k, excludeID, rng)
}

// stake 更新并按 MinActiveStake 判定 active
func applyStakeDelta(n *SimNode, delta float64, cfg SimConfig) {
	n.Stake = math.Max(0, n.Stake+delta)
	if n.Stake < cfg.MinActiveStake {
		n.Active = false
	}
}

// ======================= 【高亮-2026-03-09】新增：POS 单轮（共用 nodepool + 权重抽取 + 奖惩累计） BEGIN =======================
// RunPOSWithRoundAndSpecs：
// - round：用于固定随机源（可复现）
// - nodes：跨轮复用，stake 会累计奖惩
// - specs：来自 node.NewPool(round, ...) 的共用节点池（本轮恶意集合固定）
// 规则：
// 1) leader/committee 仍按 stake 权重抽取（只从 Active 节点里抽）
// 2) 投票行为：若该委员是恶意节点，则更倾向于 malicious/double-sign/offline
func RunPOSWithRoundAndSpecs(round int, txId string, amount int, nodes []*SimNode, specs []node.NodeSpec, cfg SimConfig) POSResult {
	// 用 round 固定随机源：保证同一轮可复现
	seed := int64(20260309 + round)
	rng := rand.New(rand.NewSource(seed))

	// 同步本轮 Active（不覆盖 stake，保证奖惩累计）
	SyncNodesFromSpecs(nodes, specs, false)

	// 索引：恶意集合
	isMalByID := make(map[int]bool, len(specs))
	for _, sp := range specs {
		isMalByID[sp.ID] = sp.IsMalicious
	}

	return runPOSCoreWithRNG(txId, amount, nodes, cfg, rng, isMalByID)
}

// 内部核心：把“节点行为概率 + 权重抽签 + 奖惩”集中在这里，便于复用
func runPOSCoreWithRNG(txId string, amount int, nodes []*SimNode, cfg SimConfig, rng *rand.Rand, isMalByID map[int]bool) POSResult {
	leader := weightedPickOneWithRNG(nodes, rng)
	posHeight++

	if leader == nil {
		return POSResult{
			TxId:         txId,
			Status:       "失败",
			Consensus:    "pos",
			BlockHeight:  posHeight,
			Timestamp:    time.Now(),
			FailedReason: "no active leader",
			Price:        0,
			SellNode:     "",
			Leader:       "",
			Committee:    []string{},
			Votes:        []Vote{},
		}
	}

	committeeNodes := weightedPickKWithRNG(nodes, cfg.CommitteeSize, leader.ID, rng)
	committee := make([]string, 0, len(committeeNodes))
	votes := make([]Vote, 0, len(committeeNodes))

	commitCount := 0

	for _, v := range committeeNodes {
		committee = append(committee, v.Name())

		// ========== 基于 nodepool 恶意标记，动态调整行为概率 ==========
		offlineProb := cfg.OfflineProb
		doubleSignProb := cfg.DoubleSignProb
		malProb := cfg.MaliciousProb

		if isMalByID != nil && isMalByID[v.ID] {
			// ======================= 【高亮-2026-03-09】恶意节点：提高作恶/双签/离线概率（使 POS 与 APBFT 可比） =======================
			offlineProb = math.Min(1.0, cfg.OfflineProb*2.0)
			doubleSignProb = math.Min(1.0, cfg.DoubleSignProb*3.0)
			malProb = math.Min(1.0, cfg.MaliciousProb*2.5)
		}

		// 离线
		if rng.Float64() < offlineProb {
			votes = append(votes, Vote{ID: v.Name(), Vote: "offline"})
			applyStakeDelta(v, -cfg.OfflinePenalty, cfg)
			continue
		}

		// 双签（简化：直接重罚）
		if rng.Float64() < doubleSignProb {
			votes = append(votes, Vote{ID: v.Name(), Vote: "double-sign"})
			applyStakeDelta(v, -cfg.DoubleSignSlash, cfg)
			continue
		}

		// 作恶：投 reject（这里归为 malicious）
		if rng.Float64() < malProb {
			votes = append(votes, Vote{ID: v.Name(), Vote: "malicious"})
			applyStakeDelta(v, -cfg.MaliciousPenalty, cfg)
			continue
		}

		// 正常投 commit
		votes = append(votes, Vote{ID: v.Name(), Vote: "commit"})
		commitCount++
		applyStakeDelta(v, cfg.VoterReward, cfg)
	}

	// 判断是否达成阈值
	status := "失败"
	reason := "not enough commits"
	if len(committeeNodes) > 0 {
		if float64(commitCount)/float64(len(committeeNodes)) >= cfg.CommitThreshold {
			status = "已确认"
			reason = ""
			applyStakeDelta(leader, cfg.LeaderReward, cfg)
		} else {
			applyStakeDelta(leader, -cfg.LeaderPenalty, cfg)
		}
	} else {
		// 委员会抽不出来（极端情况）
		applyStakeDelta(leader, -cfg.LeaderPenalty, cfg)
		reason = "committee empty"
	}

	// 价格模拟：使用 rng，避免全局随机干扰可复现性
	price := 480.0 + float64(rng.Intn(40))

	return POSResult{
		TxId:         txId,
		Status:       status,
		Consensus:    "pos",
		BlockHeight:  posHeight,
		Timestamp:    time.Now(),
		Leader:       leader.Name(),
		Committee:    committee,
		Votes:        votes,
		FailedReason: reason,
		Price:        price,
		SellNode:     leader.Name(), // 兼容老字段：SellNode=Leader
	}
}

// ======================= 【高亮-2026-03-09】修改：RunPOSWithNodes 改为薄封装（wrapper），统一走 runPOSCoreWithRNG =======================
func RunPOSWithNodes(txId string, amount int, nodes []*SimNode, cfg SimConfig) POSResult {
	// 创建临时 rng：避免依赖全局 rand，提高可控性（但仍然是“非固定 seed”模式）
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	// isMalByID=nil：表示不接入 nodepool 恶意标记，保持旧行为语义
	return runPOSCoreWithRNG(txId, amount, nodes, cfg, rng, nil)
}

// ======================= 2026-03-06 高亮新增：单轮POS共识 END =======================

// 兼容你现有调用方式（不传 nodes/cfg 时，每次新建节点集合）
// 注意：每次新建会导致 stake 不累积（不利于奖惩效果），建议你在仿真里复用 nodes。
func RunPOS(txId string, amount int) POSResult {
	rand.Seed(time.Now().UnixNano())
	cfg := DefaultSimConfig()
	nodes := NewNodes(cfg)
	return RunPOSWithNodes(txId, amount, nodes, cfg)
}