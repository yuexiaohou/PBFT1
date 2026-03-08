package main

import (
	"time"
	"sync" // ========= 高亮: 新增
	"flag" // ========= 高亮: 新增参数命令支持（本次变动） ==========
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
	"github.com/gin-contrib/cors"
	"fmt"          // 格式化输出
	apbft "PBFT1/apbft"
	// ===== 高亮-2026-03-01：新增主仿真入口导入，rand包和math包的作用是用于计算和绘图=====
	"math/rand"
	// ======================= 【高亮-2026-03-08】新增：引入通用节点池（round 固定恶意节点集合） =======================
    "PBFT1/node"
	"math"
	pos  "PBFT1/POS"
	pbft "PBFT1/PBFT"
)

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
	Round     int       `gorm:"index"` // <== 本次高亮（模拟轮次）
	BuyerNode string    // <== 本次高亮（买方模拟节点）
	SellerNode string   // <== 本次高亮（卖方模拟节点
}

// ============== PBFT相关结构体与状态缓存 ======== 高亮新增 START ==========
type PBFTValidator struct {
	ID   string `json:"id"`
	Vote string `json:"vote"`
}

// ==============结构体：共识结果========
type PBFTConsensusResult struct {
	TxId         string          `json:"txId"`
	Status       string          `json:"status"`
	Consensus    string          `json:"consensus"`
	BlockHeight  int             `json:"blockHeight"`
	Timestamp    time.Time       `json:"timestamp"`
	Validators   []PBFTValidator `json:"validators"`
	FailedReason string          `json:"failedReason,omitempty"`
	// ======================= PBFT共识结果也可加入价格与节点字段 =======================
    Price      float64          `json:"price,omitempty"`   // <== 可选用于 PBFTResult 前端展示
    LeaderNode string           `json:"leaderNode,omitempty"`
}

// ==============结构体：区块========
type PBFTBlock struct {
	Height       int       `json:"height"`
	Timestamp    time.Time `json:"timestamp"`
	ConfirmedTxs int       `json:"confirmedTxs"`
}

// === 2026-03-03 新增: 轮次统计结构定义 ===
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

// ======================= 2026-03-04 新增：性能特性扩展结构 BEGIN =======================
// 错误节点使用率（0~1）采样点
type ErrorRatePoint struct {
	Round     int     `json:"round"`
	ErrorRate float64 `json:"errorRate"`
}

// 主节点转换次数采样点
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

// ========= 性能与展示缓存 =========
var tradeMu   sync.RWMutex // ==========保护全局统计（并发） ==========
var (
	latestPBFTResult PBFTConsensusResult
	latestBlock PBFTBlock
	pbftMu sync.RWMutex
	// ==========用于存放每轮撮合结果的全局变量 ==========
    roundMatchResults []TradeHistory
)
// ======================= 后端代码，绘图的相关代码=======================
var allAlgoStats map[string][]RoundStat
// ======================= 2026-03-04 新增：性能特性扩展缓存 BEGIN =======================
var allAlgoErrorRateStats map[string][]ErrorRatePoint
var allAlgoLeaderChangeStats map[string][]LeaderChangePoint
// =======================声明全局变量共识轮次，与下方仿真函数中的arr部分变量不矛盾=======================
var roundOverview = make([]RoundStat, 0)
// ======================= 高亮-2026-03-07: 做法A - 全局PBFT round 计数器（带锁） =======================
var pbftRoundMu sync.Mutex
var globalPBFTRound int

// ======================= 高亮-2026-03-07: 做法A - 每次取round都封装成函数，避免遗漏加锁 =======================
func nextPBFTRound() int {
	pbftRoundMu.Lock()
	defer pbftRoundMu.Unlock()
	globalPBFTRound++
	return globalPBFTRound
}
// 转换 pbft.Result.Validators 到页面需要的形式，将apbft的共识结果传入到前端
// ======================= 【高亮-2026-03-07】修改：pbft.Validator -> apbft.Validator =======================
func convertValidators(origin []apbft.Validator) []PBFTValidator {
	// ======================= 【高亮-2026-03-07】END =======================
	r := make([]PBFTValidator, 0, len(origin))
	for _, v := range origin {
		r = append(r, PBFTValidator{ID: v.ID, Vote: v.Vote})
	}
	return r
}

