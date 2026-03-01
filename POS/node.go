package pos

import (
	"fmt"
	"sync"
)

type Tier int

const (
	TierLow Tier = iota
	TierNormal
	TierHigh
)

// POS 节点结构体
type Node struct {
	ID          int     // 节点唯一标识符
	Stake       float64 // 权益（随机/规则分配）
	Active      bool    // 是否激活
	Tier        Tier    // 节点层级
	mu          sync.Mutex
}

// 构造新节点（stake由外部分配）
func NewNode(id int, stake float64) *Node {
	return &Node{
		ID:     id,
		Stake:  stake,
		Active: true,
		Tier:   TierNormal,
	}
}

// POS 节点可读字符串
func (n *Node) String() string {
	return fmt.Sprintf("Node-%02d(stake=%.2f, tier=%v, active=%v)", n.ID, n.Stake, n.Tier, n.Active)
}

// POS节点激活判定
func (n *Node) IsActive() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.Active
}

// 节点失效模拟
func (n *Node) SetInactive() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Active = false
}