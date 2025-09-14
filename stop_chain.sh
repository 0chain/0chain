#!/bin/bash

# Züs Blockchain Stop Script
# Stops all running sharders and miners

echo "🛑 Stopping Züs Blockchain services..."

# Navigate to docker.local directory
cd docker.local

echo "Stopping miners..."
for i in {1..3}; do
    echo "  Stopping miner$i..."
    cd miner$i
    docker-compose -p miner$i down 2>/dev/null || echo "    Miner$i was not running"
    cd ..
done

echo "Stopping sharders..."
for i in {1..2}; do
    echo "  Stopping sharder$i..."
    cd sharder$i
    docker-compose -p sharder$i down 2>/dev/null || echo "    Sharder$i was not running"
    cd ..
done

echo ""
echo "✅ All blockchain services stopped!"
echo ""
echo "📋 To verify:"
echo "  Check processes: ps aux | grep -E \"(miner|sharder)\""
echo "  Check containers: docker ps"
echo ""
echo "🔄 To restart: ./quick_start.sh"
