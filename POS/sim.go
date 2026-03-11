package pos
//=============sim.go的1作用是实现POS算法的多轮共识===================
import (
	"fmt"
	// ======================= 【高亮-2026-03-09】新增：仿真只走共用节点池 node.NewPool(...) =======================
    "PBFT1/node"
)

// ======================= 2026-03-06 高亮新增：多轮POS仿真（复用stake）BEGIN =======================
type RoundSummary struct {
	Round   int
	Success bool
	Leader  string
}

// ======================= 【高亮-2026-03-11】修改：确保 POS 仿真完全闭环使用 NodePool 对齐 =======================
func RunSimulator(totalRounds int, cfg SimConfig) ([]RoundSummary, []*SimNode) {
	// 1. 获取初始规格（第1轮），用于初始化跨轮次复用的节点集合（以便累积 Stake 奖惩）
	specs0 := node.NewPool(1, node.FixedNumNodes, node.FixedMaliciousRatio)
	nodes := NewNodesFromSpecs(specs0)

	out := make([]RoundSummary, 0, totalRounds)

	for r := 1; r <= totalRounds; r++ {
		// 2. 【高亮-2026-03-11】对齐点：每轮重新获取 specs。
		// 这样可以确保在第 r 轮，POS 看到的恶意节点 ID 与 PBFT/APBFT/RAFT 完全一致。
		specs := node.NewPool(r, node.FixedNumNodes, node.FixedMaliciousRatio)

		// 3. 执行单轮 POS 共识逻辑
		// 注意：RunPOSWithRoundAndSpecs 内部会调用 SyncNodesFromSpecs 同步 specs 的恶意状态到 nodes 中
		res := RunPOSWithRoundAndSpecs(
			r,
			fmt.Sprintf("pos-tx-round-%d", r),
			10,
			nodes,
			specs,
			cfg,
		)

		// 4. 记录该轮结果
		ok := res.Status == "已确认"
		out = append(out, RoundSummary{
			Round:   r,
			Success: ok,
			Leader:  res.Leader,
		})

		// 5. 【备注】Stake 的奖惩更新已在 RunPOSWithRoundAndSpecs 内部的投票逻辑中完成。
		// 正常投票节点增加 Stake，恶意/拒绝节点扣除 Stake，从而影响下一轮 Leader 选举权重。
	}

	return out, nodes
}

// itoa：辅助函数，避免引入外部依赖将整数转为字符串
func itoa(x int) string {
	if x == 0 {
		return "0"
	}
	sign := ""
	if x < 0 {
		sign = "-"
		x = -x
	}
	buf := make([]byte, 0, 12)
	for x > 0 {
		buf = append(buf, byte('0'+x%10))
		x /= 10
	}
	// reverse 翻转字符串
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return sign + string(buf)
}