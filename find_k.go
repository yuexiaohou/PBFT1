package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// 提取自 apbft.go 的基础参数
const (
	basePrice     = 20.0  // 基础电价 (按我们刚刚的修改)
	lineLossCoeff = 0.2   // 线损系数 (按我们刚刚的修改)
	numNodes      = 50    // 模拟的电网节点总数
	simRounds     = 1000  // 每个 K 值跑 1000 轮蒙特卡洛模拟求平均
)

type SimNeighbor struct {
	Distance float64
	Quote    float64
}

// 模拟一轮 KNN 定价过程
func simulateOneRoundKNN(k int, rng *rand.Rand) float64 {
	// 1. 生成所有节点的数据
	neighbors := make([]SimNeighbor, 0, numNodes)
	for i := 0; i < numNodes; i++ {
		// 模拟距离: 0~100 之间的随机数
		distance := rng.Float64() * 100.0

		// 距离太远，节点可能会拒绝交易（模拟 apbft.go 里的 Reject 逻辑）
		rejectProb := distance * 0.004
		if rng.Float64() < rejectProb {
			continue // 节点拒绝，不参与本次 KNN 计算
		}

		// 模拟卖方报价: 15 + 0~10 的波动 (期望约 20)
		quote := 15.0 + rng.Float64()*10.0

		neighbors = append(neighbors, SimNeighbor{
			Distance: distance,
			Quote:    quote,
		})
	}

	// 2. 按距离从近到远排序
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Distance < neighbors[j].Distance
	})

	// 3. 截取前 K 个邻居 (如果不够 K 个，就取全部)
	actualK := k
	if len(neighbors) < k {
		actualK = len(neighbors)
	}

	if actualK == 0 {
		return basePrice // 极端情况：没人参与，返回基础价
	}

	sumQuote := 0.0
	sumDistance := 0.0
	for i := 0; i < actualK; i++ {
		sumQuote += neighbors[i].Quote
		sumDistance += neighbors[i].Distance
	}

	avgQuote := sumQuote / float64(actualK)
	avgDistance := sumDistance / float64(actualK)

	// 4. 最终价格计算公式
	finalPrice := basePrice + avgQuote + (avgDistance * lineLossCoeff)
	return finalPrice
}

func main() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Println("==================================================")
	fmt.Println("启动 KNN 最优 K 值蒙特卡洛仿真寻优 (Monte Carlo Simulation)")
	fmt.Println("==================================================")
	fmt.Printf("基础参数: 基础电价=%.1f, 线损系数=%.1f, 节点数=%d, 仿真轮数=%d\n\n",
		basePrice, lineLossCoeff, numNodes, simRounds)

	bestK := 1
	minAvgPrice := 999999.0

	// 遍历测试 K = 1 到 30
	for k := 1; k <= 30; k++ {
		totalPrice := 0.0

		// 对当前 K 值运行 1000 轮
		for r := 0; r < simRounds; r++ {
			totalPrice += simulateOneRoundKNN(k, rng)
		}

		avgPrice := totalPrice / float64(simRounds)

		fmt.Printf("当 KNN 聚类 K = %-2d 时，平均指导电价为: %.4f 元\n", k, avgPrice)

		// 记录最低价格
		if avgPrice < minAvgPrice {
			minAvgPrice = avgPrice
			bestK = k
		}
	}

	fmt.Println("==================================================")
	fmt.Printf("⭐ 寻优结束！系统的最优纳什均衡点为： K = %d ⭐\n", bestK)
	fmt.Printf("⭐ 此时系统的全局平均成交电价最低，为：%.4f 元\n", minAvgPrice)
	fmt.Println("==================================================")
	fmt.Println("💡 建议：请将此 K 值硬编码进 apbft/apbft.go 或 config.go 中，以获得最佳系统效益。")
}