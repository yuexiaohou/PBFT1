package main // 声明当前文件属于 main 包，表示这是可独立运行的程序入口

import ( // 导入所需的标准库
	"flag"         // 解析命令行参数
	"fmt"          // 格式化输出
	"math/rand"    // 生成伪随机数
	"time"         // 时间相关操作
    "encoding/csv" // 读写 CSV 文件
    "os"           // 操作系统功能（如文件）
    "strconv"      // 字符串与基本类型转换
)

func main() { // main 函数为程序入口
	// 指定是否使用 blst 库（若要用 blst，需要用 -tags blst 编译且 useBlst=true）

	// ---（1）flag 和 csv 初始化部分---
    csvPath := flag.String("csv", "", "输出每轮每节点统计 CSV 路径 (optional)") // 定义命令行参数（CSV 路径）
    var csvFile *os.File    // 声明 CSV 文件句柄
    var csvWriter *csv.Writer // 声明 CSV 写入对象
    if *csvPath != "" {       // 如果传入了 csv 路径
        csvFile, _ = os.Create(*csvPath) // 创建并打开 CSV 文件
        defer csvFile.Close()             // 程序退出时关闭文件
        csvWriter = csv.NewWriter(csvFile) // 创建 CSV writer
        // 写表头
        csvWriter.Write([]string{"round","node_id","throughput","m","active","tier","isLeader"})
    }

    // ---（2）模拟器和节点初始化与运行---
	useBlst := false // 是否启用 blst 扩展（默认不启用）

	rand.Seed(time.Now().UnixNano()) // 设置随机种子为当前时间纳秒数
	numNodes := 100
    nodes := make([]*Node, numNodes)  // 存放节点的切片
	for i := 0; i< numNodes; i++ { // 创建 100个节点
		throughput := 50.0 + rand.Float64()*150.0 // 随机生成吞吐量（50~200）
		isMal := false        // 默认节点为诚实节点
		if i == 2 || i == 7 { // 2 号和 7 号节点标记为作恶节点
			isMal = true
		}
		nodes[i] = NewNode(i, throughput, isMal, useBlst) // 直接赋值（初始化node） // 创建节点并加到 nodes
	}

	sim := NewPBFTSimulator(nodes, useBlst) // 初始化 PBFT 模拟器
	sim.ComputeTiers() // 计算每个节点的分层（例如高吞吐量/低吞吐量分层）

	fmt.Println("Initial node statuses:") // 输出初始节点状态
	for _, nd := range sim.nodes {        // 遍历所有节点
		fmt.Println(nd.String())      // 输出每个节点的属性描述
	}

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

        // 记录每一轮每个节点至 CSV
        for _, nd := range sim.nodes {
            if csvWriter != nil { // 若启用 CSV 记录
                csvWriter.Write([]string{
                    strconv.Itoa(r),                   // 轮次
                    strconv.Itoa(nd.ID),               // 节点 ID
                    fmt.Sprintf("%.3f", nd.Throughput),// 吞吐量（小数点 3 位）
                    strconv.Itoa(nd.m),                // m 参数（假设某种指标）
                    strconv.FormatBool(nd.active),     // 节点是否活跃
                    fmt.Sprintf("%v", nd.Tier),        // 节点层级
                    strconv.FormatBool(sim.SelectLeader(r).ID == nd.ID), // 是否为 Leader
                })
            }
        }
        if csvWriter != nil {
            csvWriter.Flush() // 每轮结束后强制刷新文件
        }

        time.Sleep(200 * time.Millisecond) // 暂停 200 毫秒，控制节奏方便观察
    }

    fmt.Println("\nFinal node status:") // 输出模拟结束后节点状态
    for _, nd := range sim.nodes {
        fmt.Println(nd.String())
    }

    // ---（3）电力交易演示部分---
    fmt.Println("\n========== 电力交易演示 ==========")
    
    // 创建电力交易引擎
    tradeEngine := NewPowerTradeEngine()
    
    // 模拟几个节点提交电力交易订单
    fmt.Println("\n--- 提交买单和卖单 ---")
    // 节点 5 提交买单：愿意以 1.8 元/kWh 买入 150 kWh
    tradeEngine.SubmitOrder(OrderTypeBuy, 1.8, 150.0, 5)
    // 节点 10 提交买单：愿意以 1.6 元/kWh 买入 200 kWh
    tradeEngine.SubmitOrder(OrderTypeBuy, 1.6, 200.0, 10)
    // 节点 15 提交买单：愿意以 1.5 元/kWh 买入 100 kWh
    tradeEngine.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 15)
    
    // 节点 20 提交卖单：愿意以 1.4 元/kWh 卖出 120 kWh
    tradeEngine.SubmitOrder(OrderTypeSell, 1.4, 120.0, 20)
    // 节点 25 提交卖单：愿意以 1.5 元/kWh 卖出 180 kWh
    tradeEngine.SubmitOrder(OrderTypeSell, 1.5, 180.0, 25)
    // 节点 30 提交卖单：愿意以 1.7 元/kWh 卖出 100 kWh
    tradeEngine.SubmitOrder(OrderTypeSell, 1.7, 100.0, 30)
    
    // 打印当前市场状态
    fmt.Println("\n--- 撮合前市场状态 ---")
    tradeEngine.PrintMarketStatus()
    
    // 执行市场出清
    fmt.Println("\n--- 执行市场出清 ---")
    clearingResult := tradeEngine.ClearMarket()
    
    // 打印出清结果
    fmt.Printf("\n出清价格: %.2f 元/kWh\n", clearingResult.ClearingPrice)
    fmt.Printf("出清电量: %.2f kWh\n", clearingResult.ClearingVolume)
    fmt.Printf("成交笔数: %d\n", len(clearingResult.Trades))
    
    // 打印出清后市场状态
    fmt.Println("\n--- 出清后市场状态 ---")
    tradeEngine.PrintMarketStatus()
    
    // 打印所有成交记录
    fmt.Println("\n--- 成交记录详情 ---")
    allTrades := tradeEngine.GetTrades()
    for _, trade := range allTrades {
        fmt.Println(trade.String())
    }
    
    fmt.Println("\n========== 电力交易演示结束 ==========")
    // <<< 只要这最后这一个大括号就能闭合 main 函数！！！
}