package node

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ======================= 【高亮-2026-03-08】新增：可复现的恶意行为配置（按 round 固定随机源） =======================
// BehaviorConfig 用于控制“恶意节点”/“正常节点”的行为概率与延迟策略
// 目的：让同一轮 round 的恶意节点行为更稳定（复现实验曲线），避免每次运行完全随机导致曲线不收敛。
type BehaviorConfig struct {
	// 恶意节点三种行为概率（总和不要求=1，会自动归一化处理）
	MalProbNotSign  float64 // 不签名：返回 error
	MalProbBadSign  float64 // 错签名：返回伪造签名
	MalProbGoodSign float64 // 偶尔正常签名

	// 恶意 leader 在 pre-prepare 作恶概率（pbft1.go 里会用 Node.IsMalicious + rand 来判断）
	// 这里提供配置项，供你后续在 pbft1.go 里替换为 node.Rand() 的结果。
	MalLeaderPrePrepareProb float64

	// 正常节点签名延迟上限（毫秒）
	MaxNormalDelayMs int

	// 恶意“不签名”时的固定延迟（毫秒）
	MalNotSignDelayMs int
}

// DefaultBehaviorConfig 返回默认配置（基本等价于你原 node.go 的行为）
func DefaultBehaviorConfig() BehaviorConfig {
	return BehaviorConfig{
		MalProbNotSign:            1,
		MalProbBadSign:            1,
		MalProbGoodSign:           1,
		MalLeaderPrePrepareProb:   0.5,
		MaxNormalDelayMs:          200,
		MalNotSignDelayMs:         50,
	}
}

// Tier 表示节点的等级类型（例如用于优先级或奖励策略）
type Tier int

const (
	TierLow Tier = iota
	TierNormal
	TierHigh
)

// Node 表示网络中的一个节点
// 【高亮-2026-03-08】改进点：
// 1) 增加 cfg + rng：支持每轮 round 固定随机源，保证同一 round 行为可复现
// 2) 增加 SetRoundSeed：由外部（节点池/模拟器）在每轮开始时注入 round seed
type Node struct {
	ID          int
	m           int
	IsMalicious bool
	Throughput  float64
	Tier        Tier
	bls BLS
	mu     sync.Mutex
	active bool
	// ======================= 【高亮-2026-03-08】新增字段 =======================
	cfg BehaviorConfig
	rng *rand.Rand
}

// 让没有 blst tag 的环境下也可调用 apbft.NewBlstBLS
func NewBlstBLS(id int) BLS {
	return NewSimpleBLSStub(id)
}

// NewProgressNode 创建并返回一个新的 Node（改进版构造器）
// 与旧 NewNode 的区别：多了 cfg（行为配置）与可选 seed（用于复现随机行为）
func NewProgressNode(id int, throughput float64, isMalicious bool, useBlst bool, cfg BehaviorConfig, seed int64) *Node {
	var blsImpl BLS
	if useBlst {
		blsImpl = NewBlstBLS(id)
	} else {
		blsImpl = NewSimpleBLSStub(id)
	}

	// 【高亮-2026-03-08】每个节点拥有独立 rng，避免并发竞争造成随机行为不稳定
	var rng *rand.Rand
	if seed != 0 {
		rng = rand.New(rand.NewSource(seed + int64(id)*10007))
	}

	return &Node{
		ID:          id,
		m:           InitialM,
		IsMalicious: isMalicious,
		Throughput:  throughput,
		Tier:        TierNormal,
		bls:         blsImpl,
		active:      true,

		cfg: cfg,
		rng: rng,
	}
}

// NewNode：保持你原来的 API 不变（兼容 pbft1.go / pbft1main.go 等现有调用）
// 【高亮-2026-03-08】内部改成调用 NewProgressNode，并使用默认 cfg；seed=0 表示退化为全局 rand 行为（与原一致）
func NewNode(id int, throughput float64, isMalicious bool, useBlst bool) *Node {
	return NewProgressNode(id, throughput, isMalicious, useBlst, DefaultBehaviorConfig(), 0)
}

// ======================= 【高亮-2026-03-08】新增：为每轮 round 注入可复现随机种子 =======================
// SetRoundSeed：在每一轮 round 开始时调用，使“恶意节点行为”随 round 可复现
// 推荐用法：节点池构造时 seed=baseSeed+round；或者在 PBFTSimulator.RunRound 开始时批量设置。
func (n *Node) SetRoundSeed(round int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	seed := int64(20260308 + round*1000 + n.ID)
	n.rng = rand.New(rand.NewSource(seed))
}

