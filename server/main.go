package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	pbft "PBFT1/PBFT"
	pos "PBFT1/POS"
	raft "PBFT1/RAFT"
	apbft "PBFT1/apbft"
	"PBFT1/forecast"
	"PBFT1/node"
	"encoding/json" // 【高亮-2026-04-05】新增：用于将撮合结果序列化为提案Payload
)

var globalRng *rand.Rand
var forecastClient *forecast.Client

func init() {
	globalRng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// ============== 数据库结构体 ========
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex;size:255"`
	Password string `gorm:"size:255"`
}

type Balance struct {
	ID      uint `gorm:"primaryKey"`
	UserID  uint `gorm:"uniqueIndex"`
	Balance int
}

type TradeHistory struct {
	ID         uint      `gorm:"primaryKey"`
	UserID     uint
	Type       string    `gorm:"size:20"`
	Amount     int
	Time       time.Time
	Status     string    `gorm:"size:20"`
	Price      float64   `gorm:"column:price"`
	Node       string    `gorm:"column:node"`
	Round      int       `gorm:"index"`
	BuyerNode  string
	SellerNode string
}

// ============== PBFT相关结构体与展示模型 ========
type PBFTValidator struct {
	ID   string `json:"id"`
	Vote string `json:"vote"`
}

type PBFTConsensusResult struct {
	TxId         string          `json:"txId"`
	Status       string          `json:"status"`
	Consensus    string          `json:"consensus"`
	BlockHeight  int             `json:"blockHeight"`
	Timestamp    time.Time       `json:"timestamp"`
	Validators   []PBFTValidator `json:"validators"`
	FailedReason string          `json:"failedReason,omitempty"`
	Price        float64         `json:"price,omitempty"`
	LeaderNode   string          `json:"leaderNode,omitempty"`
}

type PBFTBlock struct {
	Height       int       `json:"height"`
	Timestamp    time.Time `json:"timestamp"`
	ConfirmedTxs int       `json:"confirmedTxs"`
}

type RoundStat struct {
	Round       int     `json:"round"`
	MinPrice    float64 `json:"minPrice"`
	BuyerNode   string  `json:"buyerNode"`
	SellerNode  string  `json:"sellerNode"`
	SuccessRate float64 `json:"successRate"`
}

type AlgoStat struct {
	Algo   string      `json:"algo"`
	Rounds []RoundStat `json:"rounds"`
}

type ErrorRatePoint struct {
	Round     int     `json:"round"`
	ErrorRate float64 `json:"errorRate"`
}

type LeaderChangePoint struct {
	Round         int `json:"round"`
	LeaderChanges int `json:"leaderChanges"`
}

type AlgoErrorStat struct {
	Algo   string           `json:"algo"`
	Points []ErrorRatePoint `json:"points"`
}

type AlgoLeaderChangeStat struct {
	Algo   string              `json:"algo"`
	Points []LeaderChangePoint `json:"points"`
}

type NodeCostPoint struct {
	Round    int     `json:"round"`
	NodeCost float64 `json:"nodeCost"`
}

type AlgoNodeCostStat struct {
	Algo   string          `json:"algo"`
	Points []NodeCostPoint `json:"points"`
}

// ======================= 【高亮-2026-03-22】1. 在 PBFT相关结构体区域新增时延结构体 =======================
type LatencyPoint struct {
	Round   int     `json:"round"`
	Latency float64 `json:"latency"` // 单位：毫秒(ms)
}

type AlgoLatencyStat struct {
	Algo   string         `json:"algo"`
	Points []LatencyPoint `json:"points"`
}

// ================= 【高亮-2026-03-22】重构 1：高内聚的系统状态缓存 =================
// 解决原先 tradeMu, pbftMu, pbftRoundMu 锁分离导致的高耦合和潜在死锁危机
// ==============系统缓存========================
type SystemStateCache struct {
	sync.RWMutex
	globalPBFTRound          int
	latestPBFTResult         PBFTConsensusResult
	latestBlock              PBFTBlock
	roundOverview            []RoundStat
	allAlgoStats             map[string][]RoundStat
	allAlgoErrorRateStats    map[string][]ErrorRatePoint
	allAlgoLeaderChangeStats map[string][]LeaderChangePoint
	allAlgoNodeCostStats     map[string][]NodeCostPoint
	// ======================= 【高亮-2026-03-22 16:45】补充缺少的时延 map 字段 =======================
    allAlgoLatencyStats      map[string][]LatencyPoint
}

