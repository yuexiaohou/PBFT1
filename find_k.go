package main

import (
	"fmt"

	// 引入您的 apbft 和 node 包 (请根据您的 go.mod 模块名调整 "PBFT1")
	"PBFT1/apbft"
	"PBFT1/node"
)

func main() {
	fmt.Println("================================================================")
	fmt.Println("启动 [共识驱动型] KNN 最优 K 值蒙特卡洛寻优")
	fmt.Println("正在直接调用底层 APBFT 引擎执行真实交易共识测试...")
	fmt.Println("================================================================")

	// 关闭 apbft 运行时的打印，防止 3000 次循环把控制台淹没
	apbft.EnableLogs = false

	bestK := 1
	minAvgPrice := 999999.0
	simRounds := 100 // 每个 K 值跑 100 次真实共识来求平均值

	// 遍历测试 K = 1 到 30
	for k := 1; k <= 30; k++ {
		// 1. 将当前的 K 注入到 APBFT 共识引擎的内存中
		apbft.GlobalK = k

		totalPrice := 0.0
		successCount := 0

		// 2. 连续发起 simRounds 笔真实的微电网交易共识
		for r := 1; r <= simRounds; r++ {
			// 初始化节点池：100个节点，20% 恶意节点
			specs := node.NewPool(r, 100, 0.20)

			txId := fmt.Sprintf("test-tx-%d-k%d", r, k)

			// 【核心！】这里调用的是您的 apbft.go 中原汁原味的共识函数
			pbftResult := apbft.RunAPBFTWithRoundAndSpecs(r, txId, 10, specs)

			// 只有共识成功的交易（抵御了拜占庭攻击的），其价格才纳入统计
			if pbftResult.Status == "已确认" {
				totalPrice += pbftResult.Price
				successCount++
			}
		}

		if successCount > 0 {
			avgPrice := totalPrice / float64(successCount)
			fmt.Printf("当 KNN 聚类 K = %-2d 时，真实的共识指导电价为: %.4f 元 (成功率: %d%%)\n",
				k, avgPrice, (successCount*100)/simRounds)

			// 记录能让全网电价最低的最优 K
			if avgPrice < minAvgPrice {
				minAvgPrice = avgPrice
				bestK = k
			}
		} else {
			fmt.Printf("当 KNN 聚类 K = %-2d 时，共识全部失败！\n", k)
		}
	}

	fmt.Println("================================================================")
	fmt.Printf("⭐ 真实引擎寻优结束！对抗拜占庭攻击下的最优纳什均衡点为： K = %d ⭐\n", bestK)
	fmt.Printf("⭐ 此时系统的全局平均成交电价最低，为：%.4f 元\n", minAvgPrice)
	fmt.Println("================================================================")

	// 测试结束���，把日志开关恢复，以免影响后续主程序的运行
	apbft.EnableLogs = true
}