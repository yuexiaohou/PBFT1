# 短期电力交易系统

## 概述

本模块实现了基于 PBFT 共识框架的短期电力交易系统，提供完整的订单撮合、市场出清、报价处理和交易日志功能。

## 核心功能

### 1. 订单撮合
- 支持买单（BUY）和卖单（SELL）两种订单类型
- 实现价格优先、时间优先的撮合原则
- 自动处理部分成交和完全成交
- 成交后自动下架订单

### 2. 市场出清算法
- 基于供需匹配的出清机制
- 计算加权平均出清价格
- 统计总出清电量
- 输出详细的成交列表

### 3. 报价处理
- 支持买方报价提交
- 支持卖方报价提交
- 记录报价时间戳
- 关联用户/节点 ID

### 4. 交易日志
- 记录所有订单提交事件
- 记录所有撮合成交事件
- 记录市场出清结果
- 记录订单取消事件
- 支持日志查询和审计

## 数据结构

### Order（订单）
```go
type Order struct {
    ID        int         // 订单唯一标识
    Type      OrderType   // 订单类型（买/卖）
    Price     float64     // 报价（元/kWh）
    Volume    float64     // 电量（kWh）
    Timestamp time.Time   // 下单时间
    Status    OrderStatus // 订单状态
    UserID    int         // 用户/节点 ID
}
```

### Trade（成交记录）
```go
type Trade struct {
    ID          int       // 成交记录 ID
    BuyOrderID  int       // 买单 ID
    SellOrderID int       // 卖单 ID
    Price       float64   // 成交价格（元/kWh）
    Volume      float64   // 成交电量（kWh）
    Timestamp   time.Time // 成交时间
}
```

### ClearingResult（出清结果）
```go
type ClearingResult struct {
    ClearingPrice  float64   // 出清价格（元/kWh）
    ClearingVolume float64   // 出清电量（kWh）
    Timestamp      time.Time // 出清时间
    Trades         []*Trade  // 成交列表
}
```

## API 接口

### 创建交易引擎
```go
engine := NewPowerTradeEngine()
```

### 提交订单
```go
// 提交买单：愿意以 1.8 元/kWh 买入 150 kWh
orderID := engine.SubmitOrder(OrderTypeBuy, 1.8, 150.0, userID)

// 提交卖单：愿意以 1.4 元/kWh 卖出 120 kWh
orderID := engine.SubmitOrder(OrderTypeSell, 1.4, 120.0, userID)
```

### 执行订单撮合
```go
// 执行撮合，返回新产生的成交记录
trades := engine.MatchOrders()
```

### 执行市场出清
```go
// 执行市场出清（包含撮合），返回出清结果
result := engine.ClearMarket()
fmt.Printf("出清价格: %.2f 元/kWh\n", result.ClearingPrice)
fmt.Printf("出清电量: %.2f kWh\n", result.ClearingVolume)
```

### 取消订单
```go
success := engine.CancelOrder(orderID)
```

### 查询功能
```go
// 获取待撮合订单
buyOrders, sellOrders := engine.GetPendingOrders()

// 获取所有成交记录
trades := engine.GetTrades()

// 获取交易日志
logs := engine.GetTradeLogs()

// 打印市场状态
engine.PrintMarketStatus()
```

## 运行示例

### 运行完整演示
```bash
# 使用 demo 构建标签运行演示程序
go run -tags demo power_trade_demo.go power_trade.go
```

### 运行测试
```bash
# 运行所有测试
go test -v

# 运行电力交易相关测试
go test -v -run TestPowerTrade
```

### 集成到主程序
```bash
# 编译主程序（包含 PBFT 和电力交易）
go build -o pbft_sim .

# 运行主程序
./pbft_sim
```

## 测试用例

项目包含 8 个完整的测试场景：

1. **TestPowerTradeEngineBasic** - 基本订单提交测试
2. **TestOrderMatching** - 订单撮合功能测试
3. **TestPriorityMatching** - 价格和时间优先原则测试
4. **TestMarketClearing** - 市场出清算法测试
5. **TestNoMatching** - 无法撮合场景测试
6. **TestMultipleMatching** - 多笔连续撮合测试
7. **TestCancelOrder** - 订单取消功能测试
8. **TestTradeLogging** - 交易日志记录测试

## 撮合规则

### 价格优先原则
- 买单：价格高的优先撮合
- 卖单：价格低的优先撮合

### 时间优先原则
- 价格相同时，下单时间早的优先撮合

### 撮合条件
- 买单价格 >= 卖单价格时可以撮合
- 成交价格取买卖价格的中间值

### 成交处理
- 部分成交：订单剩余电量继续挂单
- 完全成交：订单自动下架

## 集成说明

### 与 PBFT 共识集成
电力交易系统可与现有 PBFT 共识机制结合：
- 每个 PBFT 节点可作为交易参与方
- 共识达成后执行交易撮合
- 利用 BLS 签名验证交易真实性

### 在 main.go 中的使用
```go
// 创建电力交易引擎
tradeEngine := NewPowerTradeEngine()

// PBFT 节点提交交易订单
for _, node := range sim.nodes {
    if node.IsActive() {
        // 根据节点属性提交买单或卖单
        tradeEngine.SubmitOrder(OrderTypeBuy, price, volume, node.ID)
    }
}

// 执行市场出清
result := tradeEngine.ClearMarket()
```

## 扩展建议

### 功能扩展
1. 支持批量订单提交
2. 实现订单修改功能
3. 添加订单有效期机制
4. 实现更复杂的出清算法（如双边拍卖）
5. 支持多时段交易

### 性能优化
1. 使用更高效的订单簿数据结构
2. 实现并发撮合
3. 添加订单索引加速查询
4. 优化大规模订单处理

### 安全增强
1. 添加订单签名验证
2. 实现防作弊机制
3. 添加交易限额控制
4. 实现撮合结果共识验证

## 许可证

本模块遵循项目整体许可证。

## 作者

集成到 yuexiaohou/PBFT1 仓库
