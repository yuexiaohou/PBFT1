package apbft // 定义包为 main，表示此文件属于可独立运行的程序

import ( // 导入必要的标准库包
	"fmt"   // 格式化 I/O，用于打印日志
	"math/rand" // 随机数，用于模拟恶意行为概率
	"sort"  // 排序，用于对节点排序（例如选择 leader、计算 tiers）
	"sync"  // 并发原语，用于等待并保护共享切片
	"time"
	"PBFT1/node"
)

// ======================= 【高亮-2026-03-22】新增：KNN 辅助结构与距离计算 =======================
type Neighbor struct {
	ID    int
	D     float64 // 标签 d: 与主节点的距离
	Quote float64 // 节点作为卖方的预期报价
}

// calculateNodeDistance 用于模拟两个节点在电网拓扑中的固定物理距离
func calculateNodeDistance(id1, id2 int) float64 {
	if id1 == id2 {
		return 0.0
	}
	// 利用节点ID生成固定的伪随机种子，保证两点间拓扑距离固定
	seed := int64(id1*1000 + id2)
	if id1 > id2 {
		seed = int64(id2*1000 + id1)
	}
	rng := rand.New(rand.NewSource(seed))
	return rng.Float64() * 100.0 // 模拟距离范围 0 ~ 100 KM
}

// 简化 PBFT 模拟器（PRE-PREPARE / PREPARE / COMMIT）
// 定义 PBFT 模拟器的结构体，封装节点集合与参数
//==========当使用其他文件的变量时，特别是次变量在其他目录下时，该变量要改为目录名.变量名。import("PBFT1/node")后，[]*Node变为[]*node.Node==========
type PBFTSimulator struct {
	nodes   []*node.Node// 节点切片，表示参与共识的所有节点
	n       int     // 节点总数
	f       int     // 最大容忍拜占庭节点数 (f)
	useBlst bool    // 是否使用 BLS（布鲁姆/聚合签名）库的标志
	AfterConsensusHandler func(round int) // <<< 新增：达成共识后的业务钩子
}

//Validator表示委员
// 核心模拟器
// ====== 导出共识结果结构体及节点类型 ======
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
	Price        float64      // <== 新增：成交价
    LeaderNode   string       // <== 新增：撮合节点
}

func NewPBFTSimulator(nodes []*node.Node, useBlst bool) *PBFTSimulator { // 构造函数：创建 PBFTSimulator 实例
	n := len(nodes)                         // 计算节点数
	f := (n - 1) / 3                        // 根据 PBFT 理论计算可容错的拜占庭个数 f
	return &PBFTSimulator{
    		nodes: nodes,
    		n: n,
    		f: f,
    		useBlst: useBlst,
    		AfterConsensusHandler: nil, // 默认无处理
    }// 返回新建实例
}

// 主节点选择，基于活跃节点
// 参数 offset 用于在发生 View Change 时跳过排名靠前的恶意节点
// ======================= 【高亮-2026-03-18】修改：优化选主逻辑，大幅降低视图切换（View Change）次数 =======================
func (s *PBFTSimulator) SelectLeader(round int, offset int) *node.Node {
	active := []*node.Node{}

	// 1. 过滤出信誉值合格的优质节点作为主节点候选池（从根源规避恶意节点）
	for _, nd := range s.nodes {
		if nd.IsActive() && nd.M() > node.MMin {
			active = append(active, nd)
		}
	}

	// 2. 如果高信誉节点为空（极端情况），则降级回退到所有活跃节点
	if len(active) == 0 {
		for _, nd := range s.nodes {
			if nd.IsActive() {
				active = append(active, nd)
			}
		}
	}

	if len(active) == 0 {
		return nil
	}

	// APBFT 核心：按信誉值M排序（这里简化为信誉值 M）
	sort.Slice(active, func(i, j int) bool {
		return active[i].M() > active[j].M()
	})

	// 【轮换逻辑】：仅在优质节点集合中取模，使得主节点始终是高信誉节点，极大概率避免触发 View Change
	idx := (round + offset) % len(active)
	return active[idx]
}

// 层级计算
func (s *PBFTSimulator) ComputeTiers() {
	arr := append([]*node.Node{}, s.nodes...)
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].Throughput > arr[j].Throughput
	})
	n := len(arr)
	top := int(float64(n) * 0.3)
	bottom := int(float64(n) * 0.3)
	if top < 1 { top = 1 }
	if bottom < 1 { bottom = 1 }
	for i, nd := range arr {
		if i < top {
			nd.Tier = node.TierHigh
		} else if i >= n-bottom {
			nd.Tier = node.TierLow
		} else {
			nd.Tier = node.TierNormal
		}
	}
}

