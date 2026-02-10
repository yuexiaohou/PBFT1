package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

type OrderType int

const (
	Buy OrderType = iota
	Sell
)

// Order represents a trade order
type Order struct {
	ID        int
	Timestamp time.Time
	Type      OrderType
	Price     float64
	Quantity  float64
	User      string
}

// Trade represents a matched trade
type Trade struct {
	BuyOrderID  int
	SellOrderID int
	Price       float64
	Quantity    float64
	Timestamp   time.Time
}

// OrderBook is the core book for matching/tracking buy/sell orders
type OrderBook struct {
	Buys  []Order
	Sells []Order
	mu    sync.Mutex
	NextID int
	Logs   []string
}

// NewOrderBook constructs a new OrderBook
func NewOrderBook() *OrderBook {
	return &OrderBook{
		Buys:  make([]Order, 0),
		Sells: make([]Order, 0),
		Logs:  make([]string, 0),
	}
}

// Log records an event for audit
func (ob *OrderBook) Log(event string) {
	logStr := fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), event)
	ob.Logs = append(ob.Logs, logStr)
	log.Println(logStr)
}

// SubmitOrder for buyers and sellers
func (ob *OrderBook) SubmitOrder(orderType OrderType, price, quantity float64, user string) int {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	order := Order{
		ID:        ob.NextID,
		Timestamp: time.Now(),
		Type:      orderType,
		Price:     price,
		Quantity:  quantity,
		User:      user,
	}
	ob.NextID++
	if orderType == Buy {
		ob.Buys = append(ob.Buys, order)
		ob.Log(fmt.Sprintf("Buy order submitted: %+v", order))
	} else {
		ob.Sells = append(ob.Sells, order)
		ob.Log(fmt.Sprintf("Sell order submitted: %+v", order))
	}
	return order.ID
}

// MatchAndClear runs the market clearing process: match buys and sells
func (ob *OrderBook) MatchAndClear() []Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// 按价格优先/时间优先排序
	sort.Slice(ob.Buys, func(i, j int) bool {
		if ob.Buys[i].Price == ob.Buys[j].Price {
			return ob.Buys[i].Timestamp.Before(ob.Buys[j].Timestamp)
		}
		return ob.Buys[i].Price > ob.Buys[j].Price
	})
	sort.Slice(ob.Sells, func(i, j int) bool {
		if ob.Sells[i].Price == ob.Sells[j].Price {
			return ob.Sells[i].Timestamp.Before(ob.Sells[j].Timestamp)
		}
		return ob.Sells[i].Price < ob.Sells[j].Price
	})

	buyIdx, sellIdx := 0, 0
	trades := []Trade{}

	for buyIdx < len(ob.Buys) && sellIdx < len(ob.Sells) {
		buy := &ob.Buys[buyIdx]
		sell := &ob.Sells[sellIdx]
		if buy.Price >= sell.Price {
			// 可成交
			quantity := min(buy.Quantity, sell.Quantity)
			tradePrice := (buy.Price + sell.Price) / 2 // 出清价可按需调整
			trade := Trade{
				BuyOrderID:  buy.ID,
				SellOrderID: sell.ID,
				Price:       tradePrice,
				Quantity:    quantity,
				Timestamp:   time.Now(),
			}
			trades = append(trades, trade)
			ob.Log(fmt.Sprintf("Matched trade: %+v", trade))

			buy.Quantity -= quantity
			sell.Quantity -= quantity

			if buy.Quantity <= 0 {
				buyIdx++
			}
			if sell.Quantity <= 0 {
				sellIdx++
			}
		} else {
			break
		}
	}
	// 清除已成交的订单
	ob.Buys = filterActiveOrders(ob.Buys[buyIdx:])
	ob.Sells = filterActiveOrders(ob.Sells[sellIdx:])

	// 出清日志
	if len(trades) > 0 {
		ob.Log(fmt.Sprintf("Clearing %d trades at prices around %.2f", len(trades), trades[len(trades)-1].Price))
	}

	return trades
}

func filterActiveOrders(orders []Order) []Order {
	var res []Order
	for _, o := range orders {
		if o.Quantity > 0 {
			res = append(res, o)
		}
	}
	return res
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ShowOrderBook prints current buy/sell orders, for debugging or audit
func (ob *OrderBook) ShowOrderBook() {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	fmt.Println("OrderBook Status:")
	fmt.Println("Buys:")
	for _, b := range ob.Buys {
		fmt.Printf("%+v\n", b)
	}
	fmt.Println("Sells:")
	for _, s := range ob.Sells {
		fmt.Printf("%+v\n", s)
	}
}

// ListLogs returns a snapshot of logs
func (ob *OrderBook) ListLogs() []string {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	cp := make([]string, len(ob.Logs))
	copy(cp, ob.Logs)
	return cp
}