#!/usr/bin/env bash
set -e
echo "Running BLS correctness tests (build tag: blst)"
go test -tags blst -run TestPBFTSimulationWithStub ./...
echo "If tests passed, run a short simulation"
go build -tags blst -o sim_blst ./...
./sim_blst
