#!/bin/bash

echo "🛑 Stopping Züs Blockchain services..."
cd docker.local || exit 1

stop_service() {
    local type=$1
    local count=$2
    echo "Stopping $type..."
    for i in $(seq 1 $count); do
        (
            echo "  Stopping ${type}${i}..."
            cd ${type}${i} || exit
            if ! timeout 30s docker-compose -p ${type}${i} stop 2>/dev/null; then
                echo "    Graceful stop failed, forcing stop..."
                docker-compose -p ${type}${i} kill 2>/dev/null
            fi
        )
    done
}

stop_service miner 4

echo "⏳ Waiting 10 seconds after stopping miners..."
sleep 10

stop_service sharder 3

echo ""
echo "Verifying stopped containers..."
if docker ps --format "{{.Names}}" | grep -E "(miner|sharder)" >/dev/null; then
    echo "⚠️ Some containers may still be running:"
    docker ps | grep -E "(miner|sharder)"
else
    echo "✅ All blockchain services stopped successfully!"
fi

echo ""
echo "🔄 To restart: ./zus_restart.sh"
