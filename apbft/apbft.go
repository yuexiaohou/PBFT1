package apbft // 定义包为 main，表示此文件属于可独立运行的程序

import ( // 导入必要的标准库包
	"fmt"       // 格式化 I/O，用于打印日志
	"math/rand" // 随机数，用于模拟恶意行为概率
	"sort"      // 排序，用于对节点排序（例如选择 leader、计算 tiers）
	"sync"      // 并发原语，用于等待并保护共享切片
	"time"
	"PBFT1/node"
)

// 全局日志开关
var EnableLogs = true

// ======================= 【修复多轮状态丢失】 =======================
// 创建一个全局持久化的 Simulator 单例，用于跨轮次保留 Q-Table 和 Reputation
var GlobalSim *PBFTSimulator
var simMutex sync.Mutex

// ======================= 【高亮-2026-03-29】新增一：开放系统级参数 =======================
// 暴露给外部 (如 find_k.go) 用于动态调参寻优。
// 注意：在正式生产环境或寻优结束后，建议将 GlobalK 替换为 config.go 中的常量 OptimalKNNValue 以保证共识绝对确定性。
// 经过蒙特卡洛寻优，确定了系统的最优纳什均衡 K 值为 5。
// 将其硬编码为常量，保证分布式共识结果的确定性与不可篡改性。
const OptimalKNNValue = 28

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

// ======================= 【高亮-2026-03-29】新增：强化学习 Q-Learning 核心结构 =======================
// QState 定义节点状态空间 (State)
type QState struct {
	SuccessRate int // 节点的历史成功率: 0(<50%), 1(50-80%), 2(>80%)
	Tier        int // 当前网络的吞吐情况: 映射 nd.Tier (0:High, 1:Normal, 2:Low)
	Role        int // 当前节点的身份: 0(Follower), 1(Leader)
}

// NodeQAgent 为每个节点维护一个 Q-Learning 智能体
type NodeQAgent struct {
	QTable        map[QState]map[int]float64 // 记录 Q(s, a) 值
	TotalRounds   int                        // 总参与轮次
	SuccessRounds int                        // 成功参与共识的轮次
	Reputation    int                        // 当前信誉值 R (取代原有的 m)
}

// 简化 PBFT 模拟器（PRE-PREPARE / PREPARE / COMMIT）
// 定义 PBFT 模拟器的结构体，封装节点集合与参数
type PBFTSimulator struct {
	nodes                 []*node.Node // 节点切片，表示参与共识的所有节点
	n                     int          // 节点总数
	f                     int          // 最大容忍拜占庭节点数 (f)
	useBlst               bool         // 是否使用 BLS（布鲁姆/聚合签名）库的标志
	AfterConsensusHandler func(round int) // <<< 新增：达成共识后的业务钩子
    // ======================= 【高亮-2026-03-29】新增：强化学习智能体映射 =======================
	QAgents               map[int]*NodeQAgent // 存储所有节点的智能体
}

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
	Price        float64 // <== 新增：成交价
	LeaderNode   string  // <== 新增：撮合节点
}

func NewPBFTSimulator(nodes []*node.Node, useBlst bool) *PBFTSimulator { // 构造函数：创建 PBFTSimulator 实例
	n := len(nodes)  // 计算节点数
	f := (n - 1) / 3 // 根据 PBFT 理论计算可容错的拜占庭个数 f
    sim := &PBFTSimulator{
		nodes:                 nodes,
		n:                     n,
		f:                     f,
		useBlst:               useBlst,
		AfterConsensusHandler: nil, // 默认无处理
		QAgents:               make(map[int]*NodeQAgent),
	}

	// ======================= 【高亮-2026-03-29】新增：初始化所有节点的 Q-Agent =======================
	for _, nd := range nodes {
		sim.QAgents[nd.ID] = &NodeQAgent{
			QTable:     make(map[QState]map[int]float64),
			Reputation: 50, // 信誉值初始分为 50
		}
	}
	return sim // 返回新建实例
}