// ======================= 【高亮-2026-03-08】新增：可选接口，兼容 Node 是否实现 SetRoundSeed(round) =======================
type roundSeedSetter interface {
	SetRoundSeed(round int)
}

// ======================= 【高亮-2026-03-14 11:15】恢复：RunRound 兼容旧版调用 =======================
func (s *PBFTSimulator) RunRound(round int, request []byte) bool {
	leader := s.SelectLeader(round, 0) // 默认不轮换执行
	// ======================= 【高亮-2026-03-22】修改：忽略返回的价格，只返回共识布尔值 =======================
	ok, _ := s.RunRoundWithLeader(round, request, leader)
	return ok
}

// ======================= 【高亮-2026-03-13】修改：RunRound 适配动态传入的 Leader =======================
// ======================= 【高亮-2026-03-22】修改：返回 (bool, float64) 以传递 KNN 计算出的成交价 =======================
func (s *PBFTSimulator) RunRoundWithLeader(round int, request []byte, leader *node.Node) (bool, float64) {
	for _, nd := range s.nodes {
    	if ss, ok := any(nd).(roundSeedSetter); ok {
    	   ss.SetRoundSeed(round)
    	}
    }

	if leader == nil {
		return false
	}

	// PRE-PREPARE
	if leader.IsMalicious && rand.Float64() < 0.5 { // 如果 leader 是恶意并以 50% 概率作恶
		fmt.Printf("Leader %d acted maliciously in pre-prepare\n", leader.ID) // 打印作恶日志
		leader.UpdateReward(false) // 更新 leader 的奖励/惩罚（作恶导致失败）
		return false // 提前返回失败
	}

	// ======================= 【高亮-2026-03-22】新增：KNN 参数初始化 =======================
	basePrice := 50.0       // 基础电价
	lineLossCoeff := 1.2     // 线损系数（元/单位距离）
	K := 30                   // K近邻数量
	var neighbors []Neighbor // 存储邻居节点信息用于 KNN 定价

	// PREPARE: 所有活跃节点签名
	var wg sync.WaitGroup // 等待组，用于并发收集签名
	var mu sync.Mutex     // 互斥锁，保护共享切片
	signatures := make([][]byte, 0, s.n) // 收集每个节点对请求的签名切片
	pubKeys := make([][]byte, 0, s.n)    // 收集每个节点的公钥切片
	signedIDs := []int{} // 用于记录参与节点
	for _, nd := range s.nodes { // 遍历所有节点
		if !nd.IsActive() { // 跳过非活跃节点
			continue
		}
    // ======================= 【高亮-2026-03-22】新增：计算距离 d 并生成本地报价 =======================
	d := calculateNodeDistance(nd.ID, leader.ID)
	quote := 150.0 + rand.Float64()*100.0 // 模拟节点的卖方报价
	neighbors = append(neighbors, Neighbor{ID: nd.ID, D: d, Quote: quote})
	wg.Add(1) // 增加等待计数
		// ======================= 【高亮-2026-03-22】修改：闭包传入 distance 变量 =======================
		go func(node *node.Node, distance float64) { // 并发签名以模拟真实网络的并行性
			defer wg.Done() // 完成时通知等待组

			// ======================= 【高亮-2026-03-22】新增：基于 KNN 距离的 Reject 逻辑 =======================
			// 核心逻辑：距离太远线损过高，按距离成正比的概率投 reject（拒绝提供签名）
			rejectProb := distance * 0.004 // 假设最大距离100时，有40%概率拒绝交易
			if rand.Float64() < rejectProb {
				return // 模拟节点投 reject，直接返回不签名
			}
			// ======================= 【高亮-2026-03-22】新增结束 =======================

			sig, err := node.Sign(request) // 节点对请求进行签名
			if err == nil && sig != nil {  // 如果签名成功
				mu.Lock()                                   // 保护共享切片
				signatures = append(signatures, sig)        // 添加签名
				pubKeys = append(pubKeys, node.PublicKey()) // 添加对应公钥
				signedIDs = append(signedIDs, node.ID)
				mu.Unlock() // 解锁
			}
		}(nd, d) // 传入节点和距离
	}
	wg.Wait() // 等待所有并发签名完成

	// leader 聚合
	//============原声明为aggSig, _ := leader.bls.AggregateSignatures(signatures)，目的是对leader节点使用BLS聚合签名，但是目前bls变量已在node目录下，因此要修改声明为aggSig, _ := leader.AggregateSignatures(signatures) ，同时在node目录下的相关文件中封装ggregateSignatures（）
	aggSig, _ := leader.AggregateSignatures(signatures) // 需要 node.Node 提供 AggregateSignatures()

	// leader 验证聚合签名
	// ========原因同上，将声明修改为ok, _ := leader.VerifyAggregate(pubKeys, request, aggSig) ===========
	ok, _ := leader.VerifyAggregate(pubKeys, request, aggSig) // 需要 node.Node 提供 VerifyAggregate()
	if !ok { // 如果验证失败
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
			// ======================= 【高亮-2026-03-08】Fix：不再访问 nd.bls（未导出），改用 nd.PublicKey() =======================
			commitPubKeys = append(commitPubKeys, nd.PublicKey())// 收集公钥
		}
	}

    //=============由于bls是node目录下的变量，所以需要将aggCommitSig, _ := leader.bls.AggregateSignatures(commitSigs)改为aggCommitSig, _ := leader.AggregateSignatures(commitSigs)==========
    //=============将ok2, _ := leader.bls.VerifyAggregate(commitPubKeys, aggSig, aggCommitSig)改为ok2, _ := leader.VerifyAggregate(commitPubKeys, aggSig, aggCommitSig)
	aggCommitSig, _ := leader.AggregateSignatures(commitSigs) // leader 聚合 commit 签名
	ok2, _ := leader.VerifyAggregate(commitPubKeys, aggSig, aggCommitSig)// 验证聚合的 commit 签名（以 aggSig 作为消息）
	if !ok2 { // 如果 commit 阶段验证失败
		fmt.Println("Aggregate verification failed in commit phase") // 打印错误信息
		leader.UpdateReward(false) // 更新奖励为失败
		return false // 共识失败
	}

	// 判断阈值
	quorum := int(float64(s.n) * PrepareQuorumMultiplier)
	if len(commitSigs) >= quorum  { // 如果 commit 签名数达到阈值（基于 PrepareQuorumMultiplier）
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
        // ====== 补充2：共识后业务钩子调用 ======
    	if s.AfterConsensusHandler != nil {
    		s.AfterConsensusHandler(round)
    	}
        // ======================= 【高亮-2026-03-22】新增：KNN 定价核心逻辑 =======================
        // 按距离 d 对所有参与的邻居节点进行升序排序，提取最近的 K 个邻居
        sort.Slice(neighbors, func(i, j int) bool {
        	return neighbors[i].D < neighbors[j].D
        })

        knnCount := K
        	if len(neighbors) < K {
        		knnCount = len(neighbors)
        	}

        sumQuote := 0.0
        sumDistance := 0.0
        for i := 0; i < knnCount; i++ {
        	sumQuote += neighbors[i].Quote
        	sumDistance += neighbors[i].D
        	}

        avgQuote := sumQuote / float64(knnCount)       // 最近 K 个卖方的平均报价
        avgDistance := sumDistance / float64(knnCount) // 最近 K 个节点的平均距离（KNN距离）

       	// 最终撮合价格 = 基础电价 + K邻近平均报价 + KNN平均距离 * 线损系数
        finalPrice := basePrice + avgQuote + (avgDistance * lineLossCoeff)

     // 【控制台输出】
		fmt.Printf("\n>>>>>> [APBFT 共识达成 | 轮次 %d] <<<<<<\n", round)
		fmt.Printf("├─ 主节点信息: ID=%d | 信誉值(m)=%.d | 层级(Tier)=%d | 吞吐量=%.2f\n",
			leader.ID, leader.M(), leader.Tier, leader.Throughput)
		// ======================= 【高亮-2026-03-22】修改：打印 KNN 定价信息 =======================
		fmt.Printf("├─ KNN 定价: 基础价=%.2f | K邻近均报价=%.2f | KNN均距=%.2f\n", basePrice, avgQuote, avgDistance)
		fmt.Printf("├─ 共识详情: 最终成交价=%.2f | 参与度=%d/%d (法定人数:%d)\n",
			finalPrice, len(signatures), s.n, quorum)
		// ======================= 【高亮-2026-03-22】修改结束 =======================
		fmt.Printf("└─ 参与节点列表: %v\n", signedIDs)

		for _, nd := range s.nodes {
			nd.UpdateReward(true)
		}
		// ======================= 【高亮-2026-03-22】修改：返回 true 和 KNN 计算出的 finalPrice =======================
		return true, finalPrice // 返回共识成功及最终价格
	} else {
		fmt.Println("Not enough commit signatures; consensus failed") // 未达到阈值，打印失败信息
		for _, nd := range s.nodes {                                  // 对所有节点应用失败的奖励更新
			nd.UpdateReward(false)
		}
		return false, 0 // 返回共识失败
	}
}