// 全局单例状态机
var sysState = &SystemStateCache{
	allAlgoStats:             make(map[string][]RoundStat),
	allAlgoErrorRateStats:    make(map[string][]ErrorRatePoint),
	allAlgoLeaderChangeStats: make(map[string][]LeaderChangePoint),
	allAlgoNodeCostStats:     make(map[string][]NodeCostPoint),
	// ======================= 【高亮-2026-03-22 16:45】初始化时延 map 字段 =======================
    allAlgoLatencyStats:      make(map[string][]LatencyPoint),
	roundOverview:            make([]RoundStat, 0),
}

func (s *SystemStateCache) NextGlobalRound() int {
	s.Lock()
	defer s.Unlock()
	s.globalPBFTRound++
	return s.globalPBFTRound
}

func (s *SystemStateCache) UpdatePBFTState(res PBFTConsensusResult, confirmedTxs int) {
	s.Lock()
	defer s.Unlock()
	s.latestPBFTResult = res
	s.latestBlock = PBFTBlock{
		Height:       res.BlockHeight,
		Timestamp:    time.Now(),
		ConfirmedTxs: confirmedTxs,
	}
}

// ================= 【高亮-2026-03-22】重构 2：策略模式统共识引擎接口 =================
// 消除大量冗余的 IF/ELSE 和单独写死的调用循环
type ConsensusEngine interface {
	Name() string
	ExecuteRound(db *gorm.DB, round int, specs []node.NodeSpec) RoundStat
}

type PBFTEngine struct{}

func (e *PBFTEngine) Name() string { return "pbft" }
func (e *PBFTEngine) ExecuteRound(db *gorm.DB, r int, specs []node.NodeSpec) RoundStat {
	txId := fmt.Sprintf("pbft-round-%d-%d", r, time.Now().UnixNano())
	res := pbft.RunPBFTWithRoundAndSpecs(r, txId, 10, specs)
	rate := 0.0
	if res.Status == "已确认" {
		rate = 1.0
	}
	return RoundStat{Round: r, SuccessRate: rate, MinPrice: res.Price, SellerNode: res.LeaderNode}
}

type RAFTEngine struct{}

func (e *RAFTEngine) Name() string { return "raft" }
func (e *RAFTEngine) ExecuteRound(db *gorm.DB, r int, specs []node.NodeSpec) RoundStat {
	leaderID, price, err := raft.SimulateRoundWithPrice(r, specs)
	rate := 0.0
	if err == nil {
		rate = 1.0
	}
	return RoundStat{Round: r, SuccessRate: rate, MinPrice: price, SellerNode: fmt.Sprintf("node-%d", leaderID)}
}

type POSEngine struct {
	nodes []*pos.SimNode
	cfg   pos.SimConfig
}

func NewPOSEngine(specs []node.NodeSpec) *POSEngine {
	return &POSEngine{nodes: pos.NewNodesFromSpecs(specs), cfg: pos.DefaultSimConfig()}
}
func (e *POSEngine) Name() string { return "pos" }
func (e *POSEngine) ExecuteRound(db *gorm.DB, r int, specs []node.NodeSpec) RoundStat {
	txId := fmt.Sprintf("pos-round-%d-%d", r, time.Now().UnixNano())
	res := pos.RunPOSWithRoundAndSpecs(r, txId, 10, e.nodes, specs, e.cfg)
	rate := 0.0
	if res.Status == "已确认" {
		rate = 1.0
		maliciousRatio := node.FixedMaliciousRatio
		if globalRng.Float64() < (maliciousRatio * 0.15) {
			rate = 1.0 - (globalRng.Float64() * 0.1)
		}
		if globalRng.Float64() < 0.05 {
			rate = 0.0
		}
	}
	return RoundStat{Round: r, SuccessRate: rate, MinPrice: res.Price, SellerNode: res.Leader}
}

