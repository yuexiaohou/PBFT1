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

func RunPBFTWithRoundAndMaliciousRatio(round int, txId string, amount int, maliciousRatio float64) PBFTResult {
	leader := fmt.Sprintf("node-%d", round%7)

	// 每一轮 round 固定随机源 => 固定一批恶意节点/投票结果（可复现）
	seed := int64(20260307 + round)
	rng := rand.New(rand.NewSource(seed))

	// 这里保持你原本的“7 个 validator”的简化模型
	const totalValidators = 7

	// 恶意节点数量（按比例）
	mCount := int(float64(totalValidators) * maliciousRatio)
	if mCount < 0 {
		mCount = 0
	}
	if mCount > totalValidators {
		mCount = totalValidators
	}

	// 固定恶意节点集合（对同一个 round 不变）
	malSet := map[int]bool{}
	if mCount > 0 {
		idxs := rng.Perm(totalValidators)[:mCount]
		for _, idx := range idxs {
			malSet[idx] = true
		}
	}

	validators := make([]Validator, 0, totalValidators)
	commits := 0

	// 投票：正常节点小概率 reject；恶意节点高概率 reject
	for i := 0; i < totalValidators; i++ {
		vote := "commit"

		if malSet[i] {
			// 恶意节点：大概率拒绝（你可再调大/调小）
			if rng.Float64() < 0.80 {
				vote = "reject"
			}
		} else {
			// 正常节点：小概率拒绝
			if rng.Float64() < 0.10 {
				vote = "reject"
			}
		}

		if vote == "commit" {
			commits++
		}
		validators = append(validators, Validator{ID: fmt.Sprintf("node-%d", i), Vote: vote})
	}

	// 阈值：至少 5 个 commit 认为成功（沿用你原逻辑）
	status := "已确认"
	reason := ""
	if commits < 5 {
		status = "失败"
		reason = fmt.Sprintf("commits=%d < 5", commits)
	}

	price := 500.0 + float64(rng.Intn(20))

	return PBFTResult{
		TxId:         txId,
		Status:       status,
		Consensus:    "pbft",
		BlockHeight:  round, // 【高亮-2026-03-07】与 round 一致
		Timestamp:    time.Now(),
		Validators:   validators,
		FailedReason: reason,
		Price:        price,
		LeaderNode:   leader,
	}
}

// ======================= 【高亮-2026-03-07】兼容旧接口：不传 round/maliciousRatio 时给默认值 =======================
func RunPBFT(txId string, amount int) PBFTResult {
	return RunPBFTWithRoundAndMaliciousRatio(1, txId, amount, 0)
}