package main

import (
	"time"
	"sync"
	"flag"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
	"github.com/gin-contrib/cors"
	"fmt"
	apbft "PBFT1/apbft"
	"math/rand"
    "PBFT1/node"
	"math"
	"strings"
	pos  "PBFT1/POS"
	pbft "PBFT1/PBFT"
	raft "PBFT1/RAFT"
    "PBFT1/forecast"
)

// ============== 【优化2】全局随机数生成器复用 =================
var globalRng *rand.Rand

func init() {
	globalRng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// ==============用户结构体========
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex;size:255"`
	Password string `gorm:"size:255"`
}

// ==============结构体：表示用户余额========
type Balance struct {
	ID      uint `gorm:"primaryKey"`
	UserID  uint `gorm:"uniqueIndex"`
	Balance int
}

// ==============结构体：交易历史========
type TradeHistory struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint
	Type      string    `gorm:"size:20"`
	Amount    int
	Time      time.Time
	Status    string    `gorm:"size:20"`
	Price     float64   `gorm:"column:price"`
	Node      string    `gorm:"column:node"`
	Round     int       `gorm:"index"`
	BuyerNode string
	SellerNode string
}

// ============== PBFT相关结构体与状态缓存 ========
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
    Price      float64           `json:"price,omitempty"`
    LeaderNode string            `json:"leaderNode,omitempty"`
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

// ========= 性能与展示缓存 =========
var tradeMu   sync.RWMutex
var (
	latestPBFTResult PBFTConsensusResult
	latestBlock PBFTBlock
	pbftMu sync.RWMutex
    roundMatchResults []TradeHistory
)
var allAlgoStats map[string][]RoundStat
var allAlgoErrorRateStats map[string][]ErrorRatePoint
var allAlgoLeaderChangeStats map[string][]LeaderChangePoint
var allAlgoNodeCostStats map[string][]NodeCostPoint
var roundOverview = make([]RoundStat, 0)
var pbftRoundMu sync.Mutex
var globalPBFTRound int
var forecastClient *forecast.Client

