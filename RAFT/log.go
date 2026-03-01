package raft

import (
	"fmt"
	"os"
	"time"
)

// RAFT 日志管理
type TradeLog struct {
	file *os.File
}

// 新建或追加日志文件
func NewTradeLog(filepath string) (*TradeLog, error) {
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &TradeLog{file: f}, nil
}

func (tl *TradeLog) Close() {
	if tl.file != nil {
		tl.file.Close()
	}
}

// 记录一条日志（可用于事务或主节点切换）
func (tl *TradeLog) LogLeaderSwitch(nodeID int, term int) {
	logstr := fmt.Sprintf("[RAFT-LEADER] %s | Leader: Node-%02d, Term: %d\n",
		time.Now().Format(time.RFC3339), nodeID, term)
	fmt.Print(logstr)
	if tl.file != nil {
		tl.file.WriteString(logstr)
	}
}

// 记录一次交易
func (tl *TradeLog) LogTrade(trade Trade) {
	logstr := fmt.Sprintf("[RAFT-TRADE] %s | Leader: %s Price: %.2f Qty: %.2f\n",
		time.Now().Format(time.RFC3339), trade.Leader, trade.Price, trade.Quantity)
	fmt.Print(logstr)
	if tl.file != nil {
		tl.file.WriteString(logstr)
	}
}

// 示例结构，依照主算法定义
type Trade struct {
	Leader   string
	Price    float64
	Quantity float64
}