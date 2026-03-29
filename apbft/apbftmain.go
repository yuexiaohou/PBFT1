package apbft

import (
	"fmt"
	"math/rand"
	"time"
	"encoding/csv"
	"os"
	"encoding/json"
	"PBFT1/node"
)

// ====== 高亮：支持自定义节点和恶性节点数量 ======
//======“共享同一批 specs”（恶意集合/吞吐量等输入一致），这就是 main.go 里 simulateCUSTOM 的做法============
func RunPBFTSimulator(numNodes int, maliciousCount int, maliciousRatio float64, totalRounds int) {
	var csvWriter *csv.Writer

	tradeLogger, err := NewTradeLog("trade.log")
	if err != nil {
		fmt.Println("Failed to open trade.log:", err)
		return
	}
	defer tradeLogger.Close()
	ob := NewOrderBook()

	useBlst := false
	rand.Seed(time.Now().UnixNano())

	// ======================= 【高亮-2026-03-08】Fix：nodepool.go 内部会强制固定节点数/恶意率；这里对齐 simulateCUSTOM 的写法 =======================
    _ = maliciousCount
    maliciousRatio = node.FixedMaliciousRatio
    numNodes = node.FixedNumNodes

    // ======================= 【高亮-2026-03-08】Fix：初始化一次 specs（共用节点池规格），并实例化为 node.Node；节点在 sim 中跨轮演化 =======================
    specs := node.NewPool(0, numNodes, maliciousRatio)
	nodes := make([]*node.Node, 0, len(specs))
	for _, sp := range specs {
		nd := node.NewNode(sp.ID, sp.Throughput, sp.IsMalicious, useBlst)
		nodes = append(nodes, nd)
	}

	sim := NewPBFTSimulator(nodes, useBlst)
	sim.ComputeTiers()

	fmt.Println("Initial node statuses:")
	for _, nd := range sim.nodes {
		fmt.Println(nd.String())
	}
	ob = NewOrderBook()

	for r := 0; r < totalRounds; r++ {
		if r%5 == 0 && r > 0 {
			for _, nd := range sim.nodes {
				nd.Throughput = nd.Throughput * (0.9 + rand.Float64()*0.2)
			}
			sim.ComputeTiers()
			fmt.Println("\nRecomputed tiers:")
			for _, nd := range sim.nodes {
				fmt.Println(nd.String())
			}
		}
		request := []byte(fmt.Sprintf("request-%d", r))
		ok := sim.RunRound(r, request)
		if !ok {
			fmt.Printf("Round %d failed\n", r)
		}
        // ======================= 【高亮-2026-03-11】关键修改：共识失败则不撮合（不提交订单/不MatchAndClear） =======================
    	if !ok {
    		// ===== 写同步共识结果（即使失败也落盘，便于对齐 round）=====
    		saveConsensusResult(r, sim, "/tmp/pbft_result.json")
    		if csvWriter != nil {
    			csvWriter.Flush()
    		}
            time.Sleep(200 * time.Millisecond)
    		continue
    	}
        // ======================= 【高亮-2026-03-11】关键修改结束 =======================

		// ======================= 【修改四：降低固定机器人的挂单价格】 =======================
        ob.SubmitOrder(Buy, 50+rand.Float64()*15, 10+rand.Float64()*3, "Alice")   // 50~65 元买
		ob.SubmitOrder(Sell, 45+rand.Float64()*15, 5+rand.Float64()*6, "Bob")     // 45~60 元卖
		ob.SubmitOrder(Buy, 48+rand.Float64()*10, 4+rand.Float64()*2, "Carol")    // 48~58 元买
		ob.SubmitOrder(Sell, 52+rand.Float64()*10, 8+rand.Float64()*5, "David")   // 52~62 元卖
		trades := ob.MatchAndClear()

		for _, t := range trades {
			tradeLogger.LogTrade(t)
		}
		tradeLogger.LogSingleOrderBook(0, ob)

		// ===== 写同步共识结果 =====
		saveConsensusResult(r, sim, "/tmp/pbft_result.json")

		if csvWriter != nil {
			csvWriter.Flush()
		}

		numOrders := 5
		for i := 0; i < numOrders; i++ {
			// ======================= 【修改五：降低随机散户的挂单价格】 =======================
			if i%2 == 0 {
				ob.SubmitOrder(Buy, 40+rand.Float64()*30, 5+rand.Float64()*10, fmt.Sprintf("User_%d", i)) // 40~70 买
			} else {
				ob.SubmitOrder(Sell, 35+rand.Float64()*30, 3+rand.Float64()*9, fmt.Sprintf("User_%d", i)) // 35~65 卖
			}
		}
		trades = ob.MatchAndClear()
		if len(trades) > 0 {
			fmt.Printf("Round %d matched trades:\n", r)
			for _, t := range trades {
				fmt.Printf("BuyOrderID: %d, SellOrderID: %d, Price: %.2f, Quantity: %.2f\n",
					t.BuyOrderID, t.SellOrderID, t.Price, t.Quantity)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func saveConsensusResult(round int, sim *PBFTSimulator, filename string) {
	result := map[string]interface{}{
		"TxId":         fmt.Sprintf("round%d", round),
		"Status":       "成功",
		"Consensus":    "pbft",
		"BlockHeight":  round,
		"Timestamp":    time.Now(),
		"Validators":   []map[string]interface{}{},
		"FailedReason": "",
	}
	for _, nd := range sim.nodes {
		if nd.IsActive() {
			result["Validators"] = append(result["Validators"].([]map[string]interface{}), map[string]interface{}{
				"ID":   fmt.Sprintf("node%d", nd.ID),
				"Vote": "commit",
			})
		}
	}
	data, _ := json.Marshal(result)
	_ = os.WriteFile(filename, data, 0644)
}