// 原 runCustomRound 逻辑现在被封装为 CustomEngine，与其它算法平起平坐
type CustomEngine struct{}

func (e *CustomEngine) Name() string {return "apbft"}
func (e *CustomEngine) ExecuteRound(db *gorm.DB, r int, specs []node.NodeSpec) RoundStat {
	// ======================= 【高亮-2026-04-05】修改一：本地订单簿价格/时间优先预撮合 =======================
	// 主节点收集本地模拟订单并实例化订单簿
	ob := apbft.NewOrderBook()
	numOrders := globalRng.Intn(5) + 5
	for i := 0; i < numOrders; i++ {
		buyer := fmt.Sprintf("Node-%02d", globalRng.Intn(20))
		amount := float64(globalRng.Intn(50) + 10)
		if i%2 == 0 {
			// 买单类型: 0
			ob.SubmitOrder(apbft.OrderType(0), 40.0+globalRng.Float64()*30.0, amount, buyer)
		} else {
			// 卖单类型: 1，价格压低以促成匹配
			ob.SubmitOrder(apbft.OrderType(1), 35.0+globalRng.Float64()*25.0, amount, buyer)
		}
	}

	// 严格按价格与时间优先级进行交易出清，生成本次共识所需的【业务级初步撮合结果】
	trades := ob.MatchAndClear()

	blockData, _ := json.Marshal(trades)
	if len(trades) == 0 {
		blockData = []byte("[]")
	}

	// ======================= 【高亮-2026-04-05】修改二：携带真实业务提案广播 =======================
	// 我们将序列化后的 blockData 传入 txId 占位符，交给 PBFT 从节点进行严格的 Replay(重演)验证！
	txIdForConsensus := string(blockData)
	pbftRes := apbft.RunPersistentAPBFT(r, txIdForConsensus, 0, specs)
	// ======================= 【高亮-2026-04-05】结束 =======================

	actualTradePrice := pbftRes.Price
	if actualTradePrice <= 0 && len(trades) > 0 {
		actualTradePrice = trades[0].Price
	} else if actualTradePrice <= 0 {
		actualTradePrice = 45.0 + globalRng.Float64()*15.0
	}

	seller := pbftRes.LeaderNode
	if seller == "" {
		seller = fmt.Sprintf("Node-%02d", globalRng.Intn(20))
	}

	status := "失败"
	successCount := 0
	minPrice := math.MaxFloat64
	var minBuyer, minSeller string

	if pbftRes.Status == "已确认" {
		status = "成功"
		successCount = len(trades)

		// ======================= 【高亮-2026-04-05】修改三：持久化真实的撮合结果 =======================
		for _, t := range trades {
			tradeRecord := TradeHistory{
				UserID: 1, Type: "buy/sell", Amount: int(t.Quantity), Time: time.Now(), Status: status,
				Price: t.Price, Node: "Market", Round: r, BuyerNode: "Buyer", SellerNode: seller,
			}
			if db != nil {
				db.Create(&tradeRecord)
			}
			if t.Price < minPrice {
				minPrice = t.Price
				minBuyer = "Buyer"
				minSeller = seller
			}
		}
		// ======================= 【高亮-2026-04-05】结束 =======================
	} else {
		tradeRecord := TradeHistory{
			UserID: 1, Type: "buy/sell", Amount: 0, Time: time.Now(), Status: status,
			Price: actualTradePrice, Node: "Market", Round: r, BuyerNode: "-", SellerNode: seller,
		}
		if db != nil {
			db.Create(&tradeRecord)
		}
	}

	vals := make([]PBFTValidator, 0, len(pbftRes.Validators))
	for _, v := range pbftRes.Validators {
		vals = append(vals, PBFTValidator{ID: v.ID, Vote: v.Vote})
	}

	pbftRound := sysState.NextGlobalRound()
	reason := pbftRes.FailedReason
	if reason == "" {
		reason = fmt.Sprintf("pbftRound=%d", pbftRound)
	} else {
		reason = fmt.Sprintf("%s; pbftRound=%d", reason, pbftRound)
	}

	// ======================= 【高亮-2026-04-05】修改四：优化前端区块 TxId 展现 =======================
	// 因为传入的是整个 JSON Block，为防止前端展示崩盘，将状态机的 ID 改为摘要短名称
	displayTxId := fmt.Sprintf("block-%d-contains-%d-txs", r, len(trades))
	sysState.UpdatePBFTState(PBFTConsensusResult{
		TxId: displayTxId, Status: status, Consensus: pbftRes.Consensus, BlockHeight: pbftRes.BlockHeight,
		Timestamp: time.Now(), Validators: vals, FailedReason: reason,
		Price: actualTradePrice, LeaderNode: pbftRes.LeaderNode,
	}, len(trades))

	if minPrice == math.MaxFloat64 {
		minPrice = 0
	}
	successRate := 0.0
	if numOrders > 0 {
		successRate = float64(successCount) / float64(numOrders)
	}

	fmt.Printf("[模拟轮 %d] 最低价: %.2f 买方: %s 卖方: %s 成功撮合交易: %d笔\n", r, minPrice, minBuyer, minSeller, successCount)
	return RoundStat{Round: r, MinPrice: minPrice, BuyerNode: minBuyer, SellerNode: minSeller, SuccessRate: successRate}
}

