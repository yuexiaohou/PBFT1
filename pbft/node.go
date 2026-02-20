package pbft

import (
	"fmt"      // 格式化输出、错误信息等
	"math/rand"// 用于模拟恶意节点的随机行为选择
	"sync"     // 提供互斥锁等并发原语
	"time"     // 用于模拟延迟和时间相关操作
)

// Tier 表示节点的等级类型（例如用于优先级或奖励策略）
type Tier int

const (
	TierLow Tier = iota    // 低等级
	TierNormal             // 正常等级
	TierHigh               // 高等级
)

// Node 表示网络中的一个节点，包含身份、奖励、是否恶意、吞吐量等信息
type Node struct {
	ID         int     // 节点唯一标识符
	m          int     // 奖励值或信誉值（注释中原为“奖励值”）
	IsMalicious bool   // 是否被标记为恶意节点
	Throughput float64 // 吞吐量指标（例如 TPS），用于模拟签名延迟等
	Tier       Tier     // 节点等级（TierLow/TierNormal/TierHigh）

	bls BLS        // 抽象的 BLS 签名实现接口（可替换为不同实现）

	mu     sync.Mutex // 保护以下字段的互斥锁
	active bool       // 节点是否处于激活状态（当 m < MMin 则置为 inactive）
}

// NewNode 创建并返回一个新的 Node 实例
// 参数：id 节点 id，throughput 吞吐量，isMalicious 是否恶意，useBlst 是否使用 blst 实现
func NewNode(id int, throughput float64, isMalicious bool, useBlst bool) *Node {
	var bl BLS
	if useBlst {
		// 若要求使用 blst 实现，则创建对应实现
		bl = NewBlstBLS(id)
	} else {
		// 否则使用简单的 BLS stub（可能用于测试或模拟）
		bl = NewSimpleBLSStub(id)
	}
	return &Node{
		ID:         id,            // 设置节点 ID
		m:          InitialM,      // 初始奖励/信誉值（来自全局常量 InitialM）
		IsMalicious: isMalicious,  // 是否恶意
		Throughput: throughput,    // 吞吐量
		Tier:       TierNormal,    // 默认等级为 Normal
		bls:        bl,            // BLS 签名实现
		active:     true,          // 默认激活
	}
}

// String 返回节点的可读字符串表示，便于调试和日志记录
func (n *Node) String() string {
	return fmt.Sprintf("Node-%02d(m=%d, tier=%v, tp=%.2f, mal=%v, active=%v)", n.ID, n.m, n.Tier, n.Throughput, n.IsMalicious, n.active)
}

// Sign 对给定消息进行签名，返回签名字节切片或错误。
// 该方法会根据节点是否恶意模拟不同的行为与延迟。
func (n *Node) Sign(message []byte) ([]byte, error) {
	if n.IsMalicious {
		// 对于被标记为恶意的节点，随机选择三种行为之一进行模拟：
		// 0: 不签名（延迟后返回错误）
		// 1: 返回错误签名（伪造签名）
		// 2: 偶尔正常签名（调用真实签名实现）
		choice := rand.Intn(3)
		switch choice {
		case 0:
			// 不签：模拟一定的延迟后返回错误，表示恶意节点拒绝签名
			time.Sleep(50 * time.Millisecond)
			return nil, fmt.Errorf("malicious: not signing")
		case 1:
			// 错签：返回一个格式化的“坏签名”字节数组（非真实签名格式）
			return []byte(fmt.Sprintf("bad-sign-node-%02d", n.ID)), nil
		default:
			// 偶尔正常签名：调用底层 bls 签名实现
			return n.bls.Sign(message)
		}
	}
	// 非恶意（正常）节点：签名延迟与吞吐率相关，吞吐量越高模拟延迟越小
	// 通过 1000.0 / Throughput 计算以毫秒为单位的延迟（仅为模拟）
	delay := time.Duration(1000.0/n.Throughput) * time.Millisecond
	// 限制最大延迟为 200ms，避免过长阻塞
	if delay > 200*time.Millisecond {
		delay = 200 * time.Millisecond
	}
	// 模拟处理延迟
	time.Sleep(delay)
	// 调用底层 bls 实现进行真实签名并返回
	return n.bls.Sign(message)
}

// UpdateReward 根据操作是否成功更新节点的奖励/信誉值 m。
// success 为 true 则增加奖励，否则减少；同时处理上下界和激活状态。
func (n *Node) UpdateReward(success bool) {
	n.mu.Lock()         // 获取互斥锁，保护对 m 和 active 的并发访问
	defer n.mu.Unlock() // 延迟释放锁

	if success {
		// 操作成功则增加奖励值
		n.m++
	} else {
		// 操作失败则减少奖励值
		n.m--
	}
	// 如果 m 超过 MMax，为防止中心化将 m 重置为初始值 InitialM
	if n.m > MMax {
		n.m = InitialM // 防止某些节点过度积累导致中心化风险
	}
	// 若 m 低于 MMin，则将节点标记为 inactive（被排除）
	if n.m < MMin {
		n.active = false // 判为恶意并排除（或失去资格）
	}
}

// IsActive 返回节点当前是否处于激活状态（线程安全）
func (n *Node) IsActive() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.active
}
