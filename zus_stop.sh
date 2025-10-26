#!/bin/bash

# Züs Blockchain Stop Script
# Stops all running sharders and miners

echo "🛑 Stopping Züs Blockchain services..."

# Verify Docker is working (needed for container operations)
if ! ./docker_check.sh >/dev/null 2>&1; then
    echo "⚠️  Docker issues detected, but continuing with stop operations..."
    echo "  Some operations may fail due to Docker permissions"
fi

# Load configuration
if [ -f "blockchain.config" ]; then
    source blockchain.config
else
    # Default configuration
    NUM_SHARDERS=2
    NUM_MINERS=3
fi

echo "Stopping $NUM_MINERS miners and $NUM_SHARDERS sharders..."

# Track success/failure
STOP_SUCCESS=true
ERRORS=()

# Stop miners
echo "Stopping miners..."
for i in $(seq 1 $NUM_MINERS); do
    echo "  Stopping miner$i..."
    # Try docker-compose first, then docker compose
    if command -v docker-compose >/dev/null 2>&1; then
        if ! docker-compose -p miner$i -f docker.local/build.miner/docker-compose.yml down 2>/dev/null; then
            echo "    ⚠️  Miner$i was not running or failed to stop"
            ERRORS+=("Miner$i: Not running or failed to stop")
        else
            echo "    ✅ Miner$i stopped successfully"
        fi
    else
        if ! docker compose -p miner$i -f docker.local/build.miner/docker-compose.yml down 2>/dev/null; then
            echo "    ⚠️  Miner$i was not running or failed to stop"
            ERRORS+=("Miner$i: Not running or failed to stop")
        else
            echo "    ✅ Miner$i stopped successfully"
        fi
    fi
done

# Stop sharders
echo "Stopping sharders..."
for i in $(seq 1 $NUM_SHARDERS); do
    echo "  Stopping sharder$i..."
    # Try docker-compose first, then docker compose
    if command -v docker-compose >/dev/null 2>&1; then
        if ! docker-compose -p sharder$i -f docker.local/build.sharder/docker-compose.yml down 2>/dev/null; then
            echo "    ⚠️  Sharder$i was not running or failed to stop"
            ERRORS+=("Sharder$i: Not running or failed to stop")
        else
            echo "    ✅ Sharder$i stopped successfully"
        fi
    else
        if ! docker compose -p sharder$i -f docker.local/build.sharder/docker-compose.yml down 2>/dev/null; then
            echo "    ⚠️  Sharder$i was not running or failed to stop"
            ERRORS+=("Sharder$i: Not running or failed to stop")
        else
            echo "    ✅ Sharder$i stopped successfully"
        fi
    fi
done

# Also stop any running containers by name pattern
echo "Stopping any remaining blockchain containers..."
if ! docker stop $(docker ps -q --filter "name=miner") 2>/dev/null; then
    echo "  ⚠️  No miner containers found to stop"
fi
if ! docker stop $(docker ps -q --filter "name=sharder") 2>/dev/null; then
    echo "  ⚠️  No sharder containers found to stop"
fi

# Check if any services are still running
echo ""
echo "🔍 Verifying all services are stopped..."

# Check for running containers
RUNNING_CONTAINERS=$(docker ps --filter "name=miner" --filter "name=sharder" --format "table {{.Names}}" | grep -v NAMES | wc -l)
if [ "$RUNNING_CONTAINERS" -gt 0 ]; then
    echo "  ❌ $RUNNING_CONTAINERS blockchain containers are still running:"
    docker ps --filter "name=miner" --filter "name=sharder" --format "  - {{.Names}} ({{.Status}})"
    STOP_SUCCESS=false
    ERRORS+=("$RUNNING_CONTAINERS containers still running")
else
    echo "  ✅ No blockchain containers are running"
fi

# Show final status
echo ""
if [ "$STOP_SUCCESS" = true ]; then
    echo "✅ All blockchain services stopped successfully!"
else
    echo "❌ Some blockchain services failed to stop:"
    for error in "${ERRORS[@]}"; do
        echo "  - $error"
    done
    echo ""
    echo "🛠️  Try running the stop command again or manually stop remaining services:"
    echo "  docker stop \$(docker ps -q --filter 'name=miner' --filter 'name=sharder')"
    echo "  pkill -f 'miner|sharder'"
    exit 1
fi

echo ""
echo "📋 Verification commands:"
echo "  Check processes: ps aux | grep -E \"(miner|sharder)\""
echo "  Check containers: docker ps"
echo ""
echo "🔄 To restart: ./zus_restart.sh"