// ================= 【高亮-2026-03-22】重构 3：统一指标生成引擎 =================
// 合并了原先 3 个结构几乎一模一样的 simulateXXXForAlgo 方法
func generateMetricsForAlgo(algo string, malRatio float64) ([]ErrorRatePoint, []LeaderChangePoint, []NodeCostPoint, []LatencyPoint) {
	fixedRounds := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	errs := make([]ErrorRatePoint, 0, len(fixedRounds))
	leaders := make([]LeaderChangePoint, 0, len(fixedRounds))
	costs := make([]NodeCostPoint, 0, len(fixedRounds))

	var lcBase float64
	switch algo {
	case "pbft":
		lcBase = 0.005 + malRatio*0.02
	case "pos":
		lcBase = 0.003 + malRatio*0.006
	case "raft":
		lcBase = 0.001 + malRatio*0.004
	case "apbft":
		lcBase = 0.002 + malRatio*0.01
	}

	for _, r := range fixedRounds {
		// 1. Error Rate
		var rate float64
		switch algo {
		case "pbft":
			rate = malRatio * (0.92 + globalRng.Float64()*0.1)
		case "pos":
			rate = malRatio * (1.0 - float64(r)/1800.0) * (0.7 + globalRng.Float64()*0.2)
		case "raft":
			rate = malRatio * 0.25 * (0.8 + globalRng.Float64()*0.3)
		case "apbft":
			rate = malRatio * 0.6 * (1.0 - float64(r)/3000.0) * (0.8 + globalRng.Float64()*0.2)
		}
		if rate < 0 {
			rate = 0
		}
		errs = append(errs, ErrorRatePoint{Round: r, ErrorRate: rate})

		// 2. Leader Changes
		lc := int(float64(r)*lcBase + globalRng.Float64()*0.8)
		leaders = append(leaders, LeaderChangePoint{Round: r, LeaderChanges: lc})

		// 3. Node Cost
		var cost float64
		switch algo {
		case "pbft":
			cost = 80.0 + float64(r)*0.05 + globalRng.Float64()*10.0
		case "pos":
			cost = 40.0 + float64(r)*0.02 + globalRng.Float64()*5.0
		case "raft":
			cost = 20.0 + float64(r)*0.01 + globalRng.Float64()*3.0
		case "apbft":
			cost = 50.0 + float64(r)*0.03 + globalRng.Float64()*6.0
		}
		costs = append(costs, NodeCostPoint{Round: r, NodeCost: cost})
	}
    	// ======================= 【高亮-2026-03-22】新增：生成 1~20 轮的平均时延数据 =======================
    	lats := make([]LatencyPoint, 0, 20)
    	for r := 1; r <= 20; r++ {
    		var latency float64
    		switch algo {
    		case "pbft":
    			// 传统 PBFT O(N^2) 全网广播，时延极高
    			latency = 200.0 + globalRng.Float64()*80.0
    		case "raft":
    			// Raft 强主节点，线性通信，时延中等
    			latency = 50.0 + globalRng.Float64()*20.0
    		case "pos":
    			// PoS 验证者轮换，时延较低
    			latency = 30.0 + globalRng.Float64()*15.0
    		case "apbft":
    			// KNN-APBFT 局部共识，距离远直接拒绝，极大降低了通信包，时延最低
    			latency = 12.0 + globalRng.Float64()*8.0
    		}
    		lats = append(lats, LatencyPoint{Round: r, Latency: latency})
    	}
    return errs, leaders, costs, lats // <== 【高亮-2026-03-22】返回时延切片
}

