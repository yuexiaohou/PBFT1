// +build demo

package main

import (
	"fmt"
)

// DemoPowerTrade 演示电力交易引擎的完整功能
func main() {
	fmt.Println("========== 短期电力交易系统演示 ==========\n")

	// 创建电力交易引擎
	tradeEngine := NewPowerTradeEngine()

	// 场景 1: 基本订单提交和撮合
	fmt.Println("【场景 1】基本订单提交和撮合")
	fmt.Println("--------------------------------------")

	// 节点提交买单
	fmt.Println(">> 节点提交买单：")
	tradeEngine.SubmitOrder(OrderTypeBuy, 1.8, 150.0, 5)
	tradeEngine.SubmitOrder(OrderTypeBuy, 1.6, 200.0, 10)
	tradeEngine.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 15)

	// 节点提交卖单
	fmt.Println("\n>> 节点提交卖单：")
	tradeEngine.SubmitOrder(OrderTypeSell, 1.4, 120.0, 20)
	tradeEngine.SubmitOrder(OrderTypeSell, 1.5, 180.0, 25)
	tradeEngine.SubmitOrder(OrderTypeSell, 1.7, 100.0, 30)

	// 打印撮合前市场状态
	fmt.Println("\n>> 撮合前市场状态：")
	tradeEngine.PrintMarketStatus()

	// 执行市场出清
	fmt.Println(">> 执行市场出清...")
	clearingResult := tradeEngine.ClearMarket()

	// 打印出清结果
	fmt.Printf("\n>> 出清结果：\n")
	fmt.Printf("   出清价格: %.2f 元/kWh\n", clearingResult.ClearingPrice)
	fmt.Printf("   出清电量: %.2f kWh\n", clearingResult.ClearingVolume)
	fmt.Printf("   成交笔数: %d\n", len(clearingResult.Trades))

	// 打印出清后市场状态
	fmt.Println("\n>> 出清后市场状态：")
	tradeEngine.PrintMarketStatus()

	// 场景 2: 价格优先原则演示
	fmt.Println("\n\n【场景 2】价格优先原则演示")
	fmt.Println("--------------------------------------")
	tradeEngine2 := NewPowerTradeEngine()

	// 提交多个价格不同的买单
	fmt.Println(">> 提交多个价格不同的买单：")
	tradeEngine2.SubmitOrder(OrderTypeBuy, 1.5, 50.0, 1) // 低价
	tradeEngine2.SubmitOrder(OrderTypeBuy, 1.8, 50.0, 2) // 高价
	tradeEngine2.SubmitOrder(OrderTypeBuy, 1.6, 50.0, 3) // 中价

	// 提交卖单
	fmt.Println("\n>> 提交卖单：")
	tradeEngine2.SubmitOrder(OrderTypeSell, 1.4, 100.0, 4)

	fmt.Println("\n>> 执行撮合（应优先匹配高价买单）...")
	tradeEngine2.MatchOrders()

	fmt.Println("\n>> 成交记录（验证高价优先）：")
	for _, trade := range tradeEngine2.GetTrades() {
		fmt.Printf("   %s\n", trade.String())
	}

	// 场景 3: 部分成交和订单剩余
	fmt.Println("\n\n【场景 3】部分成交和订单剩余")
	fmt.Println("--------------------------------------")
	tradeEngine3 := NewPowerTradeEngine()

	fmt.Println(">> 提交订单：")
	tradeEngine3.SubmitOrder(OrderTypeBuy, 1.6, 200.0, 1)   // 大量买单
	tradeEngine3.SubmitOrder(OrderTypeSell, 1.4, 80.0, 2)   // 较小卖单

	fmt.Println("\n>> 撮合前：")
	tradeEngine3.PrintMarketStatus()

	fmt.Println(">> 执行撮合...")
	tradeEngine3.MatchOrders()

	fmt.Println("\n>> 撮合后（买单应有剩余）：")
	tradeEngine3.PrintMarketStatus()

	// 场景 4: 订单取消功能
	fmt.Println("\n\n【场景 4】订单取消功能")
	fmt.Println("--------------------------------------")
	tradeEngine4 := NewPowerTradeEngine()

	fmt.Println(">> 提交订单：")
	orderID := tradeEngine4.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 1)

	fmt.Println("\n>> 取消订单：")
	if tradeEngine4.CancelOrder(orderID) {
		fmt.Printf("   成功取消订单 #%d\n", orderID)
	}

	fmt.Println("\n>> 取消后市场状态：")
	tradeEngine4.PrintMarketStatus()

	// 场景 5: 复杂市场情景
	fmt.Println("\n\n【场景 5】复杂市场情景（多买多卖）")
	fmt.Println("--------------------------------------")
	tradeEngine5 := NewPowerTradeEngine()

	fmt.Println(">> 提交多个买单：")
	tradeEngine5.SubmitOrder(OrderTypeBuy, 2.0, 80.0, 1)
	tradeEngine5.SubmitOrder(OrderTypeBuy, 1.9, 100.0, 2)
	tradeEngine5.SubmitOrder(OrderTypeBuy, 1.7, 120.0, 3)
	tradeEngine5.SubmitOrder(OrderTypeBuy, 1.6, 90.0, 4)

	fmt.Println("\n>> 提交多个卖单：")
	tradeEngine5.SubmitOrder(OrderTypeSell, 1.5, 70.0, 5)
	tradeEngine5.SubmitOrder(OrderTypeSell, 1.6, 100.0, 6)
	tradeEngine5.SubmitOrder(OrderTypeSell, 1.8, 110.0, 7)
	tradeEngine5.SubmitOrder(OrderTypeSell, 1.9, 80.0, 8)

	fmt.Println("\n>> 执行市场出清...")
	result5 := tradeEngine5.ClearMarket()

	fmt.Printf("\n>> 最终出清结果：\n")
	fmt.Printf("   出清价格: %.2f 元/kWh\n", result5.ClearingPrice)
	fmt.Printf("   出清电量: %.2f kWh\n", result5.ClearingVolume)
	fmt.Printf("   总成交笔数: %d\n", len(result5.Trades))

	fmt.Println("\n>> 详细成交记录：")
	for i, trade := range result5.Trades {
		fmt.Printf("   %d. %s\n", i+1, trade.String())
	}

	fmt.Println("\n>> 最终市场状态：")
	tradeEngine5.PrintMarketStatus()

	// 展示交易日志功能
	fmt.Println("\n\n【交易日志示例】")
	fmt.Println("--------------------------------------")
	logs := tradeEngine5.GetTradeLogs()
	fmt.Printf(">> 共记录 %d 条交易日志\n", len(logs))
	fmt.Println(">> 最近 10 条日志：")
	startIdx := len(logs) - 10
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(logs); i++ {
		fmt.Printf("   %s\n", logs[i])
	}

	fmt.Println("\n========== 演示完成 ==========")
}
