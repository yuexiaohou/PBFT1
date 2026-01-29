package main // 定义包为 main，表示此文件属于可独立运行的程序

import ( // 导入必要的标准库包
	"fmt"   // 格式化 I/O，用于打印日志
	"math/rand" // 随机数，用于模拟恶意行为概率
	"sort"  // 排序，用于对节点排序（例如选择 leader、计算 tiers）
	"sync"  // 并发原语，用于等待并保护共享切片
)

// 简化 PBFT 模拟器（PRE-PREPARE / PREPARE / COMMIT）
// 定义 PBFT 模拟器的结构体，封装节点集合与参数
type PBFTSimulator struct {
	nodes   []*Node // 节点切片，表示参与共识的所有节点
	n       int     // 节点总数
	f       int     // 最大容忍拜占庭节点数 (f)
	useBlst bool    // 是否使用 BLS（布鲁姆/聚合签名）库的标志
}

func NewPBFTSimulator(nodes []*Node, useBlst bool) *PBFTSimulator { // 构造函数：创建 PBFTSimulator 实例
	n := len(nodes)                         // 计算节点数
	f := (n - 1) / 3                        // 根据 PBFT 理论计算可容错的拜占庭个数 f
	return &PBFTSimulator{nodes: nodes, n: n, f: f, useBlst: useBlst} // 返回新建实例
}

func (s *PBFTSimulator) SelectLeader(round int) *Node { // 选择 leader 的策略函数，基于 round 与节点状态
	active := []*Node{} // 创建临时切片保存所有活跃节点
	for _, nd := range s.nodes { // 遍历所有节点
		if nd.IsActive() { // 如果节点是活跃的（未宕机/在线）
			active = append(active, nd) // 将其加入活跃列表
		}
	}
	if len(active) == 0 { // 如果没有活跃节点
		return nil // 返回 nil，表示无法选择 leader
	}
	// 按照节点的 m 值降序排序，m 可能代表某种优先级或信誉评分
	sort.Slice(active, func(i, j int) bool {
		return active[i].m > active[j].m // 更大的 m 值排在前面
	})
	idx := round % len(active) // 根据轮次取模选择领导者索引（简单轮换）
	return active[idx] // 返回选中的 leader 节点
}

func (s *PBFTSimulator) ComputeTiers() { // 计算节点的层级（高/普通/低）基于吞吐量排序
	arr := append([]*Node{}, s.nodes...) // 复制节点切片以便排序，不修改原切片
	// 按 Throughput（吞吐量）降序排序
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].Throughput > arr[j].Throughput
	})
	n := len(arr)                         // 节点总数
	top := int(float64(n) * 0.3)          // 取前 30% 为高层
	bottom := int(float64(n) * 0.3)       // 取后 30% 为低层
	if top < 1 {                          // 保证至少有一个高层节点
		top = 1
	}
	if bottom < 1 {                       // 保证至少有一个低层节点
		bottom = 1
	}
	for i, nd := range arr { // 遍历排序后的节点数组，分配 Tier
		if i < top { // 前 top 个为高层
			nd.Tier = TierHigh
		} else if i >= n-bottom { // 最后 bottom 个为低层
			nd.Tier = TierLow
		} else { // 中间为普通层
			nd.Tier = TierNormal
		}
	}
}