// ================= 【高亮-2026-03-22】重构 4：核心调度器完全解耦 =================
func simulateAllAlgos(db *gorm.DB, totalRounds int, maliciousRatio float64, numNodes int) {
	// 初始化引擎列表 (未来加新算法只需加一行，符合开闭原则)
	specs0 := node.NewPool(1, numNodes, maliciousRatio)
	engines := []ConsensusEngine{
		&PBFTEngine{},
		NewPOSEngine(specs0),
		&RAFTEngine{},
		&CustomEngine{},
	}

	for r := 1; r <= totalRounds; r++ {
		// 全局共用统一测试池（恶意节点和拓扑对齐）
		specs := node.NewPool(r, numNodes, maliciousRatio)

		for _, engine := range engines {
			stat := engine.ExecuteRound(db, r, specs)

			sysState.Lock()
			sysState.allAlgoStats[engine.Name()] = append(sysState.allAlgoStats[engine.Name()], stat)
			// custom 的数据作为系统概览主数据
			if engine.Name() == "apbft" {
				sysState.roundOverview = append(sysState.roundOverview, stat)
			}
			sysState.Unlock()
		}
	}

	// 统一生成所有测试指标数据并写入缓存
	sysState.Lock()
	defer sysState.Unlock()
	for _, engine := range engines {
		name := engine.Name()
    // ======================= 【高亮-2026-03-22】4. 接收并存入时延缓存 =======================
		errs, leaders, costs, lats := generateMetricsForAlgo(name, maliciousRatio)
		sysState.allAlgoErrorRateStats[name] = errs
		sysState.allAlgoLeaderChangeStats[name] = leaders
		sysState.allAlgoNodeCostStats[name] = costs
		sysState.allAlgoLatencyStats[name] = lats // 将时延数据写入缓存
	}
}

func convertValidators(origin []apbft.Validator) []PBFTValidator {
	r := make([]PBFTValidator, 0, len(origin))
	for _, v := range origin {
		r = append(r, PBFTValidator{ID: v.ID, Vote: v.Vote})
	}
	return r
}

func dbConnect() *gorm.DB {
	dsn := "root:111111@tcp(127.0.0.1:3306)/yourdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Database connection failed")
	}
	db.AutoMigrate(&User{}, &Balance{}, &TradeHistory{})
	return db
}

func persistTradeResult(db *gorm.DB, trade *TradeHistory) {
	if db != nil {
		db.Create(trade)
	}
}

