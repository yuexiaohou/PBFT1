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
	pbft "PBFT1/pbft1"
	"math/rand"
)

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

type PBFTConsensusResult struct {
	TxId         string          `json:"txId"`
	Status       string          `json:"status"`
	Consensus    string          `json:"consensus"`
	BlockHeight  int             `json:"blockHeight"`
	Timestamp    time.Time       `json:"timestamp"`
	Validators   []PBFTValidator `json:"validators"`
	FailedReason string          `json:"failedReason,omitempty"`
	// ======================= 【高亮】如需补充，PBFT共识结果也可加入价格与节点字段 =======================
    Price      float64          `json:"price,omitempty"`   // <== 可选用于 PBFTResult 前端展示
    LeaderNode string           `json:"leaderNode,omitempty"`
    // ======================= 【高亮】END =======================
}

type PBFTBlock struct {
	Height       int       `json:"height"`
	Timestamp    time.Time `json:"timestamp"`
	ConfirmedTxs int       `json:"confirmedTxs"`
}

// ========= 性能与展示缓存 =========
var (
	tradeMu         sync.RWMutex // ========== 高亮: 保护全局统计（并发） ==========
	roundOverview   []struct {
		Round      int
		MinPrice   float64
		Buyer      string
		Seller     string
		SuccessRate float64
	}
)

var (
	latestPBFTResult PBFTConsensusResult
	latestBlock PBFTBlock
	pbftMu sync.RWMutex
	// ========== 高亮：用于存放每轮撮合结果的全局变量 ==========
    roundMatchResults []TradeHistory
    roundOverview     []struct {Round int; MinPrice float64; Buyer, Seller string; SuccessRate float64}
    // ========== 高亮END ==========
)

