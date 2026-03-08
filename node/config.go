package node

// ======================= 【高亮-2026-03-08】新增：node 层共用参数（奖励/恶意阈值） =======================
// 目的：node/node_apbft.go 里的 UpdateReward / NewProgressNode 需要 InitialM/MMax/MMin，
// 不再从 pbft1/config.go 读取，避免跨包耦合。
const (
	InitialM = 5
	MMax     = 10
	MMin     = 0
)