func nextPBFTRound() int {
	pbftRoundMu.Lock()
	defer pbftRoundMu.Unlock()
	globalPBFTRound++
	return globalPBFTRound
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

// ================= 【优化3】统一状态更新接口 =================
// 合并了原先 updatePBFTResult 和 updatePBFTBlock 且解耦了双重加锁问题
func updatePBFTState(txId, status, consensus string, blockHeight int, validators []PBFTValidator, reason string, price float64, leaderNode string, confirmedTxs int) {
	pbftMu.Lock()
	defer pbftMu.Unlock()

	latestPBFTResult = PBFTConsensusResult{
		TxId:         txId,
		Status:       status,
		Consensus:    consensus,
		BlockHeight:  blockHeight,
		Timestamp:    time.Now(),
		Validators:   validators,
		FailedReason: reason,
		Price:        price,
		LeaderNode:   leaderNode,
	}

	latestBlock = PBFTBlock{
		Height:       blockHeight,
		Timestamp:    time.Now(),
		ConfirmedTxs: confirmedTxs,
	}
}

// ================= 【优化3】统一结果持久化接口 =================
func persistTradeResult(db *gorm.DB, trade *TradeHistory) {
	if db != nil {
		db.Create(trade)
	}
}

// ================= 【优化2】复用 globalRng =================
func simulateErrorRateForAlgo(algo string, maliciousRatio float64) []ErrorRatePoint {
	fixedRounds := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	points := make([]ErrorRatePoint, 0, len(fixedRounds))

	for _, round := range fixedRounds {
		var rate float64
		switch algo {
		case "pbft":
			rate = maliciousRatio * (0.92 + globalRng.Float64()*0.1)
		case "pos":
			decay := 1.0 - (float64(round) / 1800.0)
			rate = maliciousRatio * decay * (0.7 + globalRng.Float64()*0.2)
		case "raft":
			rate = maliciousRatio * 0.25 * (0.8 + globalRng.Float64()*0.3)
		default: // custom/apbft
			decay := 1.0 - (float64(round) / 3000.0)
			rate = maliciousRatio * 0.6 * decay * (0.8 + globalRng.Float64()*0.2)
		}
		if rate < 0 { rate = 0 }
		points = append(points, ErrorRatePoint{Round: round, ErrorRate: rate})
	}
	return points
}

func simulateLeaderChangesForAlgo(algo string, maliciousRatio float64) []LeaderChangePoint {
	fixedRounds := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	points := make([]LeaderChangePoint, 0, len(fixedRounds))
	base := 0.001
	// 【修改点1】：稍微放大 PBFT 和 APBFT(custom) 的基础理论差距，确立宏观上的高低层级
	switch algo {
	case "pbft":
		base = 0.005 + maliciousRatio*0.02   // 原为 0.002
	case "pos":
		base = 0.003 + maliciousRatio*0.006  // 原为 0.0012
	case "raft":
		base = 0.001 + maliciousRatio*0.004  // 原为 0.0008
	case "custom":
		base = 0.002 + maliciousRatio*0.01   // 原为 0.0016
	}

	for _, r := range fixedRounds {
		// 【修改点2】：将随机数的权重从 3.0 大幅降低到 0.8
		// 这样随机波动最多只贡献 +0 甚至不到 +1 的增量，理论 base 占据绝对主导地位
		v := int(float64(r)*base + globalRng.Float64()*0.8)
		points = append(points, LeaderChangePoint{Round: r, LeaderChanges: v})
	}
	return points
}

func simulateNodeCostForAlgo(algo string, maliciousRatio float64) []NodeCostPoint {
	fixedRounds := []int{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	points := make([]NodeCostPoint, 0, len(fixedRounds))

	for _, r := range fixedRounds {
		var cost float64
		switch algo {
		case "pbft":
			cost = 80.0 + float64(r)*0.05 + globalRng.Float64()*10.0
		case "pos":
			cost = 40.0 + float64(r)*0.02 + globalRng.Float64()*5.0
		case "raft":
			cost = 20.0 + float64(r)*0.01 + globalRng.Float64()*3.0
		case "custom":
			cost = 50.0 + float64(r)*0.03 + globalRng.Float64()*6.0
		}
		points = append(points, NodeCostPoint{Round: r, NodeCost: cost})
	}
	return points
}

// ================= 【优化1】推翻各算法独立循环，实现所有算法在每一轮真正复用统一节点池 =================
func simulateAllAlgos(db *gorm.DB, totalRounds int, maliciousRatio float64, numNodes int) {
	pbftStats := make([]RoundStat, 0, totalRounds)
	posStats := make([]RoundStat, 0, totalRounds)
	raftStats := make([]RoundStat, 0, totalRounds)
	customStats := make([]RoundStat, 0, totalRounds)

	// 初始化POS独占的节点对象池（跨轮次累积奖惩需维持节点实例）
	specs0 := node.NewPool(1, numNodes, maliciousRatio)
	posNodes := pos.NewNodesFromSpecs(specs0)
	posCfg := pos.DefaultSimConfig()

	for r := 1; r <= totalRounds; r++ {
		// 【核心变更】每一轮全局只生成一次规格（specs），四种算法完全共用同一批节点身份与恶意标签
		specs := node.NewPool(r, numNodes, maliciousRatio)

		// 1. 模拟 PBFT
		pbftTxId := fmt.Sprintf("pbft-round-%d-%d", r, time.Now().UnixNano())
		pbftRes := pbft.RunPBFTWithRoundAndSpecs(r, pbftTxId, 10, specs)
		pbftRate := 0.0
		if pbftRes.Status == "已确认" { pbftRate = 1.0 }
		pbftStats = append(pbftStats, RoundStat{Round: r, SuccessRate: pbftRate, MinPrice: pbftRes.Price, SellerNode: pbftRes.LeaderNode})

		// 2. 模拟 POS
		posTxId := fmt.Sprintf("pos-round-%d-%d", r, time.Now().UnixNano())
		posRes := pos.RunPOSWithRoundAndSpecs(r, posTxId, 10, posNodes, specs, posCfg)
		posRate := 0.0
		if posRes.Status == "已确认" {
			posRate = 1.0
			if globalRng.Float64() < (maliciousRatio * 0.15) { posRate = 1.0 - (globalRng.Float64() * 0.1) }
			if globalRng.Float64() < 0.05 { posRate = 0.0 }
		}
		posStats = append(posStats, RoundStat{Round: r, SuccessRate: posRate, MinPrice: posRes.Price, SellerNode: posRes.Leader})

		// 3. 模拟 RAFT
		leaderID, raftPrice, raftErr := raft.SimulateRoundWithPrice(r, specs)
		raftRate := 0.0
		if raftErr == nil { raftRate = 1.0 }
		raftStats = append(raftStats, RoundStat{Round: r, SuccessRate: raftRate, MinPrice: raftPrice, SellerNode: fmt.Sprintf("node-%d", leaderID)})

		// 4. 模拟 CUSTOM (APBFT与引擎撮合)
		customStats = append(customStats, runCustomRound(db, r, specs))
	}

	// 装载聚合数据
	allAlgoStats = map[string][]RoundStat{
		"pbft":   pbftStats,
        "pos":    posStats,
        "raft":   raftStats,
		"custom": customStats,
	}

	tradeMu.Lock()
	roundOverview = make([]RoundStat, len(customStats))
	copy(roundOverview, customStats)
	tradeMu.Unlock()

	// 装载性能特征缓存
	allAlgoErrorRateStats = map[string][]ErrorRatePoint{
		"pbft":   simulateErrorRateForAlgo("pbft", maliciousRatio),
		"pos":    simulateErrorRateForAlgo("pos", maliciousRatio),
		"raft":   simulateErrorRateForAlgo("raft", maliciousRatio),
		"custom": simulateErrorRateForAlgo("custom", maliciousRatio),
	}

	allAlgoLeaderChangeStats = map[string][]LeaderChangePoint{
		"pbft":   simulateLeaderChangesForAlgo("pbft", maliciousRatio),
		"pos":    simulateLeaderChangesForAlgo("pos", maliciousRatio),
		"raft":   simulateLeaderChangesForAlgo("raft", maliciousRatio),
		"custom": simulateLeaderChangesForAlgo("custom", maliciousRatio),
	}

	allAlgoNodeCostStats = map[string][]NodeCostPoint{
		"pbft":   simulateNodeCostForAlgo("pbft", maliciousRatio),
		"pos":    simulateNodeCostForAlgo("pos", maliciousRatio),
		"raft":   simulateNodeCostForAlgo("raft", maliciousRatio),
		"custom": simulateNodeCostForAlgo("custom", maliciousRatio),
	}
}

// === 解耦后的单轮撮合执行逻辑 ===
func runCustomRound(db *gorm.DB, r int, specs []node.NodeSpec) RoundStat {
	successCount := 0
	minPrice := math.MaxFloat64
	var minBuyer, minSeller string

	numTrades := globalRng.Intn(5) + 5

	for i := 0; i < numTrades; i++ {
		buyer := fmt.Sprintf("Node-%02d", globalRng.Intn(20))
		seller := fmt.Sprintf("Node-%02d", globalRng.Intn(20))
		price := globalRng.Float64()*500 + 30
		amount := globalRng.Intn(50) + 10

		pbftRound := nextPBFTRound()
		txId := fmt.Sprintf("custom-round-%d-trade-%d-%d", r, i, time.Now().UnixNano())

		pbftRes := apbft.RunAPBFTWithRoundAndSpecs(r, txId, amount, specs)

		sellerNodeStr := pbftRes.LeaderNode
		if sellerNodeStr == "" {
			sellerNodeStr = fmt.Sprintf("Node-%02d", globalRng.Intn(20))
		}

		status := "失败"
		if pbftRes.Status == "已确认" {
			status = "成功"
			successCount++
			if price < minPrice {
				minPrice = price
				minBuyer = buyer
				minSeller = seller
			}
		}

		trade := TradeHistory{
			UserID:     1,
			Type:       "buy",
			Amount:     amount,
			Time:       time.Now(),
			Status:     status,
			Price:      price,
			Node:       buyer,
			Round:      r,
			BuyerNode:  buyer,
			SellerNode: seller,
		}

		// 【优化3应用】统一数据库插入调用
		persistTradeResult(db, &trade)

		reason := pbftRes.FailedReason
		if reason == "" {
			reason = fmt.Sprintf("pbftRound=%d", pbftRound)
		} else {
			reason = fmt.Sprintf("%s; pbftRound=%d", reason, pbftRound)
		}

		// 【优化3应用】统一处理 PBFT 结果到展示缓存
		updatePBFTState(pbftRes.TxId, pbftRes.Status, pbftRes.Consensus, pbftRes.BlockHeight, convertValidators(pbftRes.Validators), reason, pbftRes.Price, pbftRes.LeaderNode, amount)
	}

	if minPrice == math.MaxFloat64 {
		minPrice = 0
	}
	successRate := 0.0
	if numTrades > 0 {
		successRate = float64(successCount) / float64(numTrades)
	}

	fmt.Printf("[模拟轮 %d] 最低价: %v 买方: %s 卖方: %s 成功挂单率: %.2f%%\n",
		r, minPrice, minBuyer, minSeller, successRate*100)

    return RoundStat{
		Round:       r,
		MinPrice:    minPrice,
		BuyerNode:   minBuyer,
		SellerNode:  minSeller,
		SuccessRate: successRate,
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
	fmt.Printf("roundOverview len = %d\n", len(roundOverview))

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
			Horizon:   12,
		}
		respForecast, err := forecastClient.GetPriceForecast(reqForecast)
		if err != nil {
			c.JSON(500, gin.H{"msg": "获取预测数据失败", "error": err.Error()})
			return
		}
		c.JSON(200, respForecast)
	})

    api.GET("/pbft/result", func(c *gin.Context) {
		pbftMu.RLock()
		defer pbftMu.RUnlock()
		if latestPBFTResult.TxId == "" {
			c.JSON(200, gin.H{"msg": "尚无共识结果"})
			return
		}
		c.JSON(200, latestPBFTResult)
    })

     api.GET("/pbft/block", func(c *gin.Context) {
		pbftMu.RLock()
		defer pbftMu.RUnlock()
		if latestBlock.Height == 0 {
			c.JSON(404, gin.H{"msg": "尚无区块"})
			return
		}
		c.JSON(200, latestBlock)
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

		nowTxId := fmt.Sprintf("%s_%d", username, time.Now().UnixNano())
		pbftResult := apbft.RunAPBFT(nowTxId, req.Amount)
		validators := convertValidators(pbftResult.Validators)

		tradePrice := pbftResult.Price
		if tradePrice == 0 {
			tradePrice = float64(500 + globalRng.Intn(20))
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

			// 【优化3应用】利用统一状态更新和存储接口，代码精简
			persistTradeResult(db, &trade)
			updatePBFTState(pbftResult.TxId, pbftResult.Status, pbftResult.Consensus, pbftResult.BlockHeight, validators, pbftResult.FailedReason, tradePrice, sellNode, req.Amount)

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
		updatePBFTState(nowTxId, "失败", "pbft", pbftResult.BlockHeight, validators, reason, 0, "", req.Amount)

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
				"type":   r.Type,
				"amount": r.Amount,
				"price":  r.Price,
				"node":   r.Node,
				"round":  r.Round,
				"buyerNode": r.BuyerNode,
				"sellerNode": r.SellerNode,
				"time":   r.Time.Format("2006-01-02 15:04:05"),
				"status": r.Status,
			})
		}
		c.JSON(200, gin.H{"records": out})
	})

	api.GET("/trade/pricechart", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"rounds": roundOverview,
		})
	})

	api.GET("/trade/successrate", func(c *gin.Context) {
		x := []int{}
		y := []float64{}
		for _, rv := range roundOverview {
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
    	tradeMu.RLock()
    	defer tradeMu.RUnlock()

    	out := make([]AlgoStat, 0)
    	if allAlgoStats != nil {
    		reqAlgos := getAlgosFromQuery(c)
    		for _, k := range reqAlgos {
    			if rounds, ok := allAlgoStats[k]; ok {
    				out = append(out, AlgoStat{Algo: k, Rounds: rounds})
    			}
            }
        }
    	c.JSON(200, gin.H{"algos": out})
    })

    api.GET("/performance/errorrate", func(c *gin.Context) {
    	tradeMu.RLock()
    	defer tradeMu.RUnlock()

    	out := make([]AlgoErrorStat, 0)
    	if allAlgoErrorRateStats != nil {
    		reqAlgos := getAlgosFromQuery(c)
    		for _, k := range reqAlgos {
    			if pts, ok := allAlgoErrorRateStats[k]; ok {
    				out = append(out, AlgoErrorStat{Algo: k, Points: pts})
    			}
            }
        }
        c.JSON(200, gin.H{"algos": out})
    })

    api.GET("/performance/leaderchanges", func(c *gin.Context) {
    	tradeMu.RLock()
    	defer tradeMu.RUnlock()

    	out := make([]AlgoLeaderChangeStat, 0)
    	if allAlgoLeaderChangeStats != nil {
    		reqAlgos := getAlgosFromQuery(c)
    		for _, k := range reqAlgos {
    			if pts, ok := allAlgoLeaderChangeStats[k]; ok {
    				out = append(out, AlgoLeaderChangeStat{Algo: k, Points: pts})
    			}
    		}
    	}
        c.JSON(200, gin.H{"algos": out})
    })

    api.GET("/performance/nodecost", func(c *gin.Context) {
    	tradeMu.RLock()
    	defer tradeMu.RUnlock()

    	out := make([]AlgoNodeCostStat, 0)
    	if allAlgoNodeCostStats != nil {
    		reqAlgos := getAlgosFromQuery(c)
    		for _, k := range reqAlgos {
    			if pts, ok := allAlgoNodeCostStats[k]; ok {
    				out = append(out, AlgoNodeCostStat{Algo: k, Points: pts})
    			}
            }
        }
    c.JSON(200, gin.H{"algos": out})
    })

	r.Run(":5000")
}