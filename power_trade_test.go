package main

import (
	"testing"
	"time"
)

// TestPowerTradeEngineBasic 测试电力交易引擎的基本功能
func TestPowerTradeEngineBasic(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 测试提交订单
	buyOrderID := engine.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 1)
	if buyOrderID != 1 {
		t.Fatalf("Expected first order ID to be 1, got %d", buyOrderID)
	}

	sellOrderID := engine.SubmitOrder(OrderTypeSell, 1.3, 80.0, 2)
	if sellOrderID != 2 {
		t.Fatalf("Expected second order ID to be 2, got %d", sellOrderID)
	}

	// 检查待撮合订单
	buyOrders, sellOrders := engine.GetPendingOrders()
	if len(buyOrders) != 1 {
		t.Fatalf("Expected 1 pending buy order, got %d", len(buyOrders))
	}
	if len(sellOrders) != 1 {
		t.Fatalf("Expected 1 pending sell order, got %d", len(sellOrders))
	}
}

// TestOrderMatching 测试订单撮合功能
func TestOrderMatching(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 提交买单：价格 1.5，数量 100
	engine.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 1)
	// 提交卖单：价格 1.3，数量 80
	engine.SubmitOrder(OrderTypeSell, 1.3, 80.0, 2)

	// 执行撮合
	trades := engine.MatchOrders()

	// 应该成交一笔
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	trade := trades[0]
	// 成交量应为 80（较小的一方）
	if trade.Volume != 80.0 {
		t.Fatalf("Expected trade volume 80.0, got %.2f", trade.Volume)
	}

	// 成交价格应为中间价：(1.5 + 1.3) / 2 = 1.4
	expectedPrice := 1.4
	if trade.Price != expectedPrice {
		t.Fatalf("Expected trade price %.2f, got %.2f", expectedPrice, trade.Price)
	}

	// 检查剩余买单
	buyOrders, sellOrders := engine.GetPendingOrders()
	if len(buyOrders) != 1 {
		t.Fatalf("Expected 1 remaining buy order, got %d", len(buyOrders))
	}
	// 买单剩余量应为 20
	if buyOrders[0].Volume != 20.0 {
		t.Fatalf("Expected remaining buy volume 20.0, got %.2f", buyOrders[0].Volume)
	}
	// 卖单应完全成交，不再有待撮合的卖单
	if len(sellOrders) != 0 {
		t.Fatalf("Expected 0 remaining sell orders, got %d", len(sellOrders))
	}
}

// TestPriorityMatching 测试价格优先和时间优先原则
func TestPriorityMatching(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 提交多个买单，价格不同
	engine.SubmitOrder(OrderTypeBuy, 1.5, 50.0, 1)  // 高价买单
	time.Sleep(1 * time.Millisecond)                // 确保时间戳不同
	engine.SubmitOrder(OrderTypeBuy, 1.3, 50.0, 2)  // 低价买单
	time.Sleep(1 * time.Millisecond)
	engine.SubmitOrder(OrderTypeSell, 1.4, 150.0, 3) // 卖单，价格在中间

	// 执行撮合
	trades := engine.MatchOrders()

	// 应该先撮合高价买单
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	trade := trades[0]
	// 应该是买单 1（价格 1.5）与卖单成交
	if trade.BuyOrderID != 1 {
		t.Fatalf("Expected buy order ID 1 to be matched first, got %d", trade.BuyOrderID)
	}
}

// TestMarketClearing 测试市场出清功能
func TestMarketClearing(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 提交多笔订单
	engine.SubmitOrder(OrderTypeBuy, 1.6, 100.0, 1)
	engine.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 2)
	engine.SubmitOrder(OrderTypeSell, 1.3, 80.0, 3)
	engine.SubmitOrder(OrderTypeSell, 1.4, 80.0, 4)

	// 执行市场出清
	result := engine.ClearMarket()

	// 应该有成交
	if len(result.Trades) == 0 {
		t.Fatal("Expected at least one trade in clearing result")
	}

	// 出清电量应大于 0
	if result.ClearingVolume <= 0 {
		t.Fatalf("Expected positive clearing volume, got %.2f", result.ClearingVolume)
	}

	// 出清价格应在合理范围内（1.3 到 1.6 之间）
	if result.ClearingPrice < 1.3 || result.ClearingPrice > 1.6 {
		t.Fatalf("Expected clearing price between 1.3 and 1.6, got %.2f", result.ClearingPrice)
	}
}

