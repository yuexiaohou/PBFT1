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
	pbft "PBFT1/pbft1"
	"math/rand"
)

// 用户表
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

// ===== 高亮：TradeHistory 增加轮次和节点信息字段 =====
type TradeHistory struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint
	Type      string    `gorm:"size:20"`
	Amount    int
	Time      time.Time
	Status    string    `gorm:"size:20"`
	Price     float64   `gorm:"column:price"`
	Node      string    `gorm:"column:node"`
	Round     int       // <== 本次高亮：共识轮次
	BuyerNode string    // <== 本次高亮：模拟买节点
	SellerNode string   // <== 本次高亮：模拟���节点
}
// ===== 高亮END =====

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

// ===== 高亮：只声明一次全局 roundOverview，类型与字段适配 =====
var roundOverview []struct {
	Round      int
	MinPrice   float64
	Buyer      string
	Seller     string
	SuccessRate float64
}
// ===== 高亮END =====
var tradeMu sync.RWMutex

func dbConnect() *gorm.DB {
	dsn := "root:111111@tcp(127.0.0.1:3306)/yourdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("Database connection failed")
	}
	db.AutoMigrate(&User{}, &Balance{}, &TradeHistory{})
	return db
}

func main() {
	// ===== 高亮：参数统一与变量名适配 =====
	numNodes := flag.Int("nodes", 100, "number of PBFT nodes")
	totalRounds := flag.Int("rounds", 20, "number of consensus rounds")
	malRatio := flag*