// 数据库连接
func dbConnect() *gorm.DB {
	dsn := "root:111111@tcp(127.0.0.1:3306)/yourdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Database connection failed")
	}
	db.AutoMigrate(&User{}, &Balance{}, &TradeHistory{})
	return db
}

// ========== PBFT状态更新函数 ========= 新增 START =========
func updatePBFTResult(txId string, status string, consensus string, blockHeight int, validators []PBFTValidator, reason string) {
	pbftMu.Lock()
	defer pbftMu.Unlock()
	latestPBFTResult = PBFTConsensusResult{
		TxId:        txId,
		Status:      status,
		Consensus:   consensus,
		BlockHeight: blockHeight,
		Timestamp:   time.Now(),
		Validators:  validators,
		FailedReason: reason,
	}
}

func updatePBFTBlock(height int, confirmedTxs int) {
	pbftMu.Lock()
	defer pbftMu.Unlock()
	latestBlock = PBFTBlock{
		Height:       height,
		Timestamp:    time.Now(),
		ConfirmedTxs: confirmedTxs,
	}
}

// ======================= 2026-03-04 新增：错误节点使用率模拟 BEGIN =======================
func simulateErrorRateForAlgo(algo string, maliciousRatio float64) []ErrorRatePoint {
	rounds := []int{100, 200, 300, 400,500, 600, 700, 800, 900, 1000}
	points := make([]ErrorRatePoint, 0, len(rounds))

	// 基线（按算法区分），并与 maliciousRatio 相关
	base := 0.10
	switch algo {
	case "pbft":
		base = 0.08 + maliciousRatio*0.35
	case "pos":
		base = 0.05 + maliciousRatio*0.20
	case "raft":
		base = 0.03 + maliciousRatio*0.15
	case "custom":
		base = 0.10 + maliciousRatio*0.25
	default:
		base = 0.08 + maliciousRatio*0.20
	}

	for _, r := range rounds {
		// 随轮数增大，假设系统逐步抑制错误节点使用率（示例）
		decay := 1.0 - (math.Log10(float64(r)) / 10.0) // r=10 -> ~0.9, 10000 -> ~0.6 左右
		v := base * decay

		// 小噪声
		v = v + (rand.Float64()-0.5)*0.03

		// clamp [0,1]
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}

		points = append(points, ErrorRatePoint{Round: r, ErrorRate: v})
	}
	return points
}

// ======================= 2026-03-04 新增：主节点转换次数模拟 BEGIN =======================
func simulateLeaderChangesForAlgo(algo string, maliciousRatio float64) []LeaderChangePoint {
	rounds := []int{100, 200, 300, 400,500, 600, 700, 800, 900, 1000}
	points := make([]LeaderChangePoint, 0, len(rounds))

	// 基线（按算法区分），并与 maliciousRatio 相关
	base := 0.001
	switch algo {
	case "pbft":
		base = 0.002 + maliciousRatio*0.020
	case "pos":
		base = 0.0012 + maliciousRatio*0.006
	case "raft":
		base = 0.0008 + maliciousRatio*0.004
	case "custom":
		base = 0.0016 + maliciousRatio*0.010
	default:
		base = 0.001
	}

	for _, r := range rounds {
		v := int(float64(r)*base + rand.Float64()*3.0)
		if v < 0 {
			v = 0
		}
		points = append(points, LeaderChangePoint{Round: r, LeaderChanges: v})
	}
	return points
}

// ======================= 2026-03-04 修正：simulateAllAlgos 增加 maliciousRatio 参数 BEGIN =======================
func simulateAllAlgos(db *gorm.DB, totalRounds int, maliciousRatio float64, numNodes int) {
	allAlgoStats = map[string][]RoundStat{
		"pbft":   simulatePBFT(db, totalRounds, maliciousRatio, numNodes),
		"pos":    simulatePOS(db, totalRounds, maliciousRatio),
		"raft":   simulateRAFT(db, totalRounds, maliciousRatio),
		"custom": simulateCUSTOM(db, totalRounds, maliciousRatio, numNodes),
	}, maliciousRatio

	// ===== 高亮新增：缓存错误节点使用率（round=100/1000）=====
	allAlgoErrorRateStats = map[string][]ErrorRatePoint{
		"pbft":   simulateErrorRateForAlgo("pbft", maliciousRatio),
		"pos":    simulateErrorRateForAlgo("pos", maliciousRatio),
		"raft":   simulateErrorRateForAlgo("raft", maliciousRatio),
		"custom": simulateErrorRateForAlgo("custom", maliciousRatio),
	}

	// ===== 高亮新增：缓存主节点转换次数（round=100/1000）=====
	allAlgoLeaderChangeStats = map[string][]LeaderChangePoint{
		"pbft":   simulateLeaderChangesForAlgo("pbft", maliciousRatio),
		"pos":    simulateLeaderChangesForAlgo("pos", maliciousRatio),
		"raft":   simulateLeaderChangesForAlgo("raft", maliciousRatio),
		"custom": simulateLeaderChangesForAlgo("custom", maliciousRatio),
	}
}

