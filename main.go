package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	// 开关：是否使用 blst（若使用 blst 请用 -tags blst 编译并把 useBlst=true）
	useBlst := false

	rand.Seed(time.Now().UnixNano())
	nodes := []*Node{}
	for i := 0; i < 10; i++ {
		throughput := 50.0 + rand.Float64()*150.0
		isMal := false
		if i == 2 || i == 7 {
			isMal = true
		}
		nodes = append(nodes, NewNode(i, throughput, isMal, useBlst))
	}

	sim := NewPBFTSimulator(nodes, useBlst)
	sim.ComputeTiers()

	fmt.Println("Initial node statuses:")
	for _, nd := range sim.nodes {
		fmt.Println(nd.String())
	}

	totalRounds := 20
	for r := 0; r < totalRounds; r++ {
		if r%5 == 0 && r > 0 {
			for _, nd := range sim.nodes {
				nd.Throughput = nd.Throughput * (0.9 + rand.Float64()*0.2)
			}
			sim.ComputeTiers()
			fmt.Println("\nRecomputed tiers:")
			for _, nd := range sim.nodes {
				fmt.Println(nd.String())
			}
		}
		request := []byte(fmt.Sprintf("request-%d", r))
		ok := sim.RunRound(r, request)
		if !ok {
			fmt.Printf("Round %d failed\n", r)
		}
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("\nFinal node status:")
	for _, nd := range sim.nodes {
		fmt.Println(nd.String())
	}
}
