package main

// 全局参数，可按需调整
const (
	InitialM   = 5   // m0
	MMax       = 10  // mmax
	MMin       = 0   // mmin, 当 m < mmin 判为恶意并排除
	PrepareQuorumMultiplier = 2.0/3.0 // 准备/提交阶段阈值（简化）
)
