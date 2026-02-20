package main // 声明当前文件属于 main 包，表示这是可独立运行的程序入口

import ( // 导入所需的标准库
	"flag"         // 解析命令行参数
	"fmt"          // 格式化输出
	"math/rand"    // 生成伪随机数
	"time"         // 时间相关操作
    "encoding/csv" // 读写 CSV 文件
    "os"           // 操作系统功能（如文件）
    "strconv"      // 字符串与基本类型转换
	// ==== 电力交易相关 START ====
	// 假定 trade.go 在同目录，并已加入工程
	// ==== 电力交易相关 END ====
)

func main() { // main 函数为程序入口
	// 指定是否使用 blst 库（若要用 blst，需要用 -tags blst 编译且 useBlst=true）
    numNodes := flag.Int("nodes", 100, "节点总数") // 设置节点总数参数，默认100
	maliciousRatio := flag.Float64("malicious", 0.05, "恶性节点比例 (默认为5%)") // 恶性节点比例参数，默认5%
	maliciousCount := flag.Int("mal_count", -1, "恶性节点数 (优先于比例，若>=0则按此指定)") // 恶性节点数参数，若>=0优先用此
	flag.Parse() // 解析命令行参数
	// ---（1）flag 和 csv 初始化部分---
    csvPath := flag.String("csv", "", "输出每轮每节点统计 CSV 路径 (optional)") // 定义命令行参数（CSV 路径）
    var csvFile *os.File    // 声明 CSV 文件句柄
    var csvWriter *csv.Writer // 声明 CSV 写入对象
    if *csvPath != "" {       // 如果传入了 csv 路径
        csvFile, _ = os.Create(*csvPath) // 创建并打开 CSV 文件
        defer csvFile.Close()             // 程序退出时关闭文件
        csvWriter = csv.NewWriter(csvFile) // 创建 CSV writer
        // 写表头，6个字段
        csvWriter.Write([]string{"round","node_id","throughput","m","active","tier","isLeader"})
    }

	// ==== 日志功能补充 START ====
	tradeLogger, err := NewTradeLog("trade.log") // 初始化交易日志对象
	if err != nil { // 如果日志文件打开失败
		fmt.Println("Failed to open trade.log:", err) // 打印错误信息
		return // 程序返回并退出
	}
	defer tradeLogger.Close() // 程序结束时关闭日志对象
	// ==== 日志功能补充 END ====
	ob := NewOrderBook() // 初始化订单簿结构
	// ==== 日志功能补充 END ====

    // ---（2）模拟器和节点初始化与运行---
	useBlst := false // 是否启用 blst 扩展（默认不启用）

    rand.Seed(time.Now().UnixNano()) // 伪随机数种子设定为当前时间
	var malNodes map[int]bool // 恶性节点集合
	if *maliciousCount >= 0 {
		malNodes = make(map[int]bool)
		// 随机选择 mal_count 个恶意节点编号
		idxs := rand.Perm(*numNodes)[:*maliciousCount] // 随机生成编号并取前mal_count个
		for _, idx := range idxs { // 均标记为��性节点
			malNodes[idx] = true
		}
	} else {
		malNodes = make(map[int]bool)
		mCount := int(float64(*numNodes) * *maliciousRatio) // 计算恶性节点数
		idxs := rand.Perm(*numNodes)[:mCount] // 随机选择mCount个
		for _, idx := range idxs {
			malNodes[idx] = true
		}
	}

	nodes := make([]*Node, *numNodes) // 节点切片，容量为总节点数
	for i := 0; i < *numNodes; i++ {
		throughput := 50.0 + rand.Float64()*150.0 // 节点吞吐量，随机区间[50,200)
		isMal := malNodes[i] // 是否恶性节点
		nodes[i] = NewNode(i, throughput, isMal, useBlst) // 创建节点并加到 nodes
	}

	sim := NewPBFTSimulator(nodes, useBlst) // 初始化 PBFT 模拟器
	sim.ComputeTiers() // 计算每个节点的分层（例如高吞吐量/低吞吐量分层）

	fmt.Println("Initial node statuses:") // 输出初始节点状态
	for _, nd := range sim.nodes {        // 遍历所有节点
		fmt.Println(nd.String())      // 输出每个节点的属性描述
	}

// ==== 电力交易相关 START ====
	ob = NewOrderBook() // 再初始化一个订单簿
	// ==== 电力交易相关 END ====


	totalRounds := 20 // 总共模拟 20 轮
	for r := 0; r < totalRounds; r++ {       // 轮次循环
		if r%5 == 0 && r > 0 {                // 每 5 轮调整一次节点吞吐量和层级
			for _, nd := range sim.nodes {    // 遍历节点
				nd.Throughput = nd.Throughput * (0.9 + rand.Float64()*0.2) // 吞吐量在 0.9～1.1 之间波动
			}
			sim.ComputeTiers() // 重新计算分层
			fmt.Println("\nRecomputed tiers:")         // 打印提示
			for _, nd := range sim.nodes {             // 输出分层后节点属性
				fmt.Println(nd.String())
			}
		}
		request := []byte(fmt.Sprintf("request-%d", r)) // 构造当前轮次的请求内容
		ok := sim.RunRound(r, request)                  // 模拟运行一轮 PBFT 共识算法
		if !ok { // 如果该轮失败
			fmt.Printf("Round %d failed\n", r) // 打印警告信息
		}

        // 订单挂单与撮合演示
    	ob.SubmitOrder(Buy, 500+rand.Float64()*30, 10+rand.Float64()*3, "Alice") // Alice买单
    	ob.SubmitOrder(Sell, 495+rand.Float64()*20, 5+rand.Float64()*6, "Bob")   // Bob卖单
    	ob.SubmitOrder(Buy, 490+rand.Float64()*15, 4+rand.Float64()*2, "Carol")  // Carol买单
    	ob.SubmitOrder(Sell, 510+rand.Float64()*10, 8+rand.Float64()*5, "David") // David卖单
    	trades := ob.MatchAndClear() // 撮合成交并返回成交列表

    	// ==== 日志功能补充 START ====
    	for _, t := range trades { // 遍历所有成交
    		tradeLogger.LogTrade(t) // 写入成交
    	}
    	tradeLogger.LogSingleOrderBook(0, ob) // 写入订单簿快照
    	// ==== 日志功能补充 END ====

        // 记录每一轮每个节点至 CSV
        for _, nd := range sim.nodes {
            if csvWriter != nil { // 若启用 CSV 记录
                csvWriter.Write([]string{
                    strconv.Itoa(r),                   // 轮次
                    strconv.Itoa(nd.ID),               // 节点 ID
                    fmt.Sprintf("%.3f", nd.Throughput),// 吞吐量（小数点3位）
                    fmt.Sprintf("%.3f", nd.m),         // m 参数（假设某种指标）
                    strconv.FormatBool(nd.active),     // 节点是否活跃
                    fmt.Sprintf("%v", nd.Tier),        // 节点层级
                    strconv.FormatBool(sim.SelectLeader(r).ID == nd.ID), // 是否为 Leader
                })
            }
        }
        if csvWriter != nil {
            csvWriter.Flush() // 每轮结束后强制刷新文件
        }

        // ==== 电力交易相关 START ====
    	// 模拟每回合下发部分买/卖订单，并撮合
    	numOrders := 5 // 本轮下发5个订单
    	for i := 0; i < numOrders; i++ {
    		// 买和卖各一半
    		if i%2 == 0 {
    			// 买单：价格 450 ~ 550 ，数量 5~15
    			ob.SubmitOrder(Buy, 450+rand.Float64()*100, 5+rand.Float64()*10, fmt.Sprintf("User_%d", i))
    		} else {
    			// 卖单：价格 440~540，数量 3~12
    			ob.SubmitOrder(Sell, 440+rand.Float64()*100, 3+rand.Float64()*9, fmt.Sprintf("User_%d", i))
    		}
    	}
    	trades = ob.MatchAndClear() // 继续撮合成交
    	if len(trades) > 0 { // 本轮有成交
    		fmt.Printf("Round %d matched trades:\n", r)
    		for _, t := range trades {
    			fmt.Printf("BuyOrderID: %d, SellOrderID: %d, Price: %.2f, Quantity: %.2f\n",
    				t.BuyOrderID, t.SellOrderID, t.Price, t.Quantity)
    		}
    	}
    	// ==== 电力交易相关 END ====

        time.Sleep(200 * time.Millisecond) // 暂停 200 毫秒，控制节奏方便观察
    }

    fmt.Println("\nFinal node status:") // 输出模拟结束后节点状态
    for _, nd := range sim.nodes {
        fmt.Println(nd.String())
    }
    // ==== 电力交易相关 START ====
	fmt.Println("\n--- 交易日志 ---") // 打印最终撮合日志
	for _, logline := range ob.ListLogs() { // 遍历订单簿日志
		fmt.Println(logline)
	}
	// ==== 电力交易相关 END ====
    // <<< 只要这最后这一个大括号就能闭合 main 函数！！！
}