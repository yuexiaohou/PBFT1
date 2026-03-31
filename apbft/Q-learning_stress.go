package apbft

import (
	"fmt"
	"PBFT1/node"
)

// ======================= 【高亮-2026-03-31】极速修复：关闭真实签名耗时，适配前端自动测试 =======================
// RunStressTestAPI 为前端提供独立运行的压力测试环境，不污染常规的 GlobalSim
// 返回每轮的共识结果 (1: 成功, 0: 失败)
func RunStressTestAPI(totalRounds int, ratio float64, scenario int) []int {
	// 暂时关闭常规控制台日志，加速仿真
	oldLog := EnableLogs
	EnableLogs = false
	defer func() { EnableLogs = oldLog }()

	results := make([]int, totalRounds)

	// 1. 独立初始化一个 Simulator 实例
	specs0 := node.NewPool(1, 100, ratio)
	nodes := make([]*node.Node, 0, len(specs0))
	for _, sp := range specs0 {
		// 【高亮-2026-03-31】修改此处：将 useBlst = true 改为 false，移除真实加密耗时，瞬间完成千轮计算！
		nd := node.NewNode(sp.ID, sp.Throughput, sp.IsMalicious, false)
		nodes = append(nodes, nd)
	}
	// 【高亮-2026-03-31】同步关闭模拟器 BLS 验证 (false)
	sim := NewPBFTSimulator(nodes, false)
	sim.ComputeTiers()

	// 2. 逐轮运行测试（移除场景重置逻辑，极大提升运行速度）
	for r := 1; r <= totalRounds; r++ {
		sim.ComputeTiers()

		// 选主与执行共识
		var success bool
		viewOffset := 0
		for viewOffset < 5 { // 最多容忍 5 次视图转换
			leader := sim.SelectLeader(r, viewOffset)
			if leader == nil {
				break
			}
			// 如果是恶意的或信誉过低，直接跳过
			if leader.IsMalicious || leader.M() <= node.MMin {
				viewOffset++
				continue
			}
			success, _ = sim.RunRoundWithLeader(r, []byte(fmt.Sprintf("tx-%d", r)), leader)
			break
		}

		if success {
			results[r-1] = 1
		} else {
			results[r-1] = 0
		}
	}

	return results
}