package main // 声明当前文件属于 main 包，程序的入口包

import ( // 导入标准库包
	"fmt"   // 格式化输出（Println、Printf 等）
	"math/rand" // 伪随机数生成
	"time"  // 时间相关（Sleep、Now、UnixNano 等）
    "encoding/csv"
    "os"
    "strconv"
)

func main() { // main 函数：程序入口点
	// 开关：是否使用 blst（若使用 blst 请用 -tags blst 编译并把 useBlst=true）
	useBlst := false // 布尔变量，指示是否启用 blst（一个可能的外部加密库或构建标签相关的功能）

	rand.Seed(time.Now().UnixNano()) // 使用当前时间的纳秒数作为随机数种子，保证每次运行随机序列不同
	nodes := []*Node{} // 创建一个空的 Node 指针切片，用于存放模拟中的节点
	for i := 0; i < 10; i++ { // 循环创建 10 个节点，索引从 0 到 9
		throughput := 50.0 + rand.Float64()*150.0 // 随机生成一个吞吐量值：50 到 200 之间的浮点数
		isMal := false // 标记节点是否为作恶节点，默认 false（诚实节点）
		if i == 2 || i == 7 { // 如果节点索引是 2 或 7，则将其标记为作恶节点
			isMal = true
		}
		nodes = append(nodes, NewNode(i, throughput, isMal, useBlst)) // 创建节点并追加到 nodes 切片中
	}

	sim := NewPBFTSimulator(nodes, useBlst) // 使用节点列表和 useBlst 开关创建 PBFT 模拟器实例
	sim.ComputeTiers() // 计算或分配节点的层级（tier），基于吞吐量或其他指标

	fmt.Println("Initial node statuses:") // 打印初始节点状态的标题
	for _, nd := range sim.nodes { // 遍历模拟器中的所有节点
		fmt.Println(nd.String()) // 打印每个节点的字符串表示（状态、属性等）
	}

	totalRounds := 20 // 总共要运行的共识轮数
	for r := 0; r < totalRounds; r++ { // 运行多轮模拟
		if r%5 == 0 && r > 0 { // 每 5 轮（且不是第 0 轮）重新调整节点吞吐量并重算层级
			for _, nd := range sim.nodes { // 遍历所有节点
				nd.Throughput = nd.Throughput * (0.9 + rand.Float64()*0.2) // 将吞吐量乘以 0.9 到 1.1 之间的随机因子，模拟波动
			}
			sim.ComputeTiers() // 重新计算层级以反映吞吐量变化
			fmt.Println("\nRecomputed tiers:") // 打印重算层级的提示
			for _, nd := range sim.nodes { // 再次打印每个节点的新状态
				fmt.Println(nd.String())
			}
		}
		request := []byte(fmt.Sprintf("request-%d", r)) // 构造当前轮的请求内容（字节切片）
		ok := sim.RunRound(r, request) // 在模拟器中运行一轮共识，传入轮数和请求，返回是否成功
		if !ok { // 如果该轮失败（ok 为 false）
			fmt.Printf("Round %d failed\n", r) // 打印失败信息，包含轮号
		}
		time.Sleep(200 * time.Millisecond) // 每轮之间暂停 200 毫秒，避免过快运行（也使输出更可读）
	}

	fmt.Println("\nFinal node status:") // 所有轮次完成后，打印最终节点状态的标题
	for _, nd := range sim.nodes { // 遍历并打印每个节点的最终状态
		fmt.Println(nd.String())
	}
}
// 在 main 中：解析 flag -csv
csvPath := flag.String("csv", "", "输出每轮每节点统计 CSV 路径 (optional)")
...
var csvFile *os.File
var csvWriter *csv.Writer
if *csvPath != "" {
    csvFile, _ = os.Create(*csvPath)
    defer csvFile.Close()
    csvWriter = csv.NewWriter(csvFile)
    // 写表头： round,node_id,throughput,m,active,role,...
    csvWriter.Write([]string{"round","node_id","throughput","m","active","tier","isLeader"})
}
// 在每轮结束处记录每个节点
for r := 0; r < totalRounds; r++ {
    ...
    for _, nd := range sim.nodes {
        if csvWriter != nil {
            csvWriter.Write([]string{
                strconv.Itoa(r),
                strconv.Itoa(nd.ID),
                fmt.Sprintf("%.3f", nd.Throughput),
                fmt.Sprintf("%.3f", nd.M),
                strconv.FormatBool(nd.Active),
                nd.Tier.String(),
                strconv.FormatBool(sim.SelectLeader(r).ID==nd.ID),
            })
        }
    }
    if csvWriter != nil {
        csvWriter.Flush()
    }
}
