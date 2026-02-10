package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// OrderType 表示订单类型：买单或卖单
type OrderType int

const (
	OrderTypeBuy  OrderType = iota // 买单
	OrderTypeSell                  // 卖单
)

// String 返回订单类型的字符串表示
func (ot OrderType) String() string {
	if ot == OrderTypeBuy {
		return "BUY"
	}
	return "SELL"
}

// OrderStatus 表示订单状态
type OrderStatus int

const (
	OrderStatusPending   OrderStatus = iota // 待撮合（挂单中）
	OrderStatusMatched                      // 已成交
	OrderStatusCancelled                    // 已取消
)

// String 返回订单状态的字符串表示
func (os OrderStatus) String() string {
	switch os {
	case OrderStatusPending:
		return "PENDING"
	case OrderStatusMatched:
		return "MATCHED"
	case OrderStatusCancelled:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

// Order 表示一个电力交易订单
type Order struct {
	ID        int         // 订单唯一标识
	Type      OrderType   // 订单类型（买/卖）
	Price     float64     // 报价（元/kWh）
	Volume    float64     // 电量（kWh）
	Timestamp time.Time   // 下单时间
	Status    OrderStatus // 订单状态
	UserID    int         // 用户/节点 ID
}

// String 返回订单的可读字符串表示
func (o *Order) String() string {
	return fmt.Sprintf("Order#%d[%s, Price=%.2f, Vol=%.2f, User=%d, Status=%s, Time=%s]",
		o.ID, o.Type, o.Price, o.Volume, o.UserID, o.Status,
		o.Timestamp.Format("15:04:05.000"))
}

// Trade 表示一笔成交记录
type Trade struct {
	ID          int       // 成交记录 ID
	BuyOrderID  int       // 买单 ID
	SellOrderID int       // 卖单 ID
	Price       float64   // 成交价格（元/kWh）
	Volume      float64   // 成交电量（kWh）
	Timestamp   time.Time // 成交时间
}

// String 返回成交记录的可读字符串表示
func (t *Trade) String() string {
	return fmt.Sprintf("Trade#%d[Buy#%d, Sell#%d, Price=%.2f, Vol=%.2f, Time=%s]",
		t.ID, t.BuyOrderID, t.SellOrderID, t.Price, t.Volume,
		t.Timestamp.Format("15:04:05.000"))
}

// ClearingResult 表示市场出清结果
type ClearingResult struct {
	ClearingPrice  float64   // 出清价格（元/kWh）
	ClearingVolume float64   // 出清电量（kWh）
	Timestamp      time.Time // 出清时间
	Trades         []*Trade  // 成交列表
}

// String 返回出清结果的可读字符串表示
func (cr *ClearingResult) String() string {
	return fmt.Sprintf("Clearing[Price=%.2f, Vol=%.2f, Trades=%d, Time=%s]",
		cr.ClearingPrice, cr.ClearingVolume, len(cr.Trades),
		cr.Timestamp.Format("15:04:05.000"))
}

// PowerTradeEngine 表示电力交易撮合引擎
type PowerTradeEngine struct {
	mu           sync.Mutex    // 保护共享数据的互斥锁
	orders       map[int]*Order // 所有订单（按 ID 索引）
	buyOrders    []*Order      // 待撮合的买单列表
	sellOrders   []*Order      // 待撮合的卖单列表
	trades       []*Trade      // 成交记录列表
	nextOrderID  int           // 下一个订单 ID
	nextTradeID  int           // 下一个成交记录 ID
	tradeLogs    []string      // 交易日志
}

// NewPowerTradeEngine 创建并返回一个新的电力交易引擎实例
func NewPowerTradeEngine() *PowerTradeEngine {
	return &PowerTradeEngine{
		orders:      make(map[int]*Order),
		buyOrders:   make([]*Order, 0),
		sellOrders:  make([]*Order, 0),
		trades:      make([]*Trade, 0),
		nextOrderID: 1,
		nextTradeID: 1,
		tradeLogs:   make([]string, 0),
	}
}

// SubmitOrder 提交一个新订单（买单或卖单）
// 参数：orderType 订单类型，price 价格，volume 电量，userID 用户 ID
// 返回：订单 ID
func (pte *PowerTradeEngine) SubmitOrder(orderType OrderType, price, volume float64, userID int) int {
	pte.mu.Lock()
	defer pte.mu.Unlock()

	order := &Order{
		ID:        pte.nextOrderID,
		Type:      orderType,
		Price:     price,
		Volume:    volume,
		Timestamp: time.Now(),
		Status:    OrderStatusPending,
		UserID:    userID,
	}
	pte.nextOrderID++

	pte.orders[order.ID] = order

	// 根据订单类型加入相应的待撮合列表
	if orderType == OrderTypeBuy {
		pte.buyOrders = append(pte.buyOrders, order)
	} else {
		pte.sellOrders = append(pte.sellOrders, order)
	}

	// 记录日志
	logMsg := fmt.Sprintf("[ORDER] %s", order.String())
	pte.tradeLogs = append(pte.tradeLogs, logMsg)
	fmt.Println(logMsg)

	return order.ID
}

// MatchOrders 执行订单撮合
// 采用价格优先、时间优先的原则进行撮合
// 返回：本次撮合生成的成交记录列表
func (pte *PowerTradeEngine) MatchOrders() []*Trade {
	pte.mu.Lock()
	defer pte.mu.Unlock()

	newTrades := make([]*Trade, 0)

	// 按价格优先、时间优先对买单排序（价格从高到低，时间从早到晚）
	sort.Slice(pte.buyOrders, func(i, j int) bool {
		if pte.buyOrders[i].Price != pte.buyOrders[j].Price {
			return pte.buyOrders[i].Price > pte.buyOrders[j].Price // 价格高的优先
		}
		return pte.buyOrders[i].Timestamp.Before(pte.buyOrders[j].Timestamp) // 时间早的优先
	})

	// 按价格优先、时间优先对卖单排序（价格从低到高，时间从早到晚）
	sort.Slice(pte.sellOrders, func(i, j int) bool {
		if pte.sellOrders[i].Price != pte.sellOrders[j].Price {
			return pte.sellOrders[i].Price < pte.sellOrders[j].Price // 价格低的优先
		}
		return pte.sellOrders[i].Timestamp.Before(pte.sellOrders[j].Timestamp) // 时间早的优先
	})

	// 尝试撮合：买单价格 >= 卖单价格时可成交
	buyIdx := 0
	sellIdx := 0
	for buyIdx < len(pte.buyOrders) && sellIdx < len(pte.sellOrders) {
		buyOrder := pte.buyOrders[buyIdx]
		sellOrder := pte.sellOrders[sellIdx]

		// 检查是否可以撮合（买价 >= 卖价）
		if buyOrder.Price >= sellOrder.Price {
			// 可撮合，计算成交价格和成交量
			// 成交价格采用中间价（买价和卖价的平均值）
			tradePrice := (buyOrder.Price + sellOrder.Price) / 2.0
			tradeVolume := buyOrder.Volume
			if sellOrder.Volume < tradeVolume {
				tradeVolume = sellOrder.Volume
			}

			// 创建成交记录
			trade := &Trade{
				ID:          pte.nextTradeID,
				BuyOrderID:  buyOrder.ID,
				SellOrderID: sellOrder.ID,
				Price:       tradePrice,
				Volume:      tradeVolume,
				Timestamp:   time.Now(),
			}
			pte.nextTradeID++
			pte.trades = append(pte.trades, trade)
			newTrades = append(newTrades, trade)

			// 更新订单电量
			buyOrder.Volume -= tradeVolume
			sellOrder.Volume -= tradeVolume

			// 如果订单完全成交，标记为已成交状态
			if buyOrder.Volume <= 0.0001 { // 使用小容差处理浮点数精度问题
				buyOrder.Status = OrderStatusMatched
				buyIdx++
			}
			if sellOrder.Volume <= 0.0001 {
				sellOrder.Status = OrderStatusMatched
				sellIdx++
			}

			// 记录日志
			logMsg := fmt.Sprintf("[MATCH] %s", trade.String())
			pte.tradeLogs = append(pte.tradeLogs, logMsg)
			fmt.Println(logMsg)
		} else {
			// 当前买单价格 < 卖单价格，无法继续撮合
			break
		}
	}

	// 移除已完全成交的订单
	pte.buyOrders = pte.removeMatchedOrders(pte.buyOrders)
	pte.sellOrders = pte.removeMatchedOrders(pte.sellOrders)

	return newTrades
}

// removeMatchedOrders 从订单列表中移除已成交的订单
func (pte *PowerTradeEngine) removeMatchedOrders(orders []*Order) []*Order {
	result := make([]*Order, 0)
	for _, order := range orders {
		if order.Status == OrderStatusPending && order.Volume > 0.0001 {
			result = append(result, order)
		}
	}
	return result
}

// ClearMarket 执行市场出清，计算出清价格和出清电量
// 返回：出清结果
func (pte *PowerTradeEngine) ClearMarket() *ClearingResult {
	// 首先执行撮合
	trades := pte.MatchOrders()

	pte.mu.Lock()
	defer pte.mu.Unlock()

	result := &ClearingResult{
		Timestamp: time.Now(),
		Trades:    trades,
	}

	// 如果有成交记录，计算加权平均成交价格和总成交量
	if len(trades) > 0 {
		var totalValue float64
		var totalVolume float64
		for _, trade := range trades {
			totalValue += trade.Price * trade.Volume
			totalVolume += trade.Volume
		}
		result.ClearingPrice = totalValue / totalVolume
		result.ClearingVolume = totalVolume
	}

	// 记录日志
	logMsg := fmt.Sprintf("[CLEARING] %s", result.String())
	pte.tradeLogs = append(pte.tradeLogs, logMsg)
	fmt.Println(logMsg)

	return result
}

// GetPendingOrders 获取当前所有待撮合的订单
func (pte *PowerTradeEngine) GetPendingOrders() (buyOrders []*Order, sellOrders []*Order) {
	pte.mu.Lock()
	defer pte.mu.Unlock()

	// 返回副本，避免外部修改
	buyOrders = make([]*Order, len(pte.buyOrders))
	copy(buyOrders, pte.buyOrders)
	sellOrders = make([]*Order, len(pte.sellOrders))
	copy(sellOrders, pte.sellOrders)

	return buyOrders, sellOrders
}

// GetTrades 获取所有成交记录
func (pte *PowerTradeEngine) GetTrades() []*Trade {
	pte.mu.Lock()
	defer pte.mu.Unlock()

	// 返回副本
	trades := make([]*Trade, len(pte.trades))
	copy(trades, pte.trades)
	return trades
}

// GetTradeLogs 获取所有交易日志
func (pte *PowerTradeEngine) GetTradeLogs() []string {
	pte.mu.Lock()
	defer pte.mu.Unlock()

	// 返回副本
	logs := make([]string, len(pte.tradeLogs))
	copy(logs, pte.tradeLogs)
	return logs
}

// PrintMarketStatus 打印当前市场状态（待撮合订单、成交记录等）
func (pte *PowerTradeEngine) PrintMarketStatus() {
	pte.mu.Lock()
	defer pte.mu.Unlock()

	fmt.Println("\n========== 市场状态 ==========")
	fmt.Printf("待撮合买单数: %d\n", len(pte.buyOrders))
	for _, order := range pte.buyOrders {
		fmt.Printf("  %s\n", order.String())
	}
	fmt.Printf("待撮合卖单数: %d\n", len(pte.sellOrders))
	for _, order := range pte.sellOrders {
		fmt.Printf("  %s\n", order.String())
	}
	fmt.Printf("总成交记录数: %d\n", len(pte.trades))
	fmt.Println("==============================")
}

// CancelOrder 取消指定的订单
func (pte *PowerTradeEngine) CancelOrder(orderID int) bool {
	pte.mu.Lock()
	defer pte.mu.Unlock()

	order, exists := pte.orders[orderID]
	if !exists || order.Status != OrderStatusPending {
		return false
	}

	order.Status = OrderStatusCancelled

	// 从待撮合列表中移除
	if order.Type == OrderTypeBuy {
		pte.buyOrders = pte.removeOrder(pte.buyOrders, orderID)
	} else {
		pte.sellOrders = pte.removeOrder(pte.sellOrders, orderID)
	}

	// 记录日志
	logMsg := fmt.Sprintf("[CANCEL] Order#%d cancelled", orderID)
	pte.tradeLogs = append(pte.tradeLogs, logMsg)
	fmt.Println(logMsg)

	return true
}

// removeOrder 从订单列表中移除指定 ID 的订单
func (pte *PowerTradeEngine) removeOrder(orders []*Order, orderID int) []*Order {
	result := make([]*Order, 0)
	for _, order := range orders {
		if order.ID != orderID {
			result = append(result, order)
		}
	}
	return result
}
