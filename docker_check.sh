#!/bin/bash

# Docker Verification Helper Script
# Checks Docker daemon, docker-compose availability, and functionality

# Returns 0 if Docker is working, 1 if not
# Sets DOCKER_COMPOSE_CMD variable for use by calling script

DOCKER_WORKING=true
DOCKER_COMPOSE_CMD=""

echo "🔍 Checking Docker status..."

# Check if Docker daemon is running
if ! docker info >/dev/null 2>&1; then
    echo "  ❌ Docker daemon is not running or not accessible"
    echo "  💡 Try: sudo systemctl start docker"
    echo "  💡 Or: sudo usermod -aG docker $USER (then log out and back in)"
    DOCKER_WORKING=false
else
    echo "  ✅ Docker daemon is running"
fi

# Check Docker Compose availability (prefer legacy docker-compose for compatibility)
if command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE_CMD="docker-compose"
    DOCKER_COMPOSE_VERSION=$(docker-compose --version | head -n1)
    echo "  ✅ Found docker-compose (legacy): $DOCKER_COMPOSE_VERSION"
elif docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE_CMD="docker compose"
    DOCKER_COMPOSE_VERSION=$(docker compose version | head -n1)
    echo "  ⚠️  Found docker compose (v2): $DOCKER_COMPOSE_VERSION"
    echo "  💡 Note: Scripts work with v2, but legacy docker-compose is recommended for compatibility"
else
    echo "  ❌ No docker-compose found"
    echo "  💡 Install with: sudo apt install docker-compose"
    echo "  💡 Recommended: Use legacy docker-compose for best compatibility"
    DOCKER_WORKING=false
fi

# Test Docker Compose functionality
if [ "$DOCKER_WORKING" = true ] && [ -n "$DOCKER_COMPOSE_CMD" ]; then
    echo "  🔍 Testing docker-compose functionality..."
    if $DOCKER_COMPOSE_CMD version >/dev/null 2>&1; then
        echo "  ✅ Docker Compose is working correctly"
    else
        echo "  ❌ Docker Compose test failed"
        DOCKER_WORKING=false
    fi
fi

# Show final Docker status
if [ "$DOCKER_WORKING" = true ]; then
    echo "✅ Docker is ready (using: $DOCKER_COMPOSE_CMD)"
    # Export the command for use by calling script
    export DOCKER_COMPOSE_CMD
    exit 0
else
    echo "❌ Docker setup issues detected!"
    echo ""
    echo "🛠️  Troubleshooting steps:"
    echo "  1. Start Docker: sudo systemctl start docker"
    echo "  2. Add user to docker group: sudo usermod -aG docker $USER"
    echo "  3. Log out and back in, or run: newgrp docker"
    echo "  4. Install docker-compose (legacy): sudo apt install docker-compose"
    echo "  5. Test: docker ps && docker-compose --version"
    echo ""
    echo "📋 Requirements:"
    echo "  - Docker daemon running and accessible"
    echo "  - docker-compose (legacy version) installed"
    echo "  - User in docker group"
    exit 1
fi
