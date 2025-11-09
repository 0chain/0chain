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

# Verify Docker is working
if ! ./check_docker.sh; then
  echo "❌ Docker check failed. Please fix Docker issues before continuing." >&2
  exit 1
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
echo "🎉 All sharders and miners started successfully!"
echo ""
echo "📋 Useful Commands:"
echo "  Check running processes: ps aux | grep -E \"(miner|sharder)\""
echo "  Check Docker containers: docker ps"
echo "  View logs: docker logs <container_name>"
echo "  Stop services: ./stop_chain.sh"
echo ""
echo "✨ Your Züs blockchain is now running!"
