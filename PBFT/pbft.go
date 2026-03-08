package pbft

import (
	"time"
	"math/rand"
	"fmt"
	// ======================= 【高亮-2026-03-08】新增：引入通用节点池规格（node.NodeSpec） =======================
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

var pbftHeight = 1

// ======================= 【高亮-2026-03-08】修改：新增支持 nodepool specs 的 PBFT 共识入口 =======================
// 说明：这个函数与 main.go 的 simulatePBFT() 保持一致：
// - round：用于固定随机源（可复现）
// - maliciousRatio：用于从 specs 中决定恶意节点集合（如果 specs 已包含 IsMalicious，则 maliciousRatio 仅用于兜底）
// - specs：来自 node.NewPool(round, numNodes, maliciousRatio)，实现“每轮固定一批恶意节点”
// 注意：简化 PBFT 仍然使用 7 个 validator 模型；当 specs 数量>7 时取前 7 个（你也可以改为抽样）。
func RunPBFTWithRoundAndMaliciousRatio(round int, txId string, amount int, maliciousRatio float64, specs []node.NodeSpec) PBFTResult {
	// leader 仅用于展示：从 7 个 validator 里轮转
	leader := fmt.Sprintf("node-%d", round%7)

	// 每一轮 round 固定随机源 => 投票结果可复现
	seed := int64(20260308 + round)
	rng := rand.New(rand.NewSource(seed))

	// 仍然保持“7 个 validator”的简化模型
	const totalValidators = 7

	// 当 specs 不足 7 个，按实际数量来
	vn := totalValidators
	if len(specs) < vn {
		vn = len(specs)
	}
	if vn <= 0 {
		return PBFTResult{
			TxId:         txId,
			Status:       "失败",
			Consensus:    "pbft",
			BlockHeight:  round,
			Timestamp:    time.Now(),
			Validators:   nil,
			FailedReason: "no validators",
			Price:        0,
			LeaderNode:   leader,
		}
	}

	// ======================= 【高亮-2026-03-08】修改：
	// 优先使用 specs[i].IsMalicious（由 nodepool 固定恶意集合）
	// 如果 specs 全部是默认 false（例如外部没按 maliciousRatio 填），则兜底按 maliciousRatio 再生成一次 malSet。
	anyMal := false
	for i := 0; i < vn; i++ {
		if specs[i].IsMalicious {
			anyMal = true
			break
		}
	}

	malSet := make(map[int]bool, vn)

	if anyMal {
		for i := 0; i < vn; i++ {
			if specs[i].IsMalicious {
				malSet[i] = true
			}
		}
	} else {
		// 兜底：按 maliciousRatio 生成恶意集合（仍对同一 round 固定）
		mCount := int(float64(vn) * maliciousRatio)
		if mCount < 0 {
			mCount = 0
		}
		if mCount > vn {
			mCount = vn
		}
		if mCount > 0 {
			idxs := rng.Perm(vn)[:mCount]
			for _, idx := range idxs {
				malSet[idx] = true
			}
		}
	}

	validators := make([]Validator, 0, vn)
	commits := 0

	// 投票：正常节点小概率 reject；恶意节点高概率 reject
	for i := 0; i < vn; i++ {
		vote := "commit"

		if malSet[i] {
			// 恶意节点：大概率拒绝（可调）
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

		// ======================= 【高亮-2026-03-08】修改：
		// validator ID 使用 specs[i].ID，确保“共用节点池”的节点编号一致
		validators = append(validators, Validator{
			ID:   fmt.Sprintf("node-%d", specs[i].ID),
			Vote: vote,
		})
	}

	// 阈值：沿用原逻辑（>=5 commit 才成功）
	status := "已确认"
	reason := ""
	if commits < 5 {
		status = "失败"
		reason = fmt.Sprintf("commits=%d < 5", commits)
	}

	price := 500.0 + float64(rng.Intn(20))

	// 保留原 pbftHeight 变量但不强依赖它（你也可以删除 pbftHeight）
	pbftHeight++

	return PBFTResult{
		TxId:         txId,
		Status:       status,
		Consensus:    "pbft",
		BlockHeight:  round, // round 与 blockHeight 对齐，便于前端按轮次展示
		Timestamp:    time.Now(),
		Validators:   validators,
		FailedReason: reason,
		Price:        price,
		LeaderNode:   leader,
	}
}

// ======================= 【高亮-2026-03-08】修改：兼容旧接口（默认 round=1 且 maliciousRatio=0，specs=nil） =======================
func RunPBFT(txId string, amount int) PBFTResult {
	return RunPBFTWithRoundAndMaliciousRatio(1, txId, amount, 0, nil)
}