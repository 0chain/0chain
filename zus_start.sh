#!/bin/bash

# Züs Blockchain Start Script
# Assumes setup is already done, just starts the services

echo "🚀 Starting Züs blockchain..."

# Load configuration
if [ -f "blockchain.config" ]; then
    source blockchain.config
else
    # Default configuration
    NUM_SHARDERS=2
    NUM_MINERS=3
    NETWORK_PLATFORM=wsl_ubuntu
fi

echo "Starting $NUM_SHARDERS sharders and $NUM_MINERS miners..."

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

# Verify Docker is working
if ! ./docker_check.sh; then
    exit 1
fi

# Navigate to docker.local directory
cd docker.local

# Sync clock
make sync_clock

# Track success/failure
START_SUCCESS=true
ERRORS=()

# Start sharders
echo "Starting sharders..."
for i in $(seq 1 $NUM_SHARDERS); do
    echo "  Starting sharder $i..."
    make sharder num=$i &
    echo "    ✅ Sharder $i started"
    sleep 15
done

# Start miners
echo "Starting miners..."
for i in $(seq 1 $NUM_MINERS); do
    echo "  Starting miner $i..."
    make miner num=$i &
    echo "    ✅ Miner $i started"
    sleep 15
done

echo "⏳ Waiting for services to initialize..."
sleep 20

# Verify services are actually running
echo ""
echo "🔍 Verifying all services are running..."

# Check for running containers
RUNNING_CONTAINERS=$(docker ps --filter "name=miner" --filter "name=sharder" --format "table {{.Names}}" | grep -v NAMES | wc -l)
EXPECTED_CONTAINERS=$((NUM_SHARDERS + NUM_MINERS))

if [ "$RUNNING_CONTAINERS" -lt "$EXPECTED_CONTAINERS" ]; then
    echo "  ❌ Only $RUNNING_CONTAINERS containers running (expected $EXPECTED_CONTAINERS)"
    START_SUCCESS=false
    ERRORS+=("Only $RUNNING_CONTAINERS/$EXPECTED_CONTAINERS containers running")
else
    echo "  ✅ $RUNNING_CONTAINERS containers are running"
fi

# Check for running processes (more specific to avoid system processes)
RUNNING_PROCESSES=$(ps aux | grep -E "(miner|sharder)" | grep -v grep | grep -v "tracker-miner" | grep -v "systemd" | wc -l)
if [ "$RUNNING_PROCESSES" -lt "$EXPECTED_CONTAINERS" ]; then
    echo "  ❌ Only $RUNNING_PROCESSES processes running (expected $EXPECTED_CONTAINERS)"
    START_SUCCESS=false
    ERRORS+=("Only $RUNNING_PROCESSES/$EXPECTED_CONTAINERS processes running")
else
    echo "  ✅ $RUNNING_PROCESSES processes are running"
fi

# Wait for background processes to complete initialization
echo ""
echo "⏳ Waiting for blockchain initialization to complete..."
sleep 20

# Final status shown in zus_restart.sh