// ======================= 【高亮-2026-03-29】新增：Q-Learning 工具函数 =======================
// getQState 获取节点当前环境状态
func (s *PBFTSimulator) getQState(nd *node.Node, isLeader bool) QState {
	agent := s.QAgents[nd.ID]
	sr := 0
	if agent.TotalRounds > 0 {
		rate := float64(agent.SuccessRounds) / float64(agent.TotalRounds)
		if rate >= 0.8 {
			sr = 2
		} else if rate >= 0.5 {
			sr = 1
		}
	}

	role := 0
	if isLeader {
		role = 1
	}

	return QState{
		SuccessRate: sr,
		Tier:        int(nd.Tier),
		Role:        role,
	}
}

// chooseAction 基于 epsilon-greedy 策略为节点选择动作 (Action)
func (s *PBFTSimulator) chooseAction(nd *node.Node, state QState, distance float64) int {
	agent := s.QAgents[nd.ID]
	epsilon := 0.1 // 10% 概率进行随机探索
	isMal := nd.IsMalicious

	// a1=0 (诚实), a2=1 (拒绝), a3=2 (作恶)
	maxActions := 2 // 诚实节点只探索动作 a1 和 a2
	if isMal {
		maxActions = 3 // 拜占庭节点可以探索动作 a3
	}

	// 探索 (Exploration)
	if rand.Float64() < epsilon {
		return rand.Intn(maxActions)
	}

	// 贪婪策略 (Exploitation) 依据 Q Table 选择最佳动作
	actions := agent.QTable[state]
	if actions == nil {
		agent.QTable[state] = make(map[int]float64)
		actions = agent.QTable[state]
	}

	bestA := 0
	bestQ := -999999.0
	for act := 0; act < maxActions; act++ {
		if val, exists := actions[act]; exists {
			if val > bestQ {
				bestQ = val
				bestA = act
			}
		} else {
			if 0.0 > bestQ {
				bestQ = 0.0
				bestA = act
			}
		}
	}
	return bestA
}

