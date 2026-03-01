package pbft

import (
	"fmt"
	"math/rand"
	"time"
	"encoding/csv"
	"os"
	"encoding/json"
)

// ====== 高亮：支持自定义节点和恶性节点数量 ======
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

	var malNodes map[int]bool
	if maliciousCount >= 0 {
		malNodes = make(map[int]bool)
		if maliciousCount > numNodes {
			maliciousCount = numNodes
		}
		idxs := rand.Perm(numNodes)[:maliciousCount]
		for _, idx := range idxs {
			malNodes[idx] = true
		}
	} else {
		malNodes = make(map[int]bool)
		mCount := int(float64(numNodes) * maliciousRatio)
		idxs := rand.Perm(numNodes)[:mCount]
		for _, idx := range idxs {
			malNodes[idx] = true
		}
	}

	nodes := make([]*Node, numNodes)
	for i := 0; i < numNodes; i++ {
		throughput := 50.0 + rand.Float64()*150.0
		isMal := malNodes[i]
		nodes[i] = NewNode(i, throughput, isMal, useBlst)
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

		ob.SubmitOrder(Buy, 500+rand.Float64()*30, 10+rand.Float64()*3, "Alice")
		ob.SubmitOrder(Sell, 495+rand.Float64()*20, 5+rand.Float64()*6, "Bob")
		ob.SubmitOrder(Buy, 490+rand.Float64()*15, 4+rand.Float64()*2, "Carol")
		ob.SubmitOrder(Sell, 510+rand.Float64()*10, 8+rand.Float64()*5, "David")
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
			if i%2 == 0 {
				ob.SubmitOrder(Buy, 450+rand.Float64()*100, 5+rand.Float64()*10, fmt.Sprintf("User_%d", i))
			} else {
				ob.SubmitOrder(Sell, 440+rand.Float64()*100, 3+rand.Float64()*9, fmt.Sprintf("User_%d", i))
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