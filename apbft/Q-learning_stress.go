package apbft

import (
	"fmt"
	"PBFT1/node"
)

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
		nd := node.NewNode(sp.ID, sp.Throughput, sp.IsMalicious, true)
		nodes = append(nodes, nd)
	}
	sim := NewPBFTSimulator(nodes, true)
	sim.ComputeTiers()

	// 2. 逐轮运行测试
	for r := 1; r <= totalRounds; r++ {
		specs := node.NewPool(r, 100, ratio)

		// 同步恶意状态与吞吐量
		for i, sp := range specs {
			sim.nodes[i].IsMalicious = sp.IsMalicious
			sim.nodes[i].Throughput = sp.Throughput
		}

		// 【场景 2：潜伏与突变 (Hit-and-Run)】
		if scenario == 2 {
			if r <= totalRounds/2 {
				for _, nd := range sim.nodes {
					nd.IsMalicious = false
				}
			}
		}

		sim.ComputeTiers()

		// 选主与执行共识
		var success bool
		viewOffset := 0
		for viewOffset < 5 { // 最多容忍 5 次视图转换
			leader := sim.SelectLeader(r, viewOffset)
			if leader == nil {
				break
			}
			// 避免调用尚未实现的 QAgents 属性，这里做简化处理：如果是恶意的或信誉过低，直接跳过
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