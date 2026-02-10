package main

import (
	"fmt"
	"os"
	"time"
)

// TradeLog 用于管理日志文件及打印
type TradeLog struct {
	file *os.File
}

// NewTradeLog 创建或追加一个日志文件
func NewTradeLog(filepath string) (*TradeLog, error) {
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &TradeLog{file: f}, nil
}

// Close 关闭日志文件
func (tl *TradeLog) Close() {
	if tl.file != nil {
		tl.file.Close()
	}
}

// LogTrade 输出一笔成交日志到文件及屏幕
func (tl *TradeLog) LogTrade(trade Trade) {
	logstr := fmt.Sprintf("[TRADE] %s | BuyID:%d SellID:%d Price:%.2f Qty:%.2f\n",
		time.Now().Format(time.RFC3339), trade.BuyOrderID, trade.SellOrderID, trade.Price, trade.Quantity)
	fmt.Print(logstr)
	if tl.file != nil {
		tl.file.WriteString(logstr)
	}
}

// LogAllOrderBook 输出挂单情况
func (tl *TradeLog) LogAllOrderBook(orderBooks map[int]*OrderBook) {
	ts := time.Now().Format(time.RFC3339)
	for nodeID, ob := range orderBooks {
		title := fmt.Sprintf("[%s] --- Node %d Order Book ---\n", ts, nodeID)
		tl.writeOrPrint(title)
		tl.writeOrPrint(fmt.Sprintf("-- Buy Orders --\n"))
		for _, b := range ob.Buys {
			tl.writeOrPrint(fmt.Sprintf("  ID:%d Price:%.2f Qty:%.2f By:%s\n", b.ID, b.Price, b.Quantity, b.User))
		}
		tl.writeOrPrint(fmt.Sprintf("-- Sell Orders --\n"))
		for _, s := range ob.Sells {
			tl.writeOrPrint(fmt.Sprintf("  ID:%d Price:%.2f Qty:%.2f By:%s\n", s.ID, s.Price, s.Quantity, s.User))
		}
	}
}

func (tl *TradeLog) writeOrPrint(msg string) {
	fmt.Print(msg)
	if tl.file != nil {
		tl.file.WriteString(msg)
	}
}

// 辅助：输出单节点订单簿
func (tl *TradeLog) LogSingleOrderBook(nodeID int, ob *OrderBook) {
	ts := time.Now().Format(time.RFC3339)
	title := fmt.Sprintf("[%s] --- Node %d Order Book ---\n", ts, nodeID)
	tl.writeOrPrint(title)
	tl.writeOrPrint(fmt.Sprintf("-- Buy Orders --\n"))
	for _, b := range ob.Buys {
		tl.writeOrPrint(fmt.Sprintf("  ID:%d Price:%.2f Qty:%.2f By:%s\n", b.ID, b.Price, b.Quantity, b.User))
	}
	tl.writeOrPrint(fmt.Sprintf("-- Sell Orders --\n"))
	for _, s := range ob.Sells {
		tl.writeOrPrint(fmt.Sprintf("  ID:%d Price:%.2f Qty:%.2f By:%s\n", s.ID, s.Price, s.Quantity, s.User))
	}
}
