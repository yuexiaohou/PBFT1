package pos

import (
	"fmt"
	"math/rand"
	"time"
)

type POSResult struct {
	TxId        string
	Status      string
	Consensus   string
	BlockHeight int
	Timestamp   time.Time
	Validators  []string
	FailedReason string
	Price      float64
	SellNode   string
}

var posHeight = 1

func RunPOS(txId string, amount int) POSResult {
	rand.Seed(time.Now().UnixNano())
	validators := []string{}
	for i := 1; i <= 100; i++ { validators = append(validators, "node-" + fmt.Sprint(i)) }
	leaderIdx := rand.Intn(100)
	sellNode := validators[leaderIdx]
	price := 480.0 + float64(rand.Intn(40))
	posHeight++
	return POSResult{
		TxId: txId,
		Status: "已确认",
		Consensus: "pos",
		BlockHeight: posHeight,
		Timestamp: time.Now(),
		Validators: validators,
		FailedReason: "",
		Price: price,
		SellNode: sellNode,
	}
}