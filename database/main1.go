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
	// ===== 高亮-2026-03-01：新增主仿真入口导入 =====
	"math/rand"
	"math"
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

// === 2026-03-03 高亮新增: 轮次统计结构定义 ===
type RoundStat struct {
	Round       int     `json:"round"`
	MinPrice    float64 `json:"minPrice"`
	BuyerNode   string  `json:"buyerNode"`
	SellerNode  string  `json:"sellerNode"`
	SuccessRate float64 `json:"successRate"`
}

// ========= 性能与展示缓存 =========
var tradeMu   sync.RWMutex // ========== 高亮: 保护全局统计（并发） ==========
var roundOverview []RoundStat

var (
	latestPBFTResult PBFTConsensusResult
	latestBlock PBFTBlock
	pbftMu sync.RWMutex
	// ========== 高亮：用于存放每轮撮合结果的全局变量 ==========
    roundMatchResults []TradeHistory
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

// === 2026-03-03 高亮新增: 撮合仿真核心逻辑示例 ===
func simulateRounds(db *gorm.DB, totalRounds int) {
	for r := 1; r <= totalRounds; r++ {
		trades := make([]TradeHistory, 0)
		successCount := 0
		minPrice := math.MaxFloat64
		var minBuyer, minSeller string

		numTrades := rand.Intn(5) + 5 // 每轮随机产生5~9个交易
		for i := 0; i < numTrades; i++ {
			buyer := fmt.Sprintf("Node-%02d", rand.Intn(20))
			seller := fmt.Sprintf("Node-%02d", rand.Intn(20))
			price := rand.Float64()*500 + 30 // 随机价格30~530
			amount := rand.Intn(50) + 10
			status := "失败"
			if rand.Float64() < 0.7 { // 70%概率成交
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
				Round:      r,             // === 2026-03-03 高亮新增 ===
				BuyerNode:  buyer,         // === 2026-03-03 高亮新增 ===
				SellerNode: seller,        // === 2026-03-03 高亮新增 ===
			}
			trades = append(trades, trade)
			db.Create(&trade)
		}

		if minPrice == math.MaxFloat64 { minPrice = 0 }
		successRate := 0.0
		if numTrades > 0 {
			successRate = float64(successCount) / float64(numTrades)
		}

		roundOverview = append(roundOverview, RoundStat{
			Round:       r,
			MinPrice:    minPrice,
			BuyerNode:   minBuyer,
			SellerNode:  minSeller,
			SuccessRate: successRate,
		})
		// 可以输出一行日志
		fmt.Printf("[模拟轮 %d] 最低价: %v 买方: %s 卖方: %s 成功挂单率: %.2f%%\n",
			r, minPrice, minBuyer, minSeller, successRate*100)
	}
}

func main() {
	// ========= 高亮-2026-03-01: 命令行参数配置 ==========
	numNodes := flag.Int("nodes", 100, "number of PBFT nodes")
	totalRounds := flag.Int("rounds", 20, "number of consensus rounds")
	simMalRatio := flag.Float64("maliciousRatio", 0.2, "malicious node ratio")
	flag.Parse()
	// ========= 高亮END ==========

	// ========= 高亮-2026-03-01: 启动仿真算法（统一入口） ==========
	go func() {
		// 启动主仿真流程，核心部分已全部交由main.go控制（不在此重复核心逻辑）
		// main.go中的RunPBFTSimulator会输出trade.log，建议trade流入数据库时务必有逻辑同步
		pbft.RunPBFTSimulator(*numNodes, -1, *simMalRatio, *totalRounds) // ===== 高亮-2026-03-01：调用入口 =====

		// ===== 高亮-2026-03-01: 可在此补充 trade.log->DB 的搬运（如主包未自动写库）
		// parseTradeLog("trade.log", db) // 可选补充
	}()
	// ========= 高亮END ==========

	db := dbConnect()
// === 2026-03-03 高亮新增: 启动时自动模拟撮合轮次（正式项目应由业务流程驱动） ===
	simulateRounds(db, 30)
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