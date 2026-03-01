package pos

import (
	"fmt"
	"os"
	"time"
)

// POS 日志管理
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

// 记录一条成交日志
func (tl *TradeLog) LogTrade(trade Trade) {
	logstr := fmt.Sprintf("[POS-TRADE] %s | Buy: %s Sell: %s Price: %.2f Qty: %.2f\n",
		time.Now().Format(time.RFC3339), trade.Buyer, trade.Seller, trade.Price, trade.Quantity)
	fmt.Print(logstr)
	if tl.file != nil {
		tl.file.WriteString(logstr)
	}
}

// Trade 结构仅示例，具体与 pos.go 协议一致
type Trade struct {
	Buyer    string
	Seller   string
	Price    float64
	Quantity float64
}