package raft

const (
	DefaultTerm    = 1      // 初始 term
	ValidatorNum   = 100    // 默认节点数
	SimRounds      = 20     // 仿真轮数
	ElectionTimeout = 5     // 选主超时时间（单位可调）
	MaxFailures     = 5     // 最大允许故障节点数
)