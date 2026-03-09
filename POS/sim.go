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

func RunSimulator(totalRounds int, cfg SimConfig) ([]RoundSummary, []*SimNode) {
	// round=1：用共用节点池初始化 stake/active
	specs0 := node.NewPool(1, node.FixedNumNodes, node.FixedMaliciousRatio)
	nodes := NewNodesFromSpecs(specs0)

	out := make([]RoundSummary, 0, totalRounds)

	for r := 1; r <= totalRounds; r++ {
		// 每轮都使用共用节点池：恶意集合固定（同一 round 可复现）
		specs := node.NewPool(r, node.FixedNumNodes, node.FixedMaliciousRatio)

		res := RunPOSWithRoundAndSpecs(
			r,
			fmt.Sprintf("round-%d", r),
			10,
			nodes,
			specs,
			cfg,
		)

		ok := res.Status == "已确认"
		out = append(out, RoundSummary{
			Round:   r,
			Success: ok,
			Leader:  res.Leader,
		})
	}

	return out, nodes
}

// 避免引入 strconv（也可以直接用 strconv.Itoa）
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
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return sign + string(buf)
}
// ======================= 2026-03-06 高亮新增：多轮POS仿真 END =======================