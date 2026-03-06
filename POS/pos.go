package pos

import (
	"fmt"
	"math"
	"math/rand"
	"time"
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
// ======================= 2026-03-06 高亮新增：POS结果结构 END =======================

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
		MaliciousProb:    0.03,
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

// ======================= 2026-03-06 高亮新增：节点集合（含stake与激活状态） BEGIN =======================
type SimNode struct {
	ID     int
	Stake  float64
	Active bool
}

func (n *SimNode) Name() string {
	return fmt.Sprintf("node-%d", n.ID)
}
// ======================= 2026-03-06 高亮新增：节点集合 END =======================

// 初始化节点（随机 stake）
func NewNodes(cfg SimConfig) []*SimNode {
	nodes := make([]*SimNode, 0, cfg.ValidatorNum)
	for i := 0; i < cfg.ValidatorNum; i++ {
		st := cfg.StakeMin + rand.Float64()*(cfg.StakeMax-cfg.StakeMin)
		nodes = append(nodes, &SimNode{ID: i, Stake: st, Active: true})
	}
	return nodes
}

// 按 stake 权重抽一个 active 节点
func weightedPickOne(nodes []*SimNode) *SimNode {
	total := 0.0
	for _, n := range nodes {
		if n.Active && n.Stake > 0 {
			total += n.Stake
		}
	}
	if total <= 0 {
		return nil
	}

	r := rand.Float64() * total
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

// 按 stake 权重抽 k 个不重复的 active 节点（不含 leader）
func weightedPickK(nodes []*SimNode, k int, excludeID int) []*SimNode {
	picked := make([]*SimNode, 0, k)
	used := map[int]bool{excludeID: true}

	// 防止死循环：最多尝试次数
	maxTry := k * 50
	for len(picked) < k && maxTry > 0 {
		maxTry--
		n := weightedPickOne(nodes)
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

// ======================= 2026-03-06 高亮新增：单轮POS共识（leader+committee+奖惩） BEGIN =======================
func RunPOSWithNodes(txId string, amount int, nodes []*SimNode, cfg SimConfig) POSResult {
	leader := weightedPickOne(nodes)
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

	committeeNodes := weightedPickK(nodes, cfg.CommitteeSize, leader.ID)
	committee := make([]string, 0, len(committeeNodes))
	votes := make([]Vote, 0, len(committeeNodes))

	commitCount := 0

	for _, v := range committeeNodes {
		committee = append(committee, v.Name())

		// 离线
		if rand.Float64() < cfg.OfflineProb {
			votes = append(votes, Vote{ID: v.Name(), Vote: "offline"})
			applyStakeDelta(v, -cfg.OfflinePenalty, cfg)
			continue
		}

		// 双签（简化：直接重罚）
		if rand.Float64() < cfg.DoubleSignProb {
			votes = append(votes, Vote{ID: v.Name(), Vote: "double-sign"})
			applyStakeDelta(v, -cfg.DoubleSignSlash, cfg)
			continue
		}

		// 作恶：投 reject
		if rand.Float64() < cfg.MaliciousProb {
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

	// 价格模拟（保持你原有风格）
	price := 480.0 + float64(rand.Intn(40))

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
// ======================= 2026-03-06 高亮新增：单轮POS共识 END =======================

// 兼容你现有调用方式（不传 nodes/cfg 时，每次新建节点集合）
// 注意：每次新建会导致 stake 不累积（不利于奖惩效果），建议你在仿真里复用 nodes。
func RunPOS(txId string, amount int) POSResult {
	rand.Seed(time.Now().UnixNano())
	cfg := DefaultSimConfig()
	nodes := NewNodes(cfg)
	return RunPOSWithNodes(txId, amount, nodes, cfg)
}