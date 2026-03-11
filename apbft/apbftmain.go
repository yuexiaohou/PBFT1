// Update to handle the new behavior when sim.RunRound returns false

result := sim.RunRound(r, request)
if !result {
    // Do not submit orders or call ob.MatchAndClear()
    saveConsensusResult() // Replace with the actual function to save consensus result
    time.Sleep(// Specify the duration for sleeping)
    continue // Continue to the next round after handling the result
}

// Existing behavior when ok is true
ob.MatchAndClear()