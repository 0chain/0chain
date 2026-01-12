#!/bin/bash

# Züs Blockchain Startup Script
# Starts 2 sharders and 3 miners using the existing Makefile commands

set -e

# Load configuration
if [ -f "blockchain.config" ]; then
    source blockchain.config
else
    # Default configuration
    NUM_SHARDERS=1
    NUM_MINERS=4
    NETWORK_PLATFORM=wsl_ubuntu
fi

# Select magic block file based on configuration
echo "🔍 Selecting magic block file for $NUM_MINERS miners and $NUM_SHARDERS sharders..."

# Try to construct the filename based on miner/sharder count
if [ "$NUM_SHARDERS" -eq 1 ]; then
    MAGIC_BLOCK_FILE="docker.local/config/b0magicBlock_${NUM_MINERS}_miners_${NUM_SHARDERS}_sharder.json"
else
    MAGIC_BLOCK_FILE="docker.local/config/b0magicBlock_${NUM_MINERS}_miners_${NUM_SHARDERS}_sharders.json"
fi

# Check if the constructed file exists, otherwise use default
if [ -f "$MAGIC_BLOCK_FILE" ]; then
    echo "✅ Using magic block file: $(basename $MAGIC_BLOCK_FILE)"
else
    echo "⚠️  File not found, using default: b0magicBlock_4_miners_2_sharders.json"
    MAGIC_BLOCK_FILE="docker.local/config/b0magicBlock_4_miners_2_sharders.json"
fi

# Update 0chain.yaml configuration
echo "🔧 Updating 0chain.yaml configuration..."
CONFIG_FILE="docker.local/config/0chain.yaml"
if [ -f "$CONFIG_FILE" ]; then
    MAGIC_BLOCK_FILENAME=$(basename "$MAGIC_BLOCK_FILE")
    sed -i '' "s|magic_block_file: config/.*\.json|magic_block_file: config/$MAGIC_BLOCK_FILENAME|" "$CONFIG_FILE"
    echo "✅ Updated configuration with: $MAGIC_BLOCK_FILENAME"
fi

echo "🚀 Starting Züs Blockchain with $NUM_SHARDERS sharders and $NUM_MINERS miners..."

# Network setup based on configuration
case "$NETWORK_PLATFORM" in
    "macos")
        echo "Setting up macOS network..."
        ./macos_network.sh
        ;;
    "wsl_ubuntu")
        echo "Setting up WSL Ubuntu network..."
        ./wsl_ubuntu_network_iptables.sh
        ;;
    "windows")
        echo "Setting up Windows network..."
        ./windows_network.ps1
        ;;
    "custom")
        if [ -n "$CUSTOM_NETWORK_SCRIPT" ] && [ -f "$CUSTOM_NETWORK_SCRIPT" ]; then
            echo "Running custom network script: $CUSTOM_NETWORK_SCRIPT"
            $CUSTOM_NETWORK_SCRIPT
        else
            echo "❌ Error: CUSTOM_NETWORK_SCRIPT not set or file not found"
            exit 1
        fi
        ;;
    "linux")
        echo "Setting up Linux network (defaulting to WSL Ubuntu)..."
        ./wsl_ubuntu_network_iptables.sh
        ;;
    *)
        echo "❌ Error: Unknown NETWORK_PLATFORM: $NETWORK_PLATFORM"
        echo "Supported platforms: macos, wsl_ubuntu, windows, custom, linux"
        exit 1
        ;;
esac

echo "Network setup"
if ! ./docker.local/bin/setup.network.sh; then
    echo "⚠️  Network setup failed (network may already exist), continuing..."
fi

# Verify Docker is working
if ! ./docker_check.sh; then
    exit 1
fi

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

# Track success/failure
SETUP_SUCCESS=true
ERRORS=()

echo "Starting sharders..."
for i in $(seq 1 $NUM_SHARDERS); do
    echo "  Starting sharder $i..."
    make sharder num=$i &
    eval "SHARDER${i}_PID=\$!"
    echo "    ✅ Sharder $i started (PID: $SHARDER${i}_PID)"
    sleep 15
done

echo "Waiting for sharders to initialize..."
sleep 15

echo "Starting miners..."
for i in $(seq 1 $NUM_MINERS); do
    echo "  Starting miner $i..."
    make miner num=$i &
    eval "MINER${i}_PID=\$!"
    echo "    ✅ Miner $i started (PID: $MINER${i}_PID)"
    sleep 15
done

echo ""
echo "⏳ Waiting for all services to initialize..."
sleep 20

# Verify services are actually running
echo ""
echo "🔍 Verifying all services are running..."

# Check for running containers
RUNNING_CONTAINERS=$(docker ps --filter "name=miner" --filter "name=sharder" --format "table {{.Names}}" | grep -v NAMES | wc -l)
EXPECTED_CONTAINERS=$((NUM_SHARDERS + NUM_MINERS))

if [ "$RUNNING_CONTAINERS" -lt "$EXPECTED_CONTAINERS" ]; then
    echo "  ❌ Only $RUNNING_CONTAINERS containers running (expected $EXPECTED_CONTAINERS)"
    SETUP_SUCCESS=false
    ERRORS+=("Only $RUNNING_CONTAINERS/$EXPECTED_CONTAINERS containers running")
else
    echo "  ✅ $RUNNING_CONTAINERS containers are running"
fi

# Wait for background processes to complete initialization
echo ""
echo "⏳ Waiting for blockchain initialization to complete..."
sleep 20

# Show final status
echo ""
if [ "$SETUP_SUCCESS" = true ]; then
    echo "🎉 All sharders and miners started successfully!"
else
    echo "❌ Some services failed to start:"
    for error in "${ERRORS[@]}"; do
        echo "  - $error"
    done
    echo ""
    echo "🛠️  Try running the setup again or check the logs:"
    echo "  docker logs <container_name>"
    echo "  ps aux | grep -E \"(miner|sharder)\""
    exit 1
fi
echo ""
echo "📊 Service Status:"
echo "  Sharders: $NUM_SHARDERS running"
echo "  Miners:   $NUM_MINERS running"
echo ""
echo "🔗 Blockchain Endpoints:"
echo "  Sharders:"
for i in $(seq 1 $NUM_SHARDERS); do
    echo "    - http://localhost:717$i/_diagnostics"
done
echo "  Miners:"
for i in $(seq 1 $NUM_MINERS); do
    echo "    - http://localhost:707$i/_diagnostics"
done
echo ""
echo "📋 Useful Commands:"
echo "  Check running processes: ps aux | grep -E \"(miner|sharder)\""
echo "  Check Docker containers: docker ps"
echo "  View logs: docker logs <container_name>"
echo "  Stop services: ./zus_stop.sh"
echo ""
echo "✨ Your Züs blockchain is now running!"
