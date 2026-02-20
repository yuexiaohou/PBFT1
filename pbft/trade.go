package main // 指定当前文件属于 main 包

import (
	"fmt"     // 字符串格式化与输出
	"log"     // 日志打印
	"sort"    // 排序相关功能
	"sync"    // 并发同步锁
	"time"    // 时间相关操作
)

// OrderType 订单类型，买/卖
type OrderType int

const (
	Buy OrderType = iota // 买单
	Sell                 // 卖单
)

// Order 表示一个交易订单
type Order struct {
	ID        int         // 订单编号
	Timestamp time.Time   // 下单时间戳
	Type      OrderType   // 订单类型（买/卖）
	Price     float64     // 报价
	Quantity  float64     // 数量
	User      string      // 用户名
}

// Trade 表示一次撮合成交
type Trade struct {
	BuyOrderID  int       // 买单订单编号
	SellOrderID int       // 卖单订单编号
	Price       float64   // 成交价
	Quantity    float64   // 成交数量
	Timestamp   time.Time // 成交时间戳
}

// OrderBook 撮合簿，维护买卖订单
type OrderBook struct {
	Buys   []Order     // 买单队列
	Sells  []Order     // 卖单队列
	mu     sync.Mutex  // 并发锁，保证线程安全
	NextID int         // 下一个订单ID编号，自动递增
	Logs   []string    // 撮合和事件日志
}

// NewOrderBook 构建新的订单簿对象
func NewOrderBook() *OrderBook {
	return &OrderBook{
		Buys:  make([]Order, 0),      // 初始化买单队列
		Sells: make([]Order, 0),      // 初始化卖单队列
		Logs:  make([]string, 0),     // 初始化日志队列
	}
}

// Log 记录事件日志，方便审计和调试
func (ob *OrderBook) Log(event string) {
	logStr := fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), event) // 带时间前缀
	ob.Logs = append(ob.Logs, logStr)      // 追加到日志队列
	log.Println(logStr)                    // 同时打印到标准输出
}

// SubmitOrder 买家/卖家提交订单
func (ob *OrderBook) SubmitOrder(orderType OrderType, price, quantity float64, user string) int {
	ob.mu.Lock()                     // 锁定订单簿，防止并发写冲突
	defer ob.mu.Unlock()
	order := Order{                  // 创建新订单对象
		ID:        ob.NextID,        // 自动生成订单编号
		Timestamp: time.Now(),       // 当前时间为订单时间
		Type:      orderType,        // 类型
		Price:     price,            // 价格
		Quantity:  quantity,         // 数量
		User:      user,             // 用户名
	}
	ob.NextID++                      // 订单编号自增
	if orderType == Buy {            // 买单
		ob.Buys = append(ob.Buys, order)
		ob.Log(fmt.Sprintf("Buy order submitted: %+v", order))  // 日志记录
	} else {                        // 卖单
		ob.Sells = append(ob.Sells, order)
		ob.Log(fmt.Sprintf("Sell order submitted: %+v", order)) // 日志记录
	}
	return order.ID                  // 返回订单编号
}

// MatchAndClear 运行撮合出清，匹配买卖订单（核心撮合算法）
func (ob *OrderBook) MatchAndClear() []Trade {
	ob.mu.Lock()         // 加锁保证线程安全
	defer ob.mu.Unlock()

	// 买单按价格（高到低），价格相等时按时间升序排列
	sort.Slice(ob.Buys, func(i, j int) bool {
		if ob.Buys[i].Price == ob.Buys[j].Price {
			return ob.Buys[i].Timestamp.Before(ob.Buys[j].Timestamp)
		}
		return ob.Buys[i].Price > ob.Buys[j].Price
	})
	// 卖单按价格（低到高），价格相等时按时间升序排列
	sort.Slice(ob.Sells, func(i, j int) bool {
		if ob.Sells[i].Price == ob.Sells[j].Price {
			return ob.Sells[i].Timestamp.Before(ob.Sells[j].Timestamp)
		}
		return ob.Sells[i].Price < ob.Sells[j].Price
	})

	buyIdx, sellIdx := 0, 0         // 买单/卖单队列下标
	trades := []Trade{}             // 成交列表

	for buyIdx < len(ob.Buys) && sellIdx < len(ob.Sells) {
		buy := &ob.Buys[buyIdx]     // 当前买单
		sell := &ob.Sells[sellIdx]  // 当前卖单
		if buy.Price >= sell.Price {            // 能成交
			quantity := min(buy.Quantity, sell.Quantity)    // 撮合数量取两者最小值
			tradePrice := (buy.Price + sell.Price) / 2      // 成交价：简单平均，实际业务可调整
			trade := Trade{
				BuyOrderID:  buy.ID,
				SellOrderID: sell.ID,
				Price:       tradePrice,
				Quantity:    quantity,
				Timestamp:   time.Now(),    // 成交时间戳
			}
			trades = append(trades, trade)    // 增加到成交记录
			ob.Log(fmt.Sprintf("Matched trade: %+v", trade)) // 日志记录

			buy.Quantity -= quantity    // 扣除买单剩余量
			sell.Quantity -= quantity   // 扣除卖单剩余量

			if buy.Quantity <= 0 {      // 买单撮合完毕，下一个
				buyIdx++
			}
			if sell.Quantity <= 0 {    // 卖单撮合完毕，下一个
				sellIdx++
			}
		} else {
			break                      // 买价低于卖价，无法成交，结束撮合
		}
	}
	// 移除已成交完毕的买卖订单，仅保留剩余订单
	ob.Buys = filterActiveOrders(ob.Buys[buyIdx:])
	ob.Sells = filterActiveOrders(ob.Sells[sellIdx:])

	// 出清日志
	if len(trades) > 0 {
		ob.Log(fmt.Sprintf("Clearing %d trades at prices around %.2f", len(trades), trades[len(trades)-1].Price))
	}

	return trades               // 返回撮合成交列表
}

// filterActiveOrders 筛选剩余未成交完的订单
func filterActiveOrders(orders []Order) []Order {
	var res []Order
	for _, o := range orders {
		if o.Quantity > 0 {       // 只有剩余量才保留
			res = append(res, o)
		}
	}
	return res
}

// min 返回两个浮点数的较小值
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ShowOrderBook 打印当前买卖订单（用于调试/审计展示）
func (ob *OrderBook) ShowOrderBook() {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	fmt.Println("OrderBook Status:")
	fmt.Println("Buys:")
	for _, b := range ob.Buys {
		fmt.Printf("%+v\n", b)     // 展示买单详情
	}
	fmt.Println("Sells:")
	for _, s := range ob.Sells {
		fmt.Printf("%+v\n", s)     // 展示卖单详情
	}
}

// ListLogs 返回日志快照
func (ob *OrderBook) ListLogs() []string {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	cp := make([]string, len(ob.Logs)) // 创建副本防止外部并发污染
	copy(cp, ob.Logs)
	return cp
}