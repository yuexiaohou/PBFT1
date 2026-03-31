package node

import (
	"math/rand"
)

// ======================= 【高亮-2026-03-08】强制固定：全局共用节点数/恶意率（四算法统一） =======================
const FixedNumNodes = 100
const FixedMaliciousRatio = 0.20

// ======================= 【高亮-2026-03-08】新增：通用节点规格 NodeSpec（所有算法共用） =======================
// NodeSpec：公共节点规格（不绑定 PBFT1/POS/RAFT 任意实现）
// 四算法要共用“节点集”，推荐共用的是这个规格，而不是共用某个算法自己的 Node struct。
type NodeSpec struct {
	ID          int
	IsMalicious bool
	Throughput  float64 // PBFT/自定义撮合 可用（模拟延迟/处理能力）
	Stake       float64 // POS 可用
	Active      bool
}

// ======================= 【高亮-2026-03-08】新增：每轮 round 固定恶意节点集合的节点池构造 =======================
// NewPool：按 round 固定一批恶意节点（同一 round 可复现同一集合）
// - round：共识轮次（影响随机种子）
// - numNodes：节点数
// - maliciousRatio：恶意比例（0~1）
// 返回：NodeSpec 切片（长度 numNodes）
func NewPool(round int, numNodes int, maliciousRatio float64) []NodeSpec {
	// ======================= 【高亮-2026-03-31】修改：解除恶意率硬编码，支持动态传参 =======================
	// 如果传入的节点数不合法，则使用全局常量兜底
	if numNodes <= 0 {
		numNodes = FixedNumNodes
	}
	// 删除了 maliciousRatio = FixedMaliciousRatio 这一行
	// 这样，主体常规实验传入 0.20 就会使用 0.20，Q-Learning 测试传入 0.40 就会使用 0.40

	// 用 round 固定随机种子：保证同一轮恶意点集合稳定
	seed := int64(20260308 + round)
	rng := rand.New(rand.NewSource(seed))

	mCount := int(float64(numNodes) * maliciousRatio)
	if mCount < 0 {
		mCount = 0
	}
	if mCount > numNodes {
		mCount = numNodes
	}

	malSet := make(map[int]bool, mCount)
	if mCount > 0 {
		idxs := rng.Perm(numNodes)[:mCount]
		for _, idx := range idxs {
			malSet[idx] = true
		}
	}

	out := make([]NodeSpec, 0, numNodes)
	for i := 0; i < numNodes; i++ {
		out = append(out, NodeSpec{
			ID:          i,
			IsMalicious: malSet[i],
			Throughput:  50.0 + rng.Float64()*150.0, // 50~200
			Stake:       10.0 + rng.Float64()*90.0,  // 10~100
			Active:      true,
		})
	}
	return out
}