// RandFloat：统一随机入口（有 rng 用 rng，无 rng 用全局 rand）
// 目的：未来你在 pbft1.go 里涉及恶意概率判断时，也可以改用 node.RandFloat()，实现完全可复现。
func (n *Node) RandFloat() float64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.rng != nil {
		return n.rng.Float64()
	}
	return rand.Float64()
}

// RandIntn：统一随机入口
func (n *Node) RandIntn(k int) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.rng != nil {
		return n.rng.Intn(k)
	}
	return rand.Intn(k)
}

func (n *Node) String() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return fmt.Sprintf(
		"Node-%02d(m=%d, tier=%v, tp=%.2f, mal=%v, active=%v)",
		n.ID, n.m, n.Tier, n.Throughput, n.IsMalicious, n.active,
	)
}

// ======================= 【高亮-2026-03-08】新增：导出访问器 + 封装 BLS（供 apbft 跨包调用，避免访��未导出字段 m/bls） =======================
// M 返回节点 m 值（apbft.SelectLeader 需要按 m 排序；原字段 n.m 未导出）
func (n *Node) M() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.m
}

// PublicKey 导出节点公钥（apbft 需要收集 pubKeys；原 n.bls 未导出）
func (n *Node) PublicKey() []byte {
	n.mu.Lock()
	bls := n.bls
	n.mu.Unlock()
	return bls.PublicKey()
}

// AggregateSignatures 导出签名聚合能力（封装 n.bls）
func (n *Node) AggregateSignatures(sigs [][]byte) ([]byte, error) {
	n.mu.Lock()
	bls := n.bls
	n.mu.Unlock()
	return bls.AggregateSignatures(sigs)
}

// VerifyAggregate 导出聚合签名验证能力（封装 n.bls）
func (n *Node) VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error) {
	n.mu.Lock()
	bls := n.bls
	n.mu.Unlock()
	return bls.VerifyAggregate(pubKeys, message, aggSig)
}

// Sign 对给定消息进行签名
// 【高亮-2026-03-08】改进：恶意行为概率由 cfg 控制；随机源优先使用 n.rng（可复现）
func (n *Node) Sign(message []byte) ([]byte, error) {
	// 读取必要字段（减少锁粒度）
	n.mu.Lock()
	isMal := n.IsMalicious
	tp := n.Throughput
	cfg := n.cfg
	rng := n.rng
	n.mu.Unlock()

	// helper：取随机
	randFloat := func() float64 {
		if rng != nil {
			return rng.Float64()
		}
		return rand.Float64()
	}
	randIntn := func(k int) int {
		if rng != nil {
			return rng.Intn(k)
		}
		return rand.Intn(k)
	}

	if isMal {
		// 归一化三种概率（允许用户不严格设置）
		a := cfg.MalProbNotSign
		b := cfg.MalProbBadSign
		c := cfg.MalProbGoodSign
		sum := a + b + c
		if sum <= 0 {
			// fallback：等价原实现
			choice := randIntn(3)
			switch choice {
			case 0:
				time.Sleep(50 * time.Millisecond)
				return nil, fmt.Errorf("malicious: not signing")
			case 1:
				return []byte(fmt.Sprintf("bad-sign-node-%02d", n.ID)), nil
			default:
				return n.bls.Sign(message)
			}
		}

		p := randFloat() * sum
		switch {
		case p < a:
			time.Sleep(time.Duration(cfg.MalNotSignDelayMs) * time.Millisecond)
			return nil, fmt.Errorf("malicious: not signing")
		case p < a+b:
			return []byte(fmt.Sprintf("bad-sign-node-%02d", n.ID)), nil
		default:
			return n.bls.Sign(message)
		}
	}

	// 正常节点：延迟与吞吐量相关
	delay := time.Duration(1000.0/tp) * time.Millisecond
	maxDelay := time.Duration(cfg.MaxNormalDelayMs) * time.Millisecond
	if delay > maxDelay {
		delay = maxDelay
	}
	time.Sleep(delay)

	return n.bls.Sign(message)
}

func (n *Node) UpdateReward(success bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if success {
		n.m++
		if n.m >= MMax {
			n.m = MMax - 1
			n.active = true
		}
	} else {
		n.m--
		if n.m < MMin {
			n.active = false
		}
	}
}

func (n *Node) IsActive() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.active
}