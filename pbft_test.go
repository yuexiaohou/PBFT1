package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestPBFTSimulationWithStub(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	nodes := []*Node{}
	for i := 0; i < 7; i++ {
		tp := 80.0 + rand.Float64()*120.0
		isMal := false
		if i == 3 {
			isMal = true
		}
		nodes = append(nodes, NewNode(i, tp, isMal, false))
	}
	sim := NewPBFTSimulator(nodes, false)
	sim.ComputeTiers()

	rounds := 5
	successes := 0
	for r := 0; r < rounds; r++ {
		if sim.RunRound(r, []byte("test-request")) {
			successes++
		}
	}
	if successes == 0 {
		t.Fatalf("expected at least 1 successful consensus round, got 0")
	}
}
