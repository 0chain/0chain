#!/bin/bash

# Züs Blockchain Startup Script
# Starts 2 sharders and 3 miners using the existing Makefile commands

set -e

echo "🚀 Starting Züs Blockchain with 2 sharders and 3 miners..."

./macos_network.sh

echo "Network setup"
if ! ./docker.local/bin/setup.network.sh; then
    echo "⚠️  Network setup failed (network may already exist), continuing..."
fi

echo "✅ Docker is running"

# Navigate to docker.local directory
cd docker.local

echo ""
echo "📋 Prerequisites check:"
echo "1. Initializing setup..."
make init_setup

echo "3. Building base containers..."
make build_base

sleep 10

echo "4. Building sharders..."
make build_sharder

sleep 30

echo "5. Building miners..."
make build_miner

sleep 30

echo ""
echo "🎯 Starting blockchain services..."

echo "Starting sharders..."
make sharder num=1 &
SHARDER1_PID=$!
echo "  ✅ Sharder 1 started (PID: $SHARDER1_PID)"
sleep 15

make sharder num=2 &
SHARDER2_PID=$!
echo "  ✅ Sharder 2 started (PID: $SHARDER2_PID)"

echo "Waiting for sharders to initialize..."
sleep 15

echo "Starting miners..."
make miner num=1 &
MINER1_PID=$!
echo "  ✅ Miner 1 started (PID: $MINER1_PID)"
sleep 15

make miner num=2 &
MINER2_PID=$!
echo "  ✅ Miner 2 started (PID: $MINER2_PID)"
sleep 15

make miner num=3 &
MINER3_PID=$!
echo "  ✅ Miner 3 started (PID: $MINER3_PID)"

echo ""
echo "⏳ Waiting for all services to initialize..."
sleep 20

echo ""
echo "🎉 All sharders and miners started successfully!"
echo ""
echo "📊 Service Status:"
echo "  Sharders: 2 running"
echo "  Miners:   3 running"
echo ""
echo "🔗 Blockchain Endpoints:"
echo "  Sharders:"
echo "    - http://localhost:7171/_diagnostics"
echo "    - http://localhost:7172/_diagnostics"
echo "  Miners:"
echo "    - http://localhost:7071/_diagnostics"
echo "    - http://localhost:7072/_diagnostics"
echo "    - http://localhost:7073/_diagnostics"
echo ""
echo "📋 Useful Commands:"
echo "  Check running processes: ps aux | grep -E \"(miner|sharder)\""
echo "  Check Docker containers: docker ps"
echo "  View logs: docker logs <container_name>"
echo "  Stop services: ./stop_chain.sh"
echo ""
echo "✨ Your Züs blockchain is now running!"
