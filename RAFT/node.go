package raft

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

// RAFT 节点结构体
type Node struct {
	ID        int
	IsLeader  bool       // 是否为主节点
	Term      int        // 当前选举term
	Active    bool       // 节点是否激活（宕机/重启影响）
	Tier      Tier
	mu        sync.Mutex // 线程安全
}

// 新建 RAFT 节点
func NewNode(id int, term int) *Node {
	return &Node{
		ID:       id,
		IsLeader: false,
		Term:     term,
		Active:   true,
		Tier:     TierNormal,
	}
}

func (n *Node) String() string {
	return fmt.Sprintf("Node-%02d(term=%d, leader=%v, tier=%v, active=%v)",
		n.ID, n.Term, n.IsLeader, n.Tier, n.Active)
}

func (n *Node) IsActive() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.Active
}

// 切换主节点
func (n *Node) SetLeader(isLeader bool, term int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.IsLeader = isLeader
	n.Term = term
}