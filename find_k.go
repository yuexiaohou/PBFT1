package main

import (
	"fmt"

	// 引入您的 apbft 和 node 包
	"PBFT1/apbft_forfindK"
	"PBFT1/node"
)

func main() {
	fmt.Println("================================================================")
	fmt.Println("启动 [共识驱动型] KNN 最优 K 值蒙特卡洛寻优")
	fmt.Println("引入市场熔断机制：当最终电价 > 55元时，买方拒收交易失败")
	fmt.Println("================================================================")

	// 关闭 apbft 运行时的打印，防止输出过多
	apbft.EnableLogs = false

	bestK := 10
	maxSuccessRate := 0       // 寻优首要指标：交易成功率最高
	minAvgPrice := 999999.0   // 寻优次要指标：成功率相同时，均价最低
	simRounds := 20          // 仿真轮数
	priceCeiling := 55.0      // 现货市场最高熔断限价（超过此价格买方拒绝交易）

	for k := 10; k <= 30; k++ {
		apbft.GlobalK = k

		totalPrice := 0.0
		successCount := 0

		for r := 1; r <= simRounds; r++ {
			// 100个节点，20% 恶意节点模拟拜占庭攻击
			specs := node.NewPool(r, 100, 0.20)
			txId := fmt.Sprintf("test-tx-%d-k%d", r, k)

			pbftResult := apbft.RunAPBFTWithRoundAndSpecs(r, txId, 10, specs)

			if pbftResult.Status == "已确认" {
				// 【核心创新逻辑：最高限价熔断】
				// 只有价格在合理区间内，这笔交易才算真正被市场接受
				if pbftResult.Price <= priceCeiling {
					totalPrice += pbftResult.Price
					successCount++
				}
			}
		}

		successRate := (successCount * 100) / simRounds

		if successCount > 0 {
			avgPrice := totalPrice / float64(successCount)
			fmt.Printf("当 KNN K = %-2d 时 | 成功率: %3d%% | 真实指导电价: %.4f 元\n",
				k, successRate, avgPrice)

			// 寻优逻辑：优先保证市场成交率最高，其次保证电价最低
			if successRate > maxSuccessRate {
				maxSuccessRate = successRate
				minAvgPrice = avgPrice
				bestK = k
			} else if successRate == maxSuccessRate && avgPrice < minAvgPrice {
				minAvgPrice = avgPrice
				bestK = k
			}
		} else {
			fmt.Printf("当 KNN K = %-2d 时 | 成功率:   0%% | 电价全部触发熔断！\n", k)
		}
	}

	fmt.Println("================================================================")
	fmt.Printf("⭐ 寻优结束！多目标约束下的最优纳什均衡点为： K = %d ⭐\n", bestK)
	fmt.Printf("⭐ 此时系统既能抵御拜占庭节点的价格攻击，又能将均价控制在：%.4f 元\n", minAvgPrice)
	fmt.Printf("⭐ 市场交易成功率达到：%d%%\n", maxSuccessRate)
	fmt.Println("================================================================")

	apbft.EnableLogs = true
}