// RunRound 发起单轮共识，返回是否达成共识；并采集简单日志（可扩展为 CSV）
func (s *PBFTSimulator) RunRound(round int, request []byte) bool {
	leader := s.SelectLeader(round) // 选择当前轮次的 leader
	if leader == nil { // 如果没有可用 leader
		fmt.Println("No active leader available") // 打印错误信息
		return false // 共识失败
	}
	fmt.Printf("\n--- Round %d: leader=%s ---\n", round, leader.String()) // 输出本轮 leader 信息

	// PRE-PREPARE
	if leader.IsMalicious && rand.Float64() < 0.5 { // 如果 leader 是恶意并以 50% 概率作恶
		fmt.Printf("Leader %d acted maliciously in pre-prepare\n", leader.ID) // 打印作恶日志
		leader.UpdateReward(false) // 更新 leader 的奖励/惩罚（作恶导致失败）
		return false // 提前返回失败
	}

	// PREPARE: 所有活跃节点签名
	var wg sync.WaitGroup // 等待组，用于并发收集签名
	var mu sync.Mutex     // 互斥锁，保护共享切片
	signatures := make([][]byte, 0, s.n) // 收集每个节点对请求的签名切片
	pubKeys := make([][]byte, 0, s.n)    // 收集每个节点的公钥切片
	for _, nd := range s.nodes { // 遍历所有节点
		if !nd.IsActive() { // 跳过非活跃节点
			continue
		}
		wg.Add(1) // 增加等待计数
		go func(node *Node) { // 并发签名以模拟真实网络的并行性
			defer wg.Done() // 完成时通知等待组
			sig, err := node.Sign(request) // 节点对请求进行签名
			if err == nil && sig != nil { // 如果签名成功
				mu.Lock() // 保护共享切片
				signatures = append(signatures, sig) // 添加签名
				pubKeys = append(pubKeys, node.bls.PublicKey()) // 添加对应公钥
				mu.Unlock() // 解锁
			}
		}(nd)
	}
	wg.Wait() // 等待所有并发签名完成

	// leader 聚合
	aggSig, _ := leader.bls.AggregateSignatures(signatures) // leader 使用 BLS 聚合所有签名（忽略错误返回）

	// leader 验证聚合签名
	ok, _ := leader.bls.VerifyAggregate(pubKeys, request, aggSig) // 验证聚合签名是否有效（忽略错误返回）
	if !ok { // 如果验证失败
		fmt.Println("Aggregate verification failed in prepare phase") // 打印错误信息
		leader.UpdateReward(false) // 更新 leader 奖励为失败
		return false // 共识失败
	}

	// COMMIT: 节点对聚合签名再次签名（模拟）
	commitSigs := make([][]byte, 0)    // 收集 commit 阶段的签名
	commitPubKeys := make([][]byte, 0) // 收集 commit 阶段的公钥
	for _, nd := range s.nodes { // 遍历所有节点
		if !nd.IsActive() { // 跳过非活跃节点
			continue
		}
		sig, err := nd.Sign(aggSig) // 节点对聚合签名再签一次，作为 commit 的签名（模拟）
		if err == nil && sig != nil { // 如果签名成功
			commitSigs = append(commitSigs, sig) // 收集 commit 签名
			commitPubKeys = append(commitPubKeys, nd.bls.PublicKey()) // 收集公钥
		}
	}

	aggCommitSig, _ := leader.bls.AggregateSignatures(commitSigs) // leader 聚合 commit 签名
	ok2, _ := leader.bls.VerifyAggregate(commitPubKeys, aggSig, aggCommitSig) // 验证聚合的 commit 签名（以 aggSig 作为消息）
	if !ok2 { // 如果 commit 阶段验证失败
		fmt.Println("Aggregate verification failed in commit phase") // 打印错误信息
		leader.UpdateReward(false) // 更新奖励为失败
		return false // 共识失败
	}

	// 判断阈值
	if len(commitSigs) >= int(float64(s.n)*PrepareQuorumMultiplier) { // 如果 commit 签名数达到阈值（基于 PrepareQuorumMultiplier）
		fmt.Println("Consensus achieved in this round") // 打印达成共识
		successIDs := map[int]bool{} // 创建映射以记录哪些节点参与了成功的 commit
		for _, pk := range commitPubKeys { // 遍历 commit 的公钥切片
			// stub 公钥解析演示（若真实 pk 为字节流则需其它映射）
			var id int
			fmt.Sscanf(string(pk), "PK-node-%02d", &id) // 通过格式化字符串解析出节点 ID（这里是模拟）
			successIDs[id] = true // 标记该 id 为成功参与者
		}
		for _, nd := range s.nodes { // 遍历所有节点以更新奖励/惩罚
			if successIDs[nd.ID] { // 如果该节点在成功列表中
				nd.UpdateReward(true) // 更新奖励为成功
			} else {
				nd.UpdateReward(false) // 否则视为未参与或失败
			}
		}
		return true // 返回共识成功
	} else {
		fmt.Println("Not enough commit signatures; consensus failed") // 未达到阈值，打印失败信息
		for _, nd := range s.nodes { // 对所有节点应用失败的奖励更新
			nd.UpdateReward(false)
		}
		return false // 返回共识失败
	}
}
