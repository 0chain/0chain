#!/bin/bash

# Züs Blockchain Restart Script
# Stops all services, then starts them again

echo "🔄 Restarting Züs Blockchain services..."

# Load configuration
if [ -f "blockchain.config" ]; then
    source blockchain.config
else
    # Default configuration
    NUM_SHARDERS=2
    NUM_MINERS=3
fi

echo "Restarting $NUM_SHARDERS sharders and $NUM_MINERS miners..."

# Track overall success
RESTART_SUCCESS=true

# Stop all services first
echo "🛑 Stopping existing services..."
if ! ./zus_stop.sh; then
    echo "❌ Failed to stop services properly"
    RESTART_SUCCESS=false
fi

# Wait a moment for cleanup
sleep 5

# Start services again
echo "🚀 Starting services..."
if ! ./zus_start.sh; then
    echo "❌ Failed to start services properly"
    RESTART_SUCCESS=false
fi

# Wait for background processes to complete initialization
echo ""
echo "⏳ Waiting for blockchain initialization to complete..."
sleep 20

# Final status
echo ""
if [ "$RESTART_SUCCESS" = true ]; then
    echo "✅ Blockchain services restarted successfully!"
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
else
    echo "❌ Restart failed! Some services may not be running properly."
    echo ""
    echo "🛠️  Troubleshooting:"
    echo "  Check running processes: ps aux | grep -E \"(miner|sharder)\""
    echo "  Check containers: docker ps"
    echo "  View logs: docker logs <container_name>"
    echo "  Try manual restart: ./zus_stop.sh && ./zus_start.sh"
    exit 1
fi
