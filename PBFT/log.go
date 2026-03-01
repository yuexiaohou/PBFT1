package pbft

const (
	InitialM                 = 5
	MMax                     = 10  // 中心化阈值
	MMin                     = 0   // 恶意阈值
	PrepareQuorumMultiplier  = 2.0/3.0
	NodeCount                = 100 // 默认节点数
	SimRounds                = 20  // 默认轮数
)