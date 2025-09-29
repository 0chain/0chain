#!/bin/bash

# Quick Start Script - Your Original Approach
# Assumes setup is already done, just starts the services

echo "🚀 Quick starting Züs blockchain..."

./macos_network.sh

# Navigate to docker.local directory
cd docker.local

make sync_clock

echo "Starting sharders..."
make sharder num=1 &
sleep 15
make sharder num=2 &
sleep 15

echo "Starting miners..."
make miner num=1 &
sleep 15
make miner num=2 &
sleep 15
make miner num=3 &

echo "⏳ Waiting for services to initialize..."
sleep 20

echo "✅ All sharders and miners started in background!"
echo ""
echo "📋 Check running processes:"
echo "  ps aux | grep -E \"(miner|sharder)\""
echo ""
echo "🔗 Blockchain Endpoints:"
echo "  Sharders: http://localhost:7171/_diagnostics, http://localhost:7172/_diagnostics"
echo "  Miners:   http://localhost:7071/_diagnostics, http://localhost:7072/_diagnostics, http://localhost:7073/_diagnostics"
echo ""
echo "🛑 To stop: ./stop_chain.sh"
