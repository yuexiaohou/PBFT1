package pos

import (
	"math/rand"
	"time"
)

// ======================= 2026-03-06 高亮新增：多轮POS仿真（复用stake）BEGIN =======================
type RoundSummary struct {
	Round   int
	Success bool
	Leader  string
}

func RunSimulator(totalRounds int, cfg SimConfig) ([]RoundSummary, []*SimNode) {
	rand.Seed(time.Now().UnixNano())

	nodes := NewNodes(cfg)
	out := make([]RoundSummary, 0, totalRounds)

	for r := 1; r <= totalRounds; r++ {
		res := RunPOSWithNodes(
			// txId 只是模拟用
			// 这里用 round 作为 txId
			// amount 在你的系统里也只是演示数值
			// 可随意给
			"round-"+itoa(r),
			10,
			nodes,
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