// 转换 pbft.Result.Validators 到页面需要的形式
func convertValidators(origin []pbft.Validator) []PBFTValidator {
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

// ========== PBFT状态更新函数 ========= 高亮新增 START =========
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
// ========== PBFT状态更新函数 ========= 高亮新增 END =========

// ========== 高亮：撮合统计写库的新函数 ==========
func recordMatchStatsToDB(db *gorm.DB, stats []struct {
	Round int; MinPrice float64; Buyer, Seller string; SuccessRate float64
}, matches map[int][]pbft.Trade) {
	for _, s := range stats {
		if trades, ok := matches[s.Round]; ok {
			for _, t := range trades {
				history := TradeHistory{
					Type:      "match",
					Amount:    int(t.Quantity),
					Time:      t.Timestamp,
					Status:    "撮合成功",
					Price:     t.Price,
					Node:      t.Node,                // 可以由主节点或撮合节点决定
					Round:     s.Round,
					BuyerNode: t.BuyerNode,
					SellerNode: t.SellerNode,
				}
				db.Create(&history)
			}
		}
	}
}
// ========== 高亮 END ==========

// ========== 高亮: 后台撮合仿真与统计 ==========
func runPBFTSimToDB(db *gorm.DB, numNodes, rounds int, malRatio float64) {
	useBlst := false
	nodes := make([]*pbft.Node, numNodes)
	for i := 0; i < numNodes; i++ {
		throughput := 50.0 + rand.Float64()*150.0
		nodes[i] = pbft.NewNode(i, throughput, false, useBlst)
	}
	sim := pbft.NewPBFTSimulator(nodes, useBlst)
	sim.ComputeTiers()
	matches := make(map[int][]pbft.Trade)
	overviewStats := make([]struct {
		Round int
		MinPrice float64
		Buyer    string
		Seller   string
		SuccessRate float64
	}, 0, rounds)

	for r := 0; r < rounds; r++ {
		if r%5 == 0 && r > 0 {
			for _, nd := range sim.Nodes() {
				nd.Throughput = nd.Throughput * (0.9 + rand.Float64()*0.2)
			}
			sim.ComputeTiers()
		}
		ob := pbft.NewOrderBook()
		users := []string{"Alice", "Bob", "Carol", "David"}
		for i := 0; i < 10; i++ {
			ob.SubmitOrder(pbft.Buy, 500+rand.Float64()*30, 10+rand.Float64()*3, users[i%len(users)])
			ob.SubmitOrder(pbft.Sell, 495+rand.Float64()*20, 5+rand.Float64()*6, users[(i+1)%len(users)])
		}
		trades := ob.MatchAndClear()

		minPrice := 0.0
		buyer := ""; seller := ""
		if len(trades) > 0 {
			minPrice = trades[0].Price
			buyer = trades[0].BuyerNode
			seller = trades[0].SellerNode
			for _, t := range trades {
				if t.Price < minPrice { minPrice = t.Price; buyer = t.BuyerNode; seller = t.SellerNode }
			}
		}
		successRate := float64(len(trades)) / float64(20)
		overviewStats = append(overviewStats, struct {
			Round int; MinPrice float64; Buyer, Seller string; SuccessRate float64
		}{r, minPrice, buyer, seller, successRate})
		matches[r] = trades
	}
	tradeMu.Lock()
	roundOverview = overviewStats
	tradeMu.Unlock()
	// ========== 高亮：撮合写入数据库 ==========
	recordMatchStatsToDB(db, overviewStats, matches)
	// ========== 高亮 END ==========
}

func main() {
    // ========= 高亮：命令行参数替代固定参数（支持配置） ========
	numNodes := flag.Int("nodes", 100, "number of PBFT nodes")
	totalRounds := flag.Int("rounds", 20, "number of consensus rounds")
	simMalRatio := flag.Float64("maliciousRatio", 0.05, "malicious node ratio")
	flag.Parse()
	// ========= 高亮END ==========

	go func() {
		// ======= 高亮：自动调用算法层后台仿真流程 =======
		db := dbConnect()
		runPBFTSimToDB(db, *numNodes, *totalRounds, *malRatio)
	}()
    // ====== 高亮END ======
	db := dbConnect()
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
    	// ====== 高亮：新增 PBFT result 接口 END ======

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
			// ======================= 【高亮】充值可无价格与节点 =======================
            Price: 0, Node: "",
            // ======================= 【高亮】END =======================
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
            db.Create(&TradeHistory{
                UserID: user.ID, Type: req.Type, Amount: req.Amount, Time: time.Now(), Status: status,
            })

            // ====== 高亮新增: 声明并赋值 nowTxId 与 pbftResult ======
            nowTxId := fmt.Sprintf("%s_%d", username, time.Now().UnixNano())
            pbftResult := pbft.RunPBFT(nowTxId, req.Amount)
            validators := convertValidators(pbftResult.Validators)
            // ======================= 【高亮】交易记入价格和撮合节点 =======================
            tradePrice := pbftResult.Price      // pbft模拟器需返回 Price 字段
            // ======================= 【高亮】END =======================
            // =========== 【高亮】成交价格与卖出节点模拟 =============
            // ==========【高亮】获取卖出节点（LeaderNode）==========
            sellNode := pbftResult.LeaderNode
            if req.Type == "buy" && status == "成功" {
			if pbftResult.Price != 0 {
				tradePrice = pbftResult.Price
			} else {
				tradePrice = float64(500 + rand.Intn(20))
			}
		    }
            // =========== 【高亮】END =============

            if status == "成功" && pbftResult.Status == "已确认" {
                updatePBFTResult(pbftResult.TxId, pbftResult.Status, pbftResult.Consensus, pbftResult.BlockHeight, validators, pbftResult.FailedReason)
                updatePBFTBlock(pbftResult.BlockHeight, req.Amount)
			db.Create(&TradeHistory{
				UserID: user.ID,
				Type: req.Type,
				Amount: req.Amount,
				Time: time.Now(),
				Status: status,
				Price: tradePrice,
				Node: sellNode, // 只用 LeaderNode
			})
                c.JSON(200, gin.H{"msg": "操作成功"})
            }else {
                reason := pbftResult.FailedReason
                if status != "成功" {
                    reason = "余额不足"
                }
                updatePBFTResult(nowTxId, "失败", "pbft", pbftResult.BlockHeight, validators, reason)
                c.JSON(400, gin.H{"msg": reason})
            }

		// ====== 高亮: 实际用pbft包算法模拟一次共识/区块 ======
        	if status == "成功" {
        	    updatePBFTResult(pbftResult.TxId, pbftResult.Status, pbftResult.Consensus, pbftResult.BlockHeight, validators, "")
        	    updatePBFTBlock(pbftResult.BlockHeight, 36)
        		c.JSON(200, gin.H{"msg": "操作成功"})
        	} else {
        		validators := []PBFTValidator{
        			{ID: "node1", Vote: "commit"},
        			{ID: "node2", Vote: "commit"},
        			{ID: "node3", Vote: "commit"},
        			{ID: "node4", Vote: "commit"},
        		}
                updatePBFTResult(nowTxId, "失败", "pbft", 10001, validators, "余额不足")
        		c.JSON(400, gin.H{"msg": "余额不足"})
        		}
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
		        "price":  r.Price,     // <== 新增
        		"node":   r.Node,      // <== 新增
				"time":   r.Time.Format("2006-01-02 15:04:05"),
				"status": r.Status,
			})
		}
		c.JSON(200, gin.H{"records": out})
	})
	// ========== 高亮：撮合图表接口1：最低价格随轮次变化 ==========
	api.GET("/trade/pricechart", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"rounds": roundOverview,
		})
	})
	// ========== 高亮：撮合图表接口2：每轮撮合率随轮次变化 ==========
	api.GET("/trade/successrate", func(c *gin.Context) {
		x := []int{}
		y := []float64{}
		for _, rv := range roundOverview {
			x = append(x, rv.Round)
			y = append(y, rv.SuccessRate)
		}
		c.JSON(200, gin.H{"x": x, "y": y})
	})
	// ========== 高亮: PBFT前端API接口 ==========

	r.Run(":5000")
}