// ======================= 【高亮-2026-03-08】新增：方案1入口（PBFT1 只信 specs；nodepool 是唯一真相） =======================
// - round：轮次
// - txId/amount：业务模拟字段
// - specs：由 node.NewPool(round, ...) 生成的公共节点规格（四算法共用）
func RunAPBFTWithRoundAndSpecs(round int, txId string, amount int, specs []node.NodeSpec) PBFTResult {
    useBlst := true // 你原逻辑写死 true；如果要无 blst 环境跑，Node 里会自动用 stub（你之前已做 NewBlstBLS stub 兼容）

	// ========== 构建节点池：把 isMal 写入节点 ==========
	nodes := make([]*node.Node, 0, len(specs))
	for _, sp := range specs {
		nd := node.NewNode(sp.ID, sp.Throughput, sp.IsMalicious, useBlst) // [2026-03-08] 用 node.NewNode
		// 可选：如果要把 Active 同步进 Node，可以在 Node 上加 SetActive/或构造时设置；这里先保持你原 Node 逻辑
		nodes = append(nodes, nd)
	}

	sim := NewPBFTSimulator(nodes, true)
	sim.ComputeTiers()

    // 【主节点轮换算法逻辑】
	var finalLeader *node.Node
	var success bool // 【修复点】：使用 success 命名
	var finalPrice float64   // ======================= 【高亮-2026-03-22】新增：接收 KNN 计算的新价格
	viewOffset := 0
	maxViewChange := 5 // 最多允许轮换 5 个备份节点

	for viewOffset < maxViewChange {
	leader := sim.SelectLeader(round, viewOffset)
	if leader == nil { break }

	// 【关键拦截】：如果选中的是恶意节点或信誉度过低，主动跳过该节点（即轮换）
    // 【视图转换触发打印】
	if leader.IsMalicious || leader.M() <= node.MMin {
        fmt.Printf("[View Change] 轮次 %d: 节点 %d (m=%.d, Malicious=%v) 不可信，触发视图转换...\n", round, leader.ID, leader.M(), leader.IsMalicious)
		viewOffset++
		continue
	}

	finalLeader = leader
	//success = sim.RunRoundWithLeader(round, []byte(txId), leader)
	// ======================= 【高亮-2026-03-22】修改：接收成功状态和基于 KNN 计算的定价 =======================
    success, finalPrice = sim.RunRoundWithLeader(round, []byte(txId), leader)
	break
	}

	status := "已确认"
	reason := ""
	if !success {
       status = "失败"
       reason = "apbft consensus failed"
       // 失败时回退给个默认价格
       seed := int64(20260307 + round)
       rngObj := rand.New(rand.NewSource(seed))  // 换了个名字确保清晰
       finalPrice = 500 + rngObj.Float64()*50    // 确保真正使用了这个随机数发生器
	}

    // 【修复点】：显式定义并初始化 leaderNodeName
    leaderNodeName := "None"
    if finalLeader != nil {
		leaderNodeName = finalLeader.String()
	}

	// 建议：BlockHeight 直接等于 round，前端展示更一致
	return PBFTResult{
		TxId:         txId,
		Status:       status,
		Consensus:    "pbft",
		BlockHeight:  round,
		Timestamp:    time.Now(),
		Validators:   nil,
		FailedReason: reason,
		// ======================= 【高亮-2026-03-22】修改：赋值新的 KNN 定价 =======================
        Price:        finalPrice,
		LeaderNode:   leaderNodeName,// 使用 leaderNode（避免未使用 & 避免 leader nil 时 panic）
	}
}

func RunAPBFT(txId string, amount int) PBFTResult {
	specs := node.NewPool(1, node.FixedNumNodes, node.FixedMaliciousRatio)
	return RunAPBFTWithRoundAndSpecs(1, txId, amount, specs)
}