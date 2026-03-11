package pbft

import (
	"fmt"
	"math/rand"
	"time"
	// ======================= 【高亮-2026-03-08】引入通用节点池规格 =======================
	"PBFT1/node"
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

// ======================= 【高亮-2026-03-11】修改：升级为完整三阶段 PBFT 并对齐阈值 =======================
func RunPBFTWithRoundAndSpecs(round int, txId string, amount int, specs []node.NodeSpec) PBFTResult {
	n := len(specs)
	if n <= 0 {
		return PBFTResult{TxId: txId, Status: "失败", FailedReason: "no nodes", Timestamp: time.Now()}
	}

	// 计算容错数 f 和达成共识需要的法定人数 (2f + 1)
	f := (n - 1) / 3
	quorum := 2*f + 1

	// Leader 轮转逻辑与 apbft 对齐
	leaderIdx := round % n
	leader := fmt.Sprintf("node-%d", specs[leaderIdx].ID)

	seed := int64(20260308 + round)
	rng := rand.New(rand.NewSource(seed))

	// --- 阶段 1: Pre-Prepare ---
	if specs[leaderIdx].IsMalicious && rng.Float64() < 0.3 {
		return failResult(txId, round, leader, "Pre-Prepare failed: Malicious leader")
	}

	// --- 阶段 2: Prepare (收集投票) ---
	prepareVotes := 0
	validators := make([]Validator, 0, n)

	for i := 0; i < n; i++ {
		vote := "prepare"
		if specs[i].IsMalicious {
			if rng.Float64() < 0.6 { vote = "reject" }
		} else {
			if rng.Float64() < 0.05 { vote = "reject" }
		}

		if vote == "prepare" {
			prepareVotes++
		}
		validators = append(validators, Validator{
			ID:   fmt.Sprintf("node-%d", specs[i].ID),
			Vote: vote,
		})
	}

	if prepareVotes < quorum {
		return failResult(txId, round, leader, fmt.Sprintf("Prepare phase failed: %d/%d", prepareVotes, quorum))
	}

	// --- 阶段 3: Commit (最终确认) ---
	commitVotes := 0
	for i := 0; i < n; i++ {
		// 只有 Prepare 成功的节点进入 Commit
		if validators[i].Vote == "prepare" {
			if specs[i].IsMalicious && rng.Float64() < 0.4 {
				continue
			}
			commitVotes++
			validators[i].Vote = "commit"
		}
	}

	status := "已确认"
	reason := ""
	if commitVotes < quorum {
		status = "失败"
		reason = fmt.Sprintf("Commit phase failed: %d/%d", commitVotes, quorum)
	}

	// 撮合价格机理对齐：500 + 随机扰动
	price := 500.0 + rng.Float64()*20.0

	return PBFTResult{
		TxId:         txId,
		Status:       status,
		Consensus:    "pbft",
		BlockHeight:  round,
		Timestamp:    time.Now(),
		Validators:   validators,
		FailedReason: reason,
		Price:        price,
		LeaderNode:   leader,
	}
}

// 【高亮-2026-03-11】新增：统一失败结果处理
func failResult(txId string, round int, leader string, reason string) PBFTResult {
	return PBFTResult{
		TxId: txId, Status: "失败", Consensus: "pbft", BlockHeight: round,
		Timestamp: time.Now(), FailedReason: reason, LeaderNode: leader,
	}
}

func RunPBFT(txId string, amount int) PBFTResult {
	specs := node.NewPool(1, node.FixedNumNodes, node.FixedMaliciousRatio)
	return RunPBFTWithRoundAndSpecs(1, txId, amount, specs)
}