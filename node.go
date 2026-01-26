package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Tier int

const (
	TierLow Tier = iota
	TierNormal
	TierHigh
)

type Node struct {
	ID         int
	m          int     // 奖励值
	IsMalicious bool
	Throughput float64 // 吞吐量指标（例如 TPS）
	Tier       Tier

	bls BLS

	mu     sync.Mutex
	active bool // 是否被排除（m < mmin 则 inactive）
}

func NewNode(id int, throughput float64, isMalicious bool, useBlst bool) *Node {
	var bl BLS
	if useBlst {
		bl = NewBlstBLS(id)
	} else {
		bl = NewSimpleBLSStub(id)
	}
	return &Node{
		ID:         id,
		m:          InitialM,
		IsMalicious: isMalicious,
		Throughput: throughput,
		Tier:       TierNormal,
		bls:        bl,
		active:     true,
	}
}

func (n *Node) String() string {
	return fmt.Sprintf("Node-%02d(m=%d, tier=%v, tp=%.2f, mal=%v, active=%v)", n.ID, n.m, n.Tier, n.Throughput, n.IsMalicious, n.active)
}

func (n *Node) Sign(message []byte) ([]byte, error) {
	if n.IsMalicious {
		choice := rand.Intn(3)
		switch choice {
		case 0:
			// 不签
			time.Sleep(50 * time.Millisecond)
			return nil, fmt.Errorf("malicious: not signing")
		case 1:
			// 错签
			return []byte(fmt.Sprintf("bad-sign-node-%02d", n.ID)), nil
		default:
			// 偶尔正常签名
			return n.bls.Sign(message)
		}
	}
	// 正常节点：延迟与吞吐率有关（仅模拟）
	delay := time.Duration(1000.0/n.Throughput) * time.Millisecond
	if delay > 200*time.Millisecond {
		delay = 200 * time.Millisecond
	}
	time.Sleep(delay)
	return n.bls.Sign(message)
}

func (n *Node) UpdateReward(success bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if success {
		n.m++
	} else {
		n.m--
	}
	if n.m > MMax {
		n.m = InitialM // 防止中心化
	}
	if n.m < MMin {
		n.active = false // 判为恶意并排除
	}
}

func (n *Node) IsActive() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.active
}
