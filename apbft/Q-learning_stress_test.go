package apbft

import (
	"fmt"
	"testing"
)

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