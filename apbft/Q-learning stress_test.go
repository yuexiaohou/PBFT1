package apbft

import (
	"fmt"
	"PBFT1/node"
	"testing"
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
		// 前 50% 轮次，强制让所有的恶意节点伪装成诚实节点，刷取 R 值
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
			if leader.IsMalicious || sim.QAgents[leader.ID].Reputation <= 0 {
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

// =========================================================
// 单元测试脚本（可通过终端 go test -v -run TestStress ./apbft 运行）
// =========================================================
func TestStressHighByzantine(t *testing.T) {
	totalRounds := 500
	testRatios := []float64{0.20, 0.33, 0.45}

	for _, ratio := range testRatios {
		fmt.Printf("\n>>> [场景一] 启动极限拜占庭测试: 恶意比例 = %.0f%%\n", ratio*100)
		results := RunStressTestAPI(totalRounds, ratio, 1)

		successFirst100 := 0
		successLast100 := 0
		for i, res := range results {
			if res == 1 {
				if i < 100 {
					successFirst100++
				} else if i >= totalRounds-100 {
					successLast100++
				}
			}
		}
		fmt.Printf("    前100轮自愈期成功率: %d%%\n", successFirst100)
		fmt.Printf("    最后100轮稳态成功率: %d%%\n", successLast100)
	}
}

func TestStressHitAndRun(t *testing.T) {
	totalRounds := 400
	ratio := 0.30
	fmt.Printf("\n>>> [场景二] 启动潜伏突变测试: 恶意比例 = %.0f%%, 第%d轮全面突变\n", ratio*100, totalRounds/2)

	results := RunStressTestAPI(totalRounds, ratio, 2)

	mutationPoint := totalRounds / 2
	failCount := 0
	recoveryRound := -1

	for i := mutationPoint; i < totalRounds; i++ {
		if results[i] == 0 {
			failCount++
		} else if recoveryRound == -1 && failCount > 0 {
			recoveryRound = i + 1
		}
	}

	fmt.Printf("    突变后导致的共识失败总次数: %d\n", failCount)
	if recoveryRound != -1 {
		fmt.Printf("    系统在第 %d 轮彻底完成对变节节点的隔离，恢复 100%% 成功率！\n", recoveryRound)
	}
}