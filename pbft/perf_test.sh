#!/usr/bin/env bash
set -e
echo "Building stub binary..."
go build -o sim_stub ./...
echo "Running stub..."
/usr/bin/time -v ./sim_stub > run_stub.log 2>&1

echo "Building blst binary (tags blst)..."
go build -tags blst -o sim_blst ./...
echo "Running blst..."
/usr/bin/time -v ./sim_blst > run_blst.log 2>&1

echo "Done. Logs: run_stub.log, run_blst.log"
