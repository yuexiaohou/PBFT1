package pbft

import (
	"time"
	"math/rand"
	"fmt"
)

type Validator struct {
	ID   string
	Vote string
}

type PBFTResult struct {
	TxId         string
	Status       string
	Consensus    string
	BlockHeight  int
	Timestamp    time.Time
	Validators   []Validator
	FailedReason string
	Price        float64
	LeaderNode   string
}

var pbftHeight = 1

func RunPBFT(txId string, amount int) PBFTResult {
	leader := "node-0"
	validators := []Validator{}
	commits := 0
	price := 500.0 + float64(rand.Intn(20))
	for i:=0; i<7; i++ {
		vote := "commit"
		if rand.Float64() < 0.1 {vote = "reject"}
		if vote=="commit" {commits++}
		validators = append(validators, Validator{ID: fmt.Sprintf("node-%d", i), Vote: vote})
	}
	pbftHeight++
	status := "已确认"
	if commits < 5 {status = "失败"}
	return PBFTResult{
		TxId: txId,
		Status: status,
		Consensus: "pbft",
		BlockHeight: pbftHeight,
		Timestamp: time.Now(),
		Validators: validators,
		FailedReason: "",
		Price: price,
		LeaderNode: leader,
	}
}