// updateQTable 采用 Q-learning 更新公式更新表
func (s *PBFTSimulator) updateQTable(nodeID int, state QState, action int, reward float64) {
	agent := s.QAgents[nodeID]
	alpha := 0.1 // 学习率
	gamma := 0.9 // 折扣因子

	if agent.QTable[state] == nil {
		agent.QTable[state] = make(map[int]float64)
	}

	oldQ := agent.QTable[state][action]
	maxNextQ := 0.0
	for _, q := range agent.QTable[state] {
		if q > maxNextQ {
			maxNextQ = q
		}
	}

	// Q(s,a) = Q(s,a) + alpha * [R + gamma * max Q(s',a') - Q(s,a)]
	newQ := oldQ + alpha*(reward+gamma*maxNextQ-oldQ)
	agent.QTable[state][action] = newQ
}
// 主节点选择，基于活跃节点
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

    // APBFT 核心：按强化学习积累的 Reputation (R) 从大到小排序
	sort.Slice(active, func(i, j int) bool {
		return s.QAgents[active[i].ID].Reputation > s.QAgents[active[j].ID].Reputation
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
	if top < 1 {
		top = 1
	}
	if bottom < 1 {
		bottom = 1
	}
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

type roundSeedSetter interface {
	SetRoundSeed(round int)
}

func (s *PBFTSimulator) RunRound(round int, request []byte) bool {
	leader := s.SelectLeader(round, 0) // 默认不轮换执行
	ok, _ := s.RunRoundWithLeader(round, request, leader)
	return ok
}

// 共识流程(本轮)
func (s *PBFTSimulator) RunRoundWithLeader(round int, request []byte, leader *node.Node) (bool, float64) {
	for _, nd := range s.nodes {
		if ss, ok := any(nd).(roundSeedSetter); ok {
			ss.SetRoundSeed(round)
		}
	}

	if leader == nil {
		// ======================= 【修复报错点】补充返回值 0 =======================
		return false, 0
	}

    // ======================= 【高亮-2026-03-29】修改：Q-Learning PRE-PREPARE 阶段动作选择 =======================
	leaderState := s.getQState(leader, true)
	leaderAction := s.chooseAction(leader, leaderState, 0.0)
	s.QAgents[leader.ID].TotalRounds++

	if leaderAction == 2 { // a3: Leader 根据经验策略决定作恶
		if EnableLogs {
			fmt.Printf("Leader %d acted maliciously (Q-learning a3) in pre-prepare\n", leader.ID)
		}
		// 如果被发现作恶导致共识失败：R = -20（重罚）
		s.updateQTable(leader.ID, leaderState, leaderAction, -20.0)
		s.QAgents[leader.ID].Reputation -= 20
		return false, 0
	}

	// 【KNN 参数初始化】
	basePrice := 20.0       // 基础电价
	lineLossCoeff := 0.2     // 线损系数（元/单位距离）
	var neighbors []Neighbor // 存储邻居节点信息用于 KNN 定价

	// PREPARE: 所有活跃节点签名
	var wg sync.WaitGroup                // 等待组，用于并发收集签名
	var mu sync.Mutex                    // 互斥锁，保护共享切片
	signatures := make([][]byte, 0, s.n) // 收集每个节点对请求的签名切片
	pubKeys := make([][]byte, 0, s.n)    // 收集每个节点的公钥切片
	signedIDs := []int{}                 // 用于记录参与节点

// ======================= 【高亮-2026-03-29】新增：记录所有节点的动作，用于结算奖励 =======================
	nodeStates := make(map[int]QState)
	nodeActions := make(map[int]int)

	for _, nd := range s.nodes { // 遍历所有节点
		if !nd.IsActive() { // 跳过非活跃节点
			continue
		}

		// 计算距离 d 并生成本地报价
		d := calculateNodeDistance(nd.ID, leader.ID)

        var state QState
		var action int

		if nd.ID == leader.ID {
			state = leaderState
			action = 0 // Leader 已经在上面决定不作恶，强制归类为 a1 诚实
		} else {
			state = s.getQState(nd, false)
			action = s.chooseAction(nd, state, d)
			s.QAgents[nd.ID].TotalRounds++
		}

		nodeStates[nd.ID] = state
		nodeActions[nd.ID] = action

		// ======================= 【高亮-2026-03-29】修改：执行 a2 (拒绝投票) 动作 =======================
		if action == 1 { // a2
			// 如果因为物理距离过远拒绝投票（合理止损）：R = +2（节省了计算资源）
			s.updateQTable(nd.ID, state, action, 2.0)
			s.QAgents[nd.ID].Reputation += 2
			continue
		}

        // ======================= 【高亮-2026-03-29】修改：执行 a1/a3 动作及报价 =======================
		var quote float64
		if action == 2 { // a3
			// 恶意节点故意报极高的垄断电价，试图破坏全网指导价
			quote = 60.0 + rand.Float64()*30.0
		} else { // a1
			// 诚实节点的正常低廉报价
			quote = 15.0 + rand.Float64()*10.0
		}

		neighbors = append(neighbors, Neighbor{ID: nd.ID, D: d, Quote: quote})
		wg.Add(1) // 增加等待计数

        go func(n *node.Node, act int) { // 并发签名以模拟真实网的并行性
			defer wg.Done() // 完成时通知等待组

			sig, err := n.Sign(request) // 节点对请求进行签名
			if err == nil && sig != nil {  // 如果签名成功
				mu.Lock()                                   // 保护共享切片
				signatures = append(signatures, sig)        // 添加签名
				pubKeys = append(pubKeys, n.PublicKey())    // 添加对应公钥
				signedIDs = append(signedIDs, n.ID)
				mu.Unlock() // 解锁
			}
		}(nd, action) // 传入节点
	}
	wg.Wait() // 等待所有并发签名完成

	// leader 聚合
	aggSig, _ := leader.AggregateSignatures(signatures) // 需要 node.Node 提供 AggregateSignatures()

	// leader 验证聚合签名
	ok, _ := leader.VerifyAggregate(pubKeys, request, aggSig) // 需要 node.Node 提供 VerifyAggregate()
	if !ok {                                                  // 如果验证失败
        // ======================= 【高亮-2026-03-29】修改：如果被发现作恶导致共识失败，重罚 Leader =======================
		s.updateQTable(leader.ID, leaderState, leaderAction, -20.0)
		s.QAgents[leader.ID].Reputation -= 20
		return false, 0
	}

	// COMMIT: 节点对聚合签名再次签名（模拟）
	commitSigs := make([][]byte, 0)    // 收集 commit 阶段的签名
	commitPubKeys := make([][]byte, 0) // 收集 commit 阶段的公钥
    for _, nd := range s.nodes {       // 遍历所有节点
		if !nd.IsActive() { // 跳过非活跃节点
			continue
		}
		if nodeActions[nd.ID] == 1 { // 如果之前动作是拒绝(a2)，也不提交 commit
			continue
		}

		sig, err := nd.Sign(aggSig) // 节点对聚合签名再签一次，作为 commit 的签名（模拟）
		if err == nil && sig != nil { // 如果签名成功
			commitSigs = append(commitSigs, sig) // 收集 commit 签名
			commitPubKeys = append(commitPubKeys, nd.PublicKey()) // 收集公钥
		}
	}

    aggCommitSig, _ := leader.AggregateSignatures(commitSigs)           // leader 聚合 commit 签名
	ok2, _ := leader.VerifyAggregate(commitPubKeys, aggSig, aggCommitSig) // 验证聚合的 commit 签名（以 aggSig 作为消息）
	if !ok2 {                                                            // 如果 commit 阶段验证失败
		if EnableLogs {
			fmt.Println("Aggregate verification failed in commit phase") // 打印错误信息
		}
		// 重罚 Leader
		s.updateQTable(leader.ID, leaderState, leaderAction, -20.0)
		s.QAgents[leader.ID].Reputation -= 20
		return false, 0
	}

    // 判断阈值
	quorum := int(float64(s.n) * PrepareQuorumMultiplier)
	if len(commitSigs) >= quorum { // 如果 commit 签名数达到阈值
		if EnableLogs {
			fmt.Println("Consensus achieved in this round") // 打印达成共识
		}

		successIDs := map[int]bool{}       // 创建映射以记录哪些节点参与了成功的 commit
		for _, pk := range commitPubKeys { // 遍历 commit 的公钥切片
			var id int
			fmt.Sscanf(string(pk), "PK-node-%02d", &id) // 通过格式化字符串解析出节点 ID（这里是模拟）
			successIDs[id] = true                       // 标记该 id 为成功参与者
		}

		// ======================= 【高亮-2026-03-29】修改：Q-Learning 成功奖励分配 =======================
		for _, nd := range s.nodes {
			act, participated := nodeActions[nd.ID]
			if !participated {
				continue
			}
			state := nodeStates[nd.ID]

			if successIDs[nd.ID] { // 如果该节点在成功列表中
				if act == 0 { // a1: 诚实投票且共识成功
					s.QAgents[nd.ID].SuccessRounds++
					s.updateQTable(nd.ID, state, act, 10.0) // R = +10
					s.QAgents[nd.ID].Reputation += 10
				} else if act == 2 { // a3: 尝试作恶但由于被多数派稀释，共识依然成功
					s.QAgents[nd.ID].SuccessRounds++
					s.updateQTable(nd.ID, state, act, 0.0) // 无奖励
				}
			} else {
				if act == 2 { // a3: 恶意签名被剔除
					s.updateQTable(nd.ID, state, act, -20.0) // R = -20
					s.QAgents[nd.ID].Reputation -= 20
				}
			}
		}

		if s.AfterConsensusHandler != nil {
			s.AfterConsensusHandler(round)
		}

		// 【KNN 定价核心逻辑】
		// 按距离 d 对所有参与的邻居节点进行升序排序，提取最近的 K 个邻居
		sort.Slice(neighbors, func(i, j int) bool {
			return neighbors[i].D < neighbors[j].D
		})

		// ======================= 【高亮-2026-03-29】使用硬编码的常量 OptimalKNNValue =======================
		knnCount := OptimalKNNValue
		if len(neighbors) < OptimalKNNValue {
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

    if EnableLogs {
			fmt.Printf("\n>>>>>> [APBFT 共识达成 | 轮次 %d] <<<<<<\n", round)
			// ======================= 【高亮-2026-03-29】修改：打印最新的 Q信誉值(R) =======================
			fmt.Printf("├─ 主节点信息: ID=%d | Q信誉值(R)=%d | 层级(Tier)=%d | 吞吐量=%.2f\n",
				leader.ID, s.QAgents[leader.ID].Reputation, leader.Tier, leader.Throughput)
			fmt.Printf("├─ KNN 定价: 基础价=%.2f | K邻近均报价=%.2f | KNN均距=%.2f\n", basePrice, avgQuote, avgDistance)
			fmt.Printf("├─ 共识详情: 最终成交价=%.2f | 参与度=%d/%d (法定人数:%d)\n",
				finalPrice, len(signatures), s.n, quorum)
			fmt.Printf("└─ 参与节点列表: %v\n", signedIDs)
		}

		return true, finalPrice // 返回共识成功及最终价格
	} else {
		if EnableLogs {
			fmt.Println("Not enough commit signatures; consensus failed") // 未达到阈值，打印失败信息
		}
		// ======================= 【高亮-2026-03-29】修改：Q-Learning 失败惩罚分配 =======================
		for _, nd := range s.nodes {
			act, participated := nodeActions[nd.ID]
			if !participated {
				continue
			}
			state := nodeStates[nd.ID]

			if act == 2 { // a3: 发现作恶导致共识失败，重罚
				s.updateQTable(nd.ID, state, act, -20.0) // R = -20
				s.QAgents[nd.ID].Reputation -= 20
			} else if act == 0 { // a1: 诚实节点被连累，无功无过
				s.updateQTable(nd.ID, state, act, 0.0)
			}
		}
		// ======================= 【修复报错点】补充返回值 0 =======================
		return false, 0
	}
}

//==================修改RunAPBFTWithRoundAndSpecs函数更正为RunPersistentAPBFT，将原函数中的node:=make([]*node.Node, 0, len(specs)),改为只有第一次才进行初始化节点池，其余时候都使用第一轮创建的节点池=======================
func RunPersistentAPBFT(round int, txId string, amount int, specs []node.NodeSpec) PBFTResult {
	simMutex.Lock()
	defer simMutex.Unlock()

	// 只有第一轮，或者单例为空时，才初始化一次
	if GlobalSim == nil || round == 1 {
		nodes := make([]*node.Node, 0, len(specs))
		for _, sp := range specs {
			nd := node.NewNode(sp.ID, sp.Throughput, sp.IsMalicious, true)
			nodes = append(nodes, nd)
		}
		GlobalSim = NewPBFTSimulator(nodes, true)
		GlobalSim.ComputeTiers()
	} else {
		// 跨轮次更新恶意状态（因为 specs 可能会让不同的节点临时变成恶意）
		// 但我们保留 nd 实例，这样就保留了信誉值和 Q表
		for i, sp := range specs {
			GlobalSim.nodes[i].IsMalicious = sp.IsMalicious
			GlobalSim.nodes[i].Throughput = sp.Throughput
		}
		GlobalSim.ComputeTiers()
	}

	var finalLeader *node.Node
	var success bool
	var finalPrice float64
	viewOffset := 0
	maxViewChange := 5

	for viewOffset < maxViewChange {
		leader := GlobalSim.SelectLeader(round, viewOffset)
		if leader == nil {
			break
		}

		if leader.IsMalicious || GlobalSim.QAgents[leader.ID].Reputation <= 0 {
			if EnableLogs {
				fmt.Printf("[View Change] 轮次 %d: 节点 %d (R=%d, Malicious=%v) 不可信，触发视图转换...\n", round, leader.ID, GlobalSim.QAgents[leader.ID].Reputation, leader.IsMalicious)
			}
			viewOffset++
			continue
		}

		finalLeader = leader
		success, finalPrice = GlobalSim.RunRoundWithLeader(round, []byte(txId), leader)
		break
	}

	status := "已确认"
	reason := ""
	if !success {
		status = "失败"
		reason = "apbft consensus failed"
		seed := int64(20260307 + round)
		rngObj := rand.New(rand.NewSource(seed))
		finalPrice = 45 + rngObj.Float64()*15
	}

	leaderNodeName := "None"
	if finalLeader != nil {
		leaderNodeName = fmt.Sprintf("Node-%02d(R=%d, tier=%d, tp=%.2f, mal=%v)",
			finalLeader.ID, GlobalSim.QAgents[finalLeader.ID].Reputation, finalLeader.Tier, finalLeader.Throughput, finalLeader.IsMalicious)
	}

	return PBFTResult{
		TxId:         txId,
		Status:       status,
		Consensus:    "pbft",
		BlockHeight:  round,
		Timestamp:    time.Now(),
		Validators:   nil,
		FailedReason: reason,
		Price:        finalPrice,
		LeaderNode:   leaderNodeName,
	}
}

func RunAPBFT(txId string, amount int) PBFTResult {
	specs := node.NewPool(1, node.FixedNumNodes, node.FixedMaliciousRatio)
	return RunPersistentAPBFT(1, txId, amount, specs)
}