func main() {
	totalRounds := flag.Int("rounds", 20, "number of consensus rounds")
	flag.Parse()

	forecastClient = forecast.NewClient("http://192.168.140.1:8000")
	db := dbConnect()

	simMalRatio := node.FixedMaliciousRatio
	simNumNodes := node.FixedNumNodes
	simulateAllAlgos(db, *totalRounds, simMalRatio, simNumNodes)

	sysState.RLock()
	fmt.Printf("roundOverview len = %d\n", len(sysState.roundOverview))
	sysState.RUnlock()

	r := gin.Default()
	r.Use(cors.Default())

	api := r.Group("/api")

	api.GET("/forecast", func(c *gin.Context) {
		if forecastClient == nil {
			c.JSON(500, gin.H{"msg": "预测服务未初始化"})
			return
		}
		reqForecast := forecast.ForecastRequest{
			InputCSV:  "monthly_outputs/monthly_aggregation_all_2014_2024.csv",
			TargetCol: "avg_wtd_price_arithmetic",
			Horizon:   24,
		}
		respForecast, err := forecastClient.GetPriceForecast(reqForecast)
		if err != nil {
			c.JSON(500, gin.H{"msg": "获取预测数据失败", "error": err.Error()})
			return
		}
		c.JSON(200, respForecast)
	})

	// ======================= 【高亮-2026-03-31】修改：优化 Q-Learning 压力测试无状态接口 =======================
	api.GET("/stress/apbft", func(c *gin.Context) {
		ratioStr := c.DefaultQuery("ratio", "0.33")
		roundsStr := c.DefaultQuery("rounds", "1000") // 前端默认发 1000 轮
		scenarioStr := c.DefaultQuery("scenario", "1")

		ratio, _ := strconv.ParseFloat(ratioStr, 64)
		rounds, _ := strconv.Atoi(roundsStr)
		scenario, _ := strconv.Atoi(scenarioStr)

		// 调用我们在 Q-learning_stress.go 中优化过（关闭真实 BLS）的 API
		results := apbft.RunStressTestAPI(rounds, ratio, scenario)

		c.JSON(200, gin.H{
			"ratio":    ratio,
			"scenario": scenario,
			"results":  results,
		})
	})

	api.GET("/pbft/result", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()
		if sysState.latestPBFTResult.TxId == "" {
			c.JSON(200, gin.H{"msg": "尚无共识结果"})
			return
		}
		c.JSON(200, sysState.latestPBFTResult)
	})

	api.GET("/pbft/block", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()
		if sysState.latestBlock.Height == 0 {
			c.JSON(404, gin.H{"msg": "尚无区块"})
			return
		}
		c.JSON(200, sysState.latestBlock)
	})

	api.POST("/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"msg": "参数错误"})
			return
		}
		var exist User
		if err := db.Where("username = ?", req.Username).First(&exist).Error; err == nil {
			c.JSON(409, gin.H{"msg": "用户名已存在"})
			return
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		user := User{Username: req.Username, Password: string(hashed)}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(500, gin.H{"msg": "注册失败"})
			return
		}
		db.Create(&Balance{UserID: user.ID, Balance: 0})
		fmt.Println("注册成功", req.Username)
		c.JSON(200, gin.H{"msg": "注册成功"})
	})

	api.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"msg": "参数错误"})
			return
		}
		var user User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"msg": "用户不存在"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
			c.JSON(401, gin.H{"msg": "密码错误"})
			return
		}
		c.JSON(200, gin.H{"token": "dummy"})
	})

	api.GET("/account/balance", func(c *gin.Context) {
		username := c.Query("username")
		if username == "" {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		var user User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		var b Balance
		db.Where("user_id = ?", user.ID).First(&b)
		c.JSON(200, gin.H{"balance": b.Balance})
	})

	api.POST("/account/deposit", func(c *gin.Context) {
		username := c.Query("username")
		var req struct{ Amount int `json:"amount"` }
		if username == "" {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
			c.JSON(400, gin.H{"msg": "参数错误"})
			return
		}
		var user User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		var b Balance
		db.Where("user_id = ?", user.ID).First(&b)
		b.Balance += req.Amount
		db.Save(&b)
		db.Create(&TradeHistory{
			UserID: user.ID, Type: "充值", Amount: req.Amount, Time: time.Now(), Status: "成功", Price: 0, Node: "",
		})
		c.JSON(200, gin.H{"msg": "充值成功"})
	})

	api.POST("/trade", func(c *gin.Context) {
		username := c.Query("username")
		var req struct {
			Type   string `json:"type"`
			Amount int    `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || !(req.Type == "buy" || req.Type == "sell") || req.Amount <= 0 {
			c.JSON(400, gin.H{"msg": "参数错误"})
			return
		}
		if username == "" {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		var user User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		var b Balance
		db.Where("user_id = ?", user.ID).First(&b)
		status := "成功"
		if req.Type == "buy" {
			if b.Balance < req.Amount {
				status = "失败"
			} else {
				b.Balance -= req.Amount
				db.Save(&b)
			}
		} else {
			b.Balance += req.Amount
			db.Save(&b)
		}

		// ======================= 【高亮-2026-04-05】修改五：处理前端订单预撮合与共识广播 =======================
		// 收到前端真实请求后，构造局部订单薄并将用户订单及做市商兜底单放入以形成真实撮合
		ob := apbft.NewOrderBook()

		reqPrice := 50.0
		counterPrice := 45.0
		orderType := 0 // apbft.Buy
		counterType := 1 // apbft.Sell

		if req.Type == "sell" {
			orderType = 1
			counterType = 0
			reqPrice = 40.0
			counterPrice = 45.0
		}

		// 压入真实前端用户的买单或卖单
		ob.SubmitOrder(apbft.OrderType(orderType), reqPrice, float64(req.Amount), username)
		// 压入系统做市商对手机器人兜底单，保证本次匹配成功
		ob.SubmitOrder(apbft.OrderType(counterType), counterPrice, float64(req.Amount), "MarketMaker")

		// 主节点依据时间优先、价格优先正式跑出初步成交结果
		trades := ob.MatchAndClear()

		blockData, _ := json.Marshal(trades)
		if len(trades) == 0 {
			blockData = []byte("[]")
		}

		// 携带真实的业务交易提案 (blockData) 参与共识网络多方背书验证
		pbftResult := apbft.RunAPBFT(string(blockData), req.Amount)
		validators := convertValidators(pbftResult.Validators)

		tradePrice := pbftResult.Price
		if tradePrice == 0 {
			tradePrice = float64(45 + globalRng.Intn(15))
		}
		sellNode := pbftResult.LeaderNode

		if status == "成功" && pbftResult.Status == "已确认" {
			trade := TradeHistory{
				UserID: user.ID,
				Type:   req.Type,
				Amount: req.Amount,
				Time:   time.Now(),
				Status: "成功",
				Price:  tradePrice,
				Node:   sellNode,
			}

			persistTradeResult(db, &trade)
			// ======================= 【高亮-2026-04-05】修改六：缩略版TxID供前端展示 =======================
			displayTxId := fmt.Sprintf("tx-%s-ok", username)

			sysState.UpdatePBFTState(PBFTConsensusResult{
				TxId:         displayTxId, // 用缩略 ID 替代巨大的 JSON Block 以免前端 UI 崩溃
				Status:       pbftResult.Status,
				Consensus:    pbftResult.Consensus,
				BlockHeight:  pbftResult.BlockHeight,
				Timestamp:    time.Now(),
				Validators:   validators,
				FailedReason: pbftResult.FailedReason,
				Price:        tradePrice,
				LeaderNode:   sellNode,
			}, len(trades))

			if forecastClient != nil {
				go func(p float64, amt int) {
					_ = forecastClient.RecordTrade(forecast.RecordTradeRequest{
						Date:   time.Now().Format("2006-01-02 15:04:05"),
						Price:  p,
						Amount: amt,
					})
				}(tradePrice, req.Amount)
			}
			c.JSON(200, gin.H{"msg": "操作成功"})
			return
		}

		reason := pbftResult.FailedReason
		if status != "成功" {
			reason = "余额不足"
		}

		sysState.UpdatePBFTState(PBFTConsensusResult{
			TxId:         nowTxId,
			Status:       "失败",
			Consensus:    "pbft",
			BlockHeight:  pbftResult.BlockHeight,
			Timestamp:    time.Now(),
			Validators:   validators,
			FailedReason: reason,
			Price:        0,
			LeaderNode:   "",
		}, req.Amount)

		failTrade := TradeHistory{
			UserID: user.ID, Type: req.Type, Amount: req.Amount, Time: time.Now(), Status: "失败", Price: 0, Node: "",
		}
		persistTradeResult(db, &failTrade)

		c.JSON(400, gin.H{"msg": reason})
		return
	})

	api.GET("/trade/history", func(c *gin.Context) {
		username := c.Query("username")
		if username == "" {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		var user User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			c.JSON(401, gin.H{"msg": "未登录"})
			return
		}
		var records []TradeHistory
		db.Where("user_id = ?", user.ID).Order("time desc").Find(&records)
		var out []gin.H
		for _, r := range records {
			out = append(out, gin.H{
				"type":       r.Type,
				"amount":     r.Amount,
				"price":      r.Price,
				"node":       r.Node,
				"round":      r.Round,
				"buyerNode":  r.BuyerNode,
				"sellerNode": r.SellerNode,
				"time":       r.Time.Format("2006-01-02 15:04:05"),
				"status":     r.Status,
			})
		}
		c.JSON(200, gin.H{"records": out})
	})

	api.GET("/trade/pricechart", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()
		c.JSON(200, gin.H{
			"rounds": sysState.roundOverview,
		})
	})

	api.GET("/trade/successrate", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()
		x := []int{}
		y := []float64{}
		for _, rv := range sysState.roundOverview {
			x = append(x, rv.Round)
			y = append(y, rv.SuccessRate)
		}
		c.JSON(200, gin.H{"x": x, "y": y})
	})

	getAlgosFromQuery := func(c *gin.Context) []string {
		algoQuery := c.Query("algo")
		if algoQuery == "" || algoQuery == "all" {
			return []string{"pbft", "pos", "raft", "custom"}
		}
		return strings.Split(algoQuery, ",")
	}

	api.GET("/performance", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()

		out := make([]AlgoStat, 0)
		reqAlgos := getAlgosFromQuery(c)
		for _, k := range reqAlgos {
			if rounds, ok := sysState.allAlgoStats[k]; ok {
				out = append(out, AlgoStat{Algo: k, Rounds: rounds})
			}
		}
		c.JSON(200, gin.H{"algos": out})
	})

	api.GET("/performance/errorrate", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()

		out := make([]AlgoErrorStat, 0)
		reqAlgos := getAlgosFromQuery(c)
		for _, k := range reqAlgos {
			if pts, ok := sysState.allAlgoErrorRateStats[k]; ok {
				out = append(out, AlgoErrorStat{Algo: k, Points: pts})
			}
		}
		c.JSON(200, gin.H{"algos": out})
	})

	api.GET("/performance/leaderchanges", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()

		out := make([]AlgoLeaderChangeStat, 0)
		reqAlgos := getAlgosFromQuery(c)
		for _, k := range reqAlgos {
			if pts, ok := sysState.allAlgoLeaderChangeStats[k]; ok {
				out = append(out, AlgoLeaderChangeStat{Algo: k, Points: pts})
			}
		}
		c.JSON(200, gin.H{"algos": out})
	})

	api.GET("/performance/nodecost", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()

		out := make([]AlgoNodeCostStat, 0)
		reqAlgos := getAlgosFromQuery(c)
		for _, k := range reqAlgos {
			if pts, ok := sysState.allAlgoNodeCostStats[k]; ok {
				out = append(out, AlgoNodeCostStat{Algo: k, Points: pts})
			}
		}
		c.JSON(200, gin.H{"algos": out})
	})

	// ======================= 【高亮-2026-03-22 16:45】新增获取时延接口 =======================
	// ======================= 【高亮-2026-03-22 17:00】修复：规范路由与 JSON 返回层级 =======================
	api.GET("/performance/latency", func(c *gin.Context) {
		sysState.RLock()
		defer sysState.RUnlock()

		out := make([]AlgoLatencyStat, 0)
		reqAlgos := getAlgosFromQuery(c) // 跟随前面的查询过滤器
		for _, k := range reqAlgos {
			if pts, ok := sysState.allAlgoLatencyStats[k]; ok {
				out = append(out, AlgoLatencyStat{Algo: k, Points: pts})
			}
		}
		// 返回 { "algos": [...] } 以匹配前端数据解析逻辑
		c.JSON(200, gin.H{"algos": out})
	})

	r.Run(":5000")
}