// 你实际业务算法可换为真实聚合，只要最终返回[]RoundStat即可
// ==== 2026-03-04 高亮: PBFT 节点池参与业务 ====
func simulatePBFT(db *gorm.DB, totalRounds int, maliciousRatio float64, numNodes int) []RoundStat {
	// ======================= 【高亮-2026-03-08】修改：PBFT 仿真改为使用 nodepool.go 的节点池（按 round 固定恶意集合） =======================
	arr := make([]RoundStat, 0, totalRounds)
	// 原代码会读 users，但并未真正用于 PBFT 共识；保留读取不影响功能（也可删除）
	var users []User
	db.Find(&users)
	for round := 1; round <= totalRounds; round++ {
		// ======================= 【高亮-2026-03-08】新增：每轮用 node.NewPool 生成固定恶意节点集合 =======================
		specs := node.NewPool(round, numNodes, maliciousRatio)
		// txId / amount 仅用于模拟
		txId := fmt.Sprintf("pbft-round-%d-%d", round, time.Now().UnixNano())
		amount := rand.Intn(50) + 10
		// 使用 /PBFT/pbft.go的RunPBFTWithRoundAndSpecs(round, txId, amount, specs)
		// 注意：这要求你在 PBFT/pbft.go 中新增/改造该函数，使其能接收节点池 specs，
		// 否则这里仍会 undefined。函数签名建议如下：
		res := pbft.RunPBFTWithRoundAndSpecs(round, txId, amount, specs)
		rate := 0.0
		if res.Status == "已确认" {
			rate = 1.0
		}

		arr = append(arr, RoundStat{
			Round:       round,
			SuccessRate: rate,
			MinPrice:    res.Price,
			BuyerNode:   "",
			SellerNode:  res.LeaderNode,
		})
	}
	return arr
}

// ==== 2026-03-06 修正: POS 使用真实 stake 抽取 + 奖惩仿真 ====
func simulatePOS(db *gorm.DB, totalRounds int) []RoundStat {
	// 这里不依赖数据库 users/balance，避免你数据库数据导致 successRate 恒为 1
	// 如果你想把 stake 映射到 Balance，也可以后续再扩展
	cfg := pos.DefaultSimConfig()
	summaries, _ := pos.RunSimulator(totalRounds, cfg)

	arr := make([]RoundStat, 0, totalRounds)
	successSoFar := 0

	for _, s := range summaries {
		if s.Success {
			successSoFar++
		}
		// 定义 successRate 为 “截至当前轮的累计成功率”，曲线更平滑
		rate := float64(successSoFar) / float64(s.Round)

		arr = append(arr, RoundStat{
			Round:       s.Round,
			SuccessRate: rate, // 0~1
			MinPrice:    0,
			BuyerNode:   "",
			SellerNode:  "",
		})
	}
	return arr
}

// ==== 2026-03-04 新增: RAFT 节点池参与业务 ====
func simulateRAFT(db *gorm.DB, totalRounds int) []RoundStat {
	var arr []RoundStat
	var users []User
	db.Find(&users) // === 高亮 ===
	for i := 1; i <= totalRounds; i++ {
		active := 0
		for _, u := range users {
			var b Balance
			db.Where("user_id = ?", u.ID).First(&b) // === 高亮 ===
			if b.Balance > 20 { active++ }
		}
		rate := 0.5 + rand.Float64()*0.3
		if len(users) > 0 {
			rate = float64(active) / float64(len(users)) // === 高亮 ===
		}
		arr = append(arr, RoundStat{Round: i, SuccessRate: rate})
	}
	return arr
}