// TestNoMatching 测试无法撮合的情况（买价 < 卖价）
func TestNoMatching(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 买单价格低于卖单价格
	engine.SubmitOrder(OrderTypeBuy, 1.0, 100.0, 1)
	engine.SubmitOrder(OrderTypeSell, 1.5, 100.0, 2)

	// 执行撮合
	trades := engine.MatchOrders()

	// 不应有成交
	if len(trades) != 0 {
		t.Fatalf("Expected no trades, got %d", len(trades))
	}

	// 两个订单都应该还在待撮合列表中
	buyOrders, sellOrders := engine.GetPendingOrders()
	if len(buyOrders) != 1 || len(sellOrders) != 1 {
		t.Fatal("Expected both orders to remain pending")
	}
}

// TestMultipleMatching 测试多笔连续撮合
func TestMultipleMatching(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 提交多笔买单和卖单
	engine.SubmitOrder(OrderTypeBuy, 1.6, 50.0, 1)
	engine.SubmitOrder(OrderTypeBuy, 1.5, 60.0, 2)
	engine.SubmitOrder(OrderTypeBuy, 1.4, 70.0, 3)

	engine.SubmitOrder(OrderTypeSell, 1.2, 40.0, 4)
	engine.SubmitOrder(OrderTypeSell, 1.3, 50.0, 5)
	engine.SubmitOrder(OrderTypeSell, 1.35, 60.0, 6)

	// 执行撮合
	trades := engine.MatchOrders()

	// 应该有多笔成交
	if len(trades) < 2 {
		t.Fatalf("Expected multiple trades, got %d", len(trades))
	}

	// 检查所有成交记录
	allTrades := engine.GetTrades()
	if len(allTrades) != len(trades) {
		t.Fatalf("Expected %d total trades, got %d", len(trades), len(allTrades))
	}
}

// TestCancelOrder 测试订单取消功能
func TestCancelOrder(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 提交订单
	orderID := engine.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 1)

	// 取消订单
	success := engine.CancelOrder(orderID)
	if !success {
		t.Fatal("Expected order cancellation to succeed")
	}

	// 检查订单已从待撮合列表中移除
	buyOrders, _ := engine.GetPendingOrders()
	if len(buyOrders) != 0 {
		t.Fatalf("Expected 0 pending buy orders after cancellation, got %d", len(buyOrders))
	}

	// 尝试再次取消同一订单应失败
	success = engine.CancelOrder(orderID)
	if success {
		t.Fatal("Expected second cancellation to fail")
	}
}

// TestTradeLogging 测试交易日志记录
func TestTradeLogging(t *testing.T) {
	engine := NewPowerTradeEngine()

	// 提交订单并撮合
	engine.SubmitOrder(OrderTypeBuy, 1.5, 100.0, 1)
	engine.SubmitOrder(OrderTypeSell, 1.3, 80.0, 2)
	engine.MatchOrders()

	// 获取日志
	logs := engine.GetTradeLogs()

	// 应该至少有 3 条日志：2 条下单日志 + 1 条成交日志
	if len(logs) < 3 {
		t.Fatalf("Expected at least 3 log entries, got %d", len(logs))
	}

	// 检查日志中是否包含关键词
	foundOrderLog := false
	foundMatchLog := false
	for _, log := range logs {
		if len(log) > 7 && log[:7] == "[ORDER]" {
			foundOrderLog = true
		}
		if len(log) > 7 && log[:7] == "[MATCH]" {
			foundMatchLog = true
		}
	}

	if !foundOrderLog {
		t.Fatal("Expected to find [ORDER] log entries")
	}
	if !foundMatchLog {
		t.Fatal("Expected to find [MATCH] log entries")
	}
}
