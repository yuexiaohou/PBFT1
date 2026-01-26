package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
)

// 简化 PBFT 模拟器（PRE-PREPARE / PREPARE / COMMIT）
type PBFTSimulator struct {
	nodes   []*Node
	n       int
	f       int
	useBlst bool
}

func NewPBFTSimulator(nodes []*Node, useBlst bool) *PBFTSimulator {
	n := len(nodes)
	f := (n - 1) / 3
	return &PBFTSimulator{nodes: nodes, n: n, f: f, useBlst: useBlst}
}

func (s *PBFTSimulator) SelectLeader(round int) *Node {
	active := []*Node{}
	for _, nd := range s.nodes {
		if nd.IsActive() {
			active = append(active, nd)
		}
	}
	if len(active) == 0 {
		return nil
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].m > active[j].m
	})
	idx := round % len(active)
	return active[idx]
}

func (s *PBFTSimulator) ComputeTiers() {
	arr := append([]*Node{}, s.nodes...)
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
			nd.Tier = TierHigh
		} else if i >= n-bottom {
			nd.Tier = TierLow
		} else {
			nd.Tier = TierNormal
		}
	}
}

// RunRound 发起单轮共识，返回是否达成共识；并采集简单日志（可扩展为 CSV）
func (s *PBFTSimulator) RunRound(round int, request []byte) bool {
	leader := s.SelectLeader(round)
	if leader == nil {
		fmt.Println("No active leader available")
		return false
	}
	fmt.Printf("\n--- Round %d: leader=%s ---\n", round, leader.String())

	// PRE-PREPARE
	if leader.IsMalicious && rand.Float64() < 0.5 {
		fmt.Printf("Leader %d acted maliciously in pre-prepare\n", leader.ID)
		leader.UpdateReward(false)
		return false
	}

	// PREPARE: 所有活跃节点签名
	var wg sync.WaitGroup
	var mu sync.Mutex
	signatures := make([][]byte, 0, s.n)
	pubKeys := make([][]byte, 0, s.n)
	for _, nd := range s.nodes {
		if !nd.IsActive() {
			continue
		}
		wg.Add(1)
		go func(node *Node) {
			defer wg.Done()
			sig, err := node.Sign(request)
			if err == nil && sig != nil {
				mu.Lock()
				signatures = append(signatures, sig)
				pubKeys = append(pubKeys, node.bls.PublicKey())
				mu.Unlock()
			}
		}(nd)
	}
	wg.Wait()

	// leader 聚合
	aggSig, _ := leader.bls.AggregateSignatures(signatures)

	// leader 验证聚合签名
	ok, _ := leader.bls.VerifyAggregate(pubKeys, request, aggSig)
	if !ok {
		fmt.Println("Aggregate verification failed in prepare phase")
		leader.UpdateReward(false)
		return false
	}

	// COMMIT: 节点对聚合签名再次签名（模拟）
	commitSigs := make([][]byte, 0)
	commitPubKeys := make([][]byte, 0)
	for _, nd := range s.nodes {
		if !nd.IsActive() {
			continue
		}
		sig, err := nd.Sign(aggSig)
		if err == nil && sig != nil {
			commitSigs = append(commitSigs, sig)
			commitPubKeys = append(commitPubKeys, nd.bls.PublicKey())
		}
	}

	aggCommitSig, _ := leader.bls.AggregateSignatures(commitSigs)
	ok2, _ := leader.bls.VerifyAggregate(commitPubKeys, aggSig, aggCommitSig)
	if !ok2 {
		fmt.Println("Aggregate verification failed in commit phase")
		leader.UpdateReward(false)
		return false
	}

	// 判断阈值
	if len(commitSigs) >= int(float64(s.n)*PrepareQuorumMultiplier) {
		fmt.Println("Consensus achieved in this round")
		successIDs := map[int]bool{}
		for _, pk := range commitPubKeys {
			// stub 公钥解析演示（若真实 pk 为字节流则需其它映射）
			var id int
			fmt.Sscanf(string(pk), "PK-node-%02d", &id)
			successIDs[id] = true
		}
		for _, nd := range s.nodes {
			if successIDs[nd.ID] {
				nd.UpdateReward(true)
			} else {
				nd.UpdateReward(false)
			}
		}
		return true
	} else {
		fmt.Println("Not enough commit signatures; consensus failed")
		for _, nd := range s.nodes {
			nd.UpdateReward(false)
		}
		return false
	}
}