// === 2026-03-03 新增: 撮合仿真核心逻辑示例 ===
func simulateCUSTOM(db *gorm.DB, totalRounds int, maliciousRatio float64) []RoundStat {
	maliciousRatio = node.FixedMaliciousRatio
    numNodes = node.FixedNumNodes
    var arr []RoundStat
    var users []User
    db.Find(&users)
	for r := 1; r <= totalRounds; r++ {
	    successCount := 0
    	minPrice := math.MaxFloat64
    	var minBuyer, minSeller string
    	// 同一轮所有交易共享同一批 specs（同一 round 恶意集合稳定）
    	specs := node.NewPool(r, numNodes, maliciousRatio)
        numTrades := rand.Intn(5) + 5 // 每轮随机产生5~9个交易

		for i := 0; i < numTrades; i++ {
			buyer := fmt.Sprintf("Node-%02d", rand.Intn(20))
			seller := fmt.Sprintf("Node-%02d", rand.Intn(20))
			price := rand.Float64()*500 + 30 // 随机价格30~530
			amount := rand.Intn(50) + 10
			// ======================= 【高亮-2026-03-07】做法A：全局PBFT round 计数器（带锁），每次调用+1 =======================
			pbftRound := nextPBFTRound()
			// ======================= 【高亮-本次修改】用 PBFT 共识结果决定交易是否成功 =======================
			// 为每笔 trade 生成一个 txId，然后用 apbft.RunAPBFTWithRoundAndSpecs来判定是否“已确认”
			txId := fmt.Sprintf("custom-round-%d-trade-%d-%d", r, i, time.Now().UnixNano())
			// ======================= 【高亮-2026-03-07】方案A：PBFT Round = 撮合轮 r（严格一致） =======================
			pbftRes := apbft.RunAPBFTWithRoundAndSpecs(r, txId, amount, specs)

			status := "失败"
			if pbftRes.Status == "已确认" {
				status = "成功"
				successCount++

				// 注意：minPrice 仍使用撮合生成的 price（你也可以改成 pbftRes.Price）
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
			db.Create(&trade)

			// 作用是将 apbft.RunAPBFT(txId, amount) 得到的最新结果”写进全局缓存GET /api/pbft/result、GET /api/pbft/block
			// ======================= 【高亮-本次修改】可选：同步 PBFTResult 到全局缓存 =======================
			validators := convertValidators(pbftRes.Validators)
			// ======================= 【高亮-2026-03-07】把 pbftRound 写入 FailedReason 便于前端/日志定位 =======================
			reason := pbftRes.FailedReason
			if reason == "" {
				reason = fmt.Sprintf("pbftRound=%d", pbftRound)
			} else {
				reason = fmt.Sprintf("%s; pbftRound=%d", reason, pbftRound)
			}// === 2026-03-03 新增: 撮合仿真核心逻辑示例 ===
			// 注意：这里要传 reason，而不是 pbftRes.FailedReason
			updatePBFTResult(pbftRes.TxId, pbftRes.Status, pbftRes.Consensus, pbftRes.BlockHeight, validators, reason)
			// 补充 price / leader 到 latestPBFTResult，便于前端展示
            pbftMu.Lock()
            latestPBFTResult.Price = pbftRes.Price
            latestPBFTResult.LeaderNode = pbftRes.LeaderNode
            pbftMu.Unlock()
			updatePBFTBlock(pbftRes.BlockHeight, amount)
		}

		if minPrice == math.MaxFloat64 {
			minPrice = 0
		}
		successRate := 0.0
		if numTrades > 0 {
			successRate = float64(successCount) / float64(numTrades)
		}

		arr = append(arr, RoundStat{
			Round:       r,
			MinPrice:    minPrice,
			BuyerNode:   minBuyer,
			SellerNode:  minSeller,
			SuccessRate: successRate,
		})

		// ======================= 2026-03-06 新增：PBFT 同步到撮合总览（可选）BEGIN =======================
		tradeMu.Lock()
		roundOverview = make([]RoundStat, len(arr))
		copy(roundOverview, arr)
		tradeMu.Unlock()

		fmt.Printf("[模拟轮 %d] 最低价: %v 买方: %s 卖方: %s 成功挂单率: %.2f%%\n",
			r, minPrice, minBuyer, minSeller, successRate*100)
	}
	return arr
}

func main() {
	// =========2026-03-01: 命令行参数配置 ==========
	totalRounds := flag.Int("rounds", 20, "number of consensus rounds")
	flag.Parse()
	// =========调用数据库==========
	_ = node.FixedNumNodes// ========= 高亮-2026-03-07: 取消并行 RunAPBFTSimulator 后暂不使用（保留参数兼容） ==========
    _ = node.FixedMaliciousRatio// ========= 高亮-2026-03-07: 取消并行 RunAPBFTSimulator 后暂不使用（保留参数兼容） ==========
	db := dbConnect()
    // === 2026-03-03 新增: 启动时自动模拟撮合轮次（正式项目应由业务流程驱动） ===
    // ==== 2026-03-04 高亮：调用聚合填充所有算法 simulateCUSTOM() 内部已调用 PBFT1，且不再与 RunAPBFTSimulator 并行====
	simulateAllAlgos(db,30,*simMalRatio)
	fmt.Printf("roundOverview len = %d\n", len(roundOverview) )// === 2026-03-03 高亮调试 ===
	for _, rv := range roundOverview {
            fmt.Printf("round stat: %+v\n", rv)
        }

	//==调用web界面
	r := gin.Default()

	// 允许前端跨域
	r.Use(cors.Default())

	api := r.Group("/api")

	// ====== 高亮：新增 PBFT result 接口 BEGIN ======
    api.GET("/pbft/result", func(c *gin.Context) {
	pbftMu.RLock()
	defer pbftMu.RUnlock()
	if latestPBFTResult.TxId == "" {
		c.JSON(200, gin.H{"msg": "尚无共识结果"})
		return
	}
	c.JSON(200, latestPBFTResult)
    })

    // ====== 高亮：新增 PBFT block 接口 BEGIN ======
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
		// 只做演示, token直接用"dummy"（生产环境请用JWT）
		c.JSON(200, gin.H{"token": "dummy"})
	})

	// 登录状态校验可用中间件实现，这里简化跳过
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
		// 前端应带username, 实际建议token里带id
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
		// 添加充值历史
		db.Create(&TradeHistory{
			UserID: user.ID, Type: "充值", Amount: req.Amount, Time: time.Now(), Status: "成功",
			// =======================充值可无价格与节点 =======================
            Price: 0, Node: "",
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
        // 2) 每次交易生成 txId，跑PBFT共识
		nowTxId := fmt.Sprintf("%s_%d", username, time.Now().UnixNano())
		pbftResult := apbft.RunAPBFT(nowTxId, req.Amount)
		validators := convertValidators(pbftResult.Validators)
		// 3) 价格与节点（leader）
		tradePrice := pbftResult.Price
		if tradePrice == 0 {
			tradePrice = float64(500 + rand.Intn(20)) // fallback（当PBFT未返回价格）
		}
		sellNode := pbftResult.LeaderNode

		// 4) 方案B：以 “业务成功 && PBFT已确认” 才认为最终成功，并只写一次TradeHistory、只返回一次JSON
		if status == "成功" && pbftResult.Status == "已确认" {
			// 写业务交易记录（只写一次，避免你原代码里重复 db.Create）
			db.Create(&TradeHistory{
				UserID: user.ID,
				Type:   req.Type,
				Amount: req.Amount,
				Time:   time.Now(),
				Status: "成功",
				Price:  tradePrice,
				Node:   sellNode,
			})

			// 更新PBFT缓存（补充Price/LeaderNode给前端）
			updatePBFTResult(pbftResult.TxId, pbftResult.Status, pbftResult.Consensus, pbftResult.BlockHeight, validators, pbftResult.FailedReason)

			// 高亮-2026-03-06：把价格和LeaderNode同步进latestPBFTResult，便于 /api/pbft/result 展示
			pbftMu.Lock()
			latestPBFTResult.Price = tradePrice
			latestPBFTResult.LeaderNode = sellNode
			pbftMu.Unlock()

			updatePBFTBlock(pbftResult.BlockHeight, req.Amount)

			c.JSON(200, gin.H{"msg": "操作成功"})
			return
		}

		// 5) 失败分支：统一失败原因与PBFT缓存写入（只返回一次）
		reason := pbftResult.FailedReason
		if status != "成功" {
			reason = "余额不足"
		}
		updatePBFTResult(nowTxId, "失败", "pbft", pbftResult.BlockHeight, validators, reason)

		// 高亮-2026-03-06：失败时也可选择记录失败流水（如你不想记录可删除此段）
		db.Create(&TradeHistory{
			UserID: user.ID,
			Type:   req.Type,
			Amount: req.Amount,
			Time:   time.Now(),
			Status: "失败",
			Price:  0,
			Node:   "",
		})

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
		// 转换格式给前端
		var out []gin.H
		for _, r := range records {
			out = append(out, gin.H{
				"type":   r.Type,
				"amount": r.Amount,
				"price":  r.Price,
				"node":   r.Node,
				"round":  r.Round,       // =====高亮：轮次
				"buyerNode": r.BuyerNode, // =====高亮：买节点
				"sellerNode": r.SellerNode,// =====高亮：卖节点
				"time":   r.Time.Format("2006-01-02 15:04:05"),
				"status": r.Status,
			})
		}
		c.JSON(200, gin.H{"records": out})
	})

	// ========== 撮合图表接口1：最低价格随轮次变化 ==========
	api.GET("/trade/pricechart", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"rounds": roundOverview,
		})
	})
	// ========== 撮合图表接口2：每轮撮合率随轮次变化 ==========
	api.GET("/trade/successrate", func(c *gin.Context) {
		x := []int{}
		y := []float64{}
		for _, rv := range roundOverview {
			x = append(x, rv.Round)
			y = append(y, rv.SuccessRate)
		}
		c.JSON(200, gin.H{"x": x, "y": y})
	})

	// ======================= 2026-03-04 新增：性能特性接口 BEGIN =======================
    // GET /api/performance
    // - 不带 algo 或 algo=all：返回 { algos: [{ algo, rounds:[{round,successRate,...}]}] }
    // - 带 algo=pbft|pos|raft|custom：返回 { algo, rounds:[...] }
    api.GET("/performance", func(c *gin.Context) {
    	tradeMu.RLock()
    	defer tradeMu.RUnlock()

    	if allAlgoStats == nil {
    		c.JSON(200, gin.H{"algos": []AlgoStat{}})
    		return
    	}

    	algo := c.Query("algo")
    	if algo != "" && algo != "all" {
    		rounds, ok := allAlgoStats[algo]
    		if !ok {
    			c.JSON(404, gin.H{"msg": "unknown algo", "algo": algo})
    			return
    		}
    		c.JSON(200, gin.H{"algo": algo, "rounds": rounds})
    		return
    	}

    	out := make([]AlgoStat, 0, 4)
    	order := []string{"pbft", "pos", "raft", "custom"}
    	for _, k := range order {
    		if rounds, ok := allAlgoStats[k]; ok {
    			out = append(out, AlgoStat{Algo: k, Rounds: rounds})
    		}
    	}
    	c.JSON(200, gin.H{"algos": out})
    })

    // GET /api/performance/errorrate
    // 返回 { algos: [{algo, points:[{round,errorRate}]}] }，round=100/1000
    api.GET("/performance/errorrate", func(c *gin.Context) {
    	tradeMu.RLock()
    	defer tradeMu.RUnlock()

    	if allAlgoErrorRateStats == nil {
    		c.JSON(200, gin.H{"algos": []AlgoErrorStat{}})
    		return
    	}

    	out := make([]AlgoErrorStat, 0, 4)
    	order := []string{"pbft", "pos", "raft", "custom"}
    	for _, k := range order {
    		if pts, ok := allAlgoErrorRateStats[k]; ok {
    			out = append(out, AlgoErrorStat{Algo: k, Points: pts})
    		}
    	}
    	c.JSON(200, gin.H{"algos": out})
    })

    // GET /api/performance/leaderchanges
    // 返回 { algos: [{algo, points:[{round,leaderChanges}]}] }，round=100/1000
    api.GET("/performance/leaderchanges", func(c *gin.Context) {
    	tradeMu.RLock()
    	defer tradeMu.RUnlock()

    	if allAlgoLeaderChangeStats == nil {
    		c.JSON(200, gin.H{"algos": []AlgoLeaderChangeStat{}})
    		return
    	}

    	out := make([]AlgoLeaderChangeStat, 0, 4)
    	order := []string{"pbft", "pos", "raft", "custom"}
    	for _, k := range order {
    		if pts, ok := allAlgoLeaderChangeStats[k]; ok {
    			out = append(out, AlgoLeaderChangeStat{Algo: k, Points: pts})
    		}
    	}
    	c.JSON(200, gin.H{"algos": out})
    })
	r.Run(":5000")
}