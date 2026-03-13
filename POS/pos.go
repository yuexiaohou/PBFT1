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
			Malicious: sp.IsMalicious, // 【高亮-2026-03-11】修正：初始化时记录恶意状态
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
	        // 【高亮-2026-03-11】核心修复：同步恶意标记。如果不同步此字段，所有节点都会投 commit
            n.Malicious = sp.IsMalicious
            if stakeOverride {
            n.Stake = sp.Stake
		    }
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
// ======================= 【高亮-2026-03-11】修改：RunPOSWithRoundAndSpecs 对齐委员会规模与共识阈值 =======================
func RunPOSWithRoundAndSpecs(round int, txId string, amount int, nodes []*SimNode, specs []node.NodeSpec, cfg SimConfig) POSResult {
	// 1. 同步本轮状态（包含恶意标记同步）
	SyncNodesFromSpecs(nodes, specs, false)

	n := len(nodes)
	// 【对齐点】委员会人数固定为总人数的 2/3 (对齐 PBFT 的 2f+1 逻辑权重)
	committeeSize := (n * 2) / 3
	if committeeSize < 1 { committeeSize = 1 }

	// 统一随机种子：保证同一轮次结果可复现
	seed := int64(20260308 + round)
	rng := rand.New(rand.NewSource(seed))

	// 2. 选取 Leader (基于 Stake 权重)
	leaderNode := weightedPickOneWithRNG(nodes, rng)
	if leaderNode == nil {
		return POSResult{TxId: txId, Status: "失败", FailedReason: "no active nodes", Timestamp: time.Now()}
	}

	// 3. 选取委员会成员
	committeeNodes := weightedPickKWithRNG(nodes, committeeSize, leaderNode.ID, rng)
	committeeNames := make([]string, 0, len(committeeNodes))
	votes := make([]Vote, 0, len(committeeNodes))
	commitCount := 0

	// 4. 执行投票逻辑
	for _, v := range committeeNodes {
		committeeNames = append(committeeNames, v.Name())
		voteStr := "commit"

		// 【对齐点】恶意节点行为逻辑与 PBFT 对齐
		if v.Malicious {
			// 恶意节点：大概率拒绝
			if rng.Float64() < 0.60 {
				voteStr = "reject"
				applyStakeDelta(v, -cfg.MaliciousPenalty, cfg)
			}
		} else {
			// 正常节点：极小概率网络抖动
			if rng.Float64() < 0.05 {
				voteStr = "reject"
			}
		}

		if voteStr == "commit" {
			commitCount++
			applyStakeDelta(v, cfg.VoterReward, cfg)
		}
		votes = append(votes, Vote{ID: v.Name(), Vote: voteStr})
	}

	// 5. 【对齐点】共识阈值判定：委员会内 2/3 赞成
	quorum := (len(committeeNodes) * 2) / 3
	if quorum < 1 { quorum = 1 }

	status := "已确认"
	reason := ""
	price := 0.0

	if commitCount < quorum {
		status = "失败"
		reason = fmt.Sprintf("Consensus failed: votes %d/%d (threshold 2/3)", commitCount, len(committeeNodes))
		applyStakeDelta(leaderNode, -cfg.LeaderPenalty, cfg)
	} else {
		// 【对齐点】撮合成功价格生成逻辑对齐
		price = 500.0 + rng.Float64()*20.0
		applyStakeDelta(leaderNode, cfg.LeaderReward, cfg)
	}

	posHeight++

	return POSResult{
		TxId:         txId,
		Status:       status,
		Consensus:    "pos",
		BlockHeight:  round,
		Timestamp:    time.Now(),
		Leader:       leaderNode.Name(),
		Committee:    committeeNames,
		Votes:        votes,
		FailedReason: reason,
		Price:        price,
		SellNode:     leaderNode.Name(),
	}
}

// 兼容你现有调用方式（不传 nodes/cfg 时，每次新建节点集合）
// 注意：每次新建会导致 stake 不累积（不利于奖惩效果），建议你在仿真里复用 nodes。
func RunPOS(txId string, amount int) POSResult {
    cfg := DefaultSimConfig()
	// round=1：用共用节点池初始化 stake/active
	specs := node.NewPool(1, node.FixedNumNodes, node.FixedMaliciousRatio)
	nodes := NewNodesFromSpecs(specs)

	// 本次 RunPOS 作为单次调用：round=1
	return RunPOSWithRoundAndSpecs(1, txId, amount, nodes, specs, cfg)
}