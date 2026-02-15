package main

import (
	"time"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
	"github.com/gin-contrib/cors"
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
	ID     uint      `gorm:"primaryKey"`
	UserID uint
	Type   string    `gorm:"size:20"`
	Amount int
	Time   time.Time
	Status string    `gorm:"size:20"`
}

// 数据库连接
func dbConnect() *gorm.DB {
	dsn := "user:password@tcp(127.0.0.1:3306)/yourdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Database connection failed")
	}
	db.AutoMigrate(&User{}, &Balance{}, &TradeHistory{})
	return db
}

func main() {
	db := dbConnect()
	r := gin.Default()

	// 允许前端跨域
	r.Use(cors.Default())

	api := r.Group("/api")

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
		if status == "成功" {
			c.JSON(200, gin.H{"msg": "操作成功"})
		} else {
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
				"time":   r.Time.Format("2006-01-02 15:04:05"),
				"status": r.Status,
			})
		}
		c.JSON(200, gin.H{"records": out})
	})

	r.Run(":5000")
}