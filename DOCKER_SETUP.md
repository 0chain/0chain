# Docker Setup Guide

This guide explains how to properly set up Docker and docker-compose for the Züs Blockchain.

## Requirements

- **Docker Engine 20.10+** (recommended: latest stable)
- **docker-compose 1.29+** (legacy version recommended for compatibility)
- **User in docker group** for permission access
- **Go 1.22+** (for local development, not required for Docker-only usage)

## Quick Setup

### 1. Install Docker
```bash
# Update package index
sudo apt update

# Install Docker
sudo apt install docker.io

# Start Docker service
sudo systemctl start docker
sudo systemctl enable docker
```

### 2. Install docker-compose (Legacy Version)
```bash
# Install docker-compose (legacy version)
sudo apt install docker-compose

# Verify installation
docker-compose --version
```

### 3. Add User to Docker Group
```bash
# Add your user to docker group
sudo usermod -aG docker $USER

# Log out and back in, or run:
newgrp docker

# Test Docker access
docker ps
```

## Verification

Run the Docker check script to verify everything is working:
```bash
./docker_check.sh
```

You should see:
```
🔍 Checking Docker status...
  ✅ Docker daemon is running
  ✅ Found docker-compose (legacy): docker-compose version 1.29.2, build unknown
  ✅ Docker Compose is working correctly
✅ Docker is ready (using: docker-compose)
```

## Version Compatibility

### Go Version Requirements
- **Minimum**: Go 1.21
- **Recommended**: Go 1.22.1 (matches Docker build environment)
- **Docker Build**: Uses `golang:1.22.1-alpine3.18`
- **Local Development**: Go 1.21+ required for `go mod` commands

### Docker Version Requirements
- **Docker Engine**: 20.10+ (recommended: latest stable)
- **Docker Compose**: 1.29+ (legacy version recommended)
- **Docker Compose v2**: Supported but not recommended

## Docker Compose Versions

### Legacy docker-compose (Recommended)
- **Command**: `docker-compose`
- **Installation**: `sudo apt install docker-compose`
- **Version**: 1.29+ recommended
- **Compatibility**: Best with existing scripts
- **Status**: ✅ Recommended

### Modern Docker Compose v2
- **Command**: `docker compose`
- **Installation**: Included with Docker
- **Version**: 2.0+ (latest recommended)
- **Compatibility**: Works but may have issues
- **Status**: ⚠️ Supported but not recommended

## Troubleshooting

### Docker Permission Denied
```bash
# Error: permission denied while trying to connect to the Docker daemon socket
# Solution:
sudo usermod -aG docker $USER
newgrp docker
```

### Docker Daemon Not Running
```bash
# Error: Cannot connect to the Docker daemon
# Solution:
sudo systemctl start docker
sudo systemctl enable docker
```

### docker-compose Not Found
```bash
# Error: docker-compose: command not found
# Solution:
sudo apt install docker-compose
```

### Wrong Docker Compose Version
```bash
# If you have Docker Compose v2 but need legacy:
# Install legacy version:
sudo apt install docker-compose

# Verify both versions:
docker-compose --version  # Should show 1.x.x
docker compose version    # Should show 2.x.x
```

## Testing Your Setup

### 1. Test Docker
```bash
docker ps
docker info
```

### 2. Test docker-compose
```bash
docker-compose --version
docker-compose config
```

### 3. Test Full Setup
```bash
./docker_check.sh
```

### 4. Test Blockchain Setup
```bash
./zus_setup.sh
```

## Common Issues

### Issue: Scripts prefer docker-compose but you have docker compose v2
**Solution**: Install legacy docker-compose alongside v2
```bash
sudo apt install docker-compose
```

### Issue: Permission denied errors
**Solution**: Add user to docker group and restart session
```bash
sudo usermod -aG docker $USER
newgrp docker
```

### Issue: Docker daemon not running
**Solution**: Start Docker service
```bash
sudo systemctl start docker
sudo systemctl enable docker
```

### Issue: docker-compose command not found
**Solution**: Install docker-compose
```bash
sudo apt install docker-compose
```

## Why Legacy docker-compose?

1. **Better Compatibility**: Works with all existing scripts and configurations
2. **Stable**: Mature, well-tested version
3. **Widely Supported**: Most tutorials and documentation use this version
4. **Consistent**: Same command across all systems
5. **Reliable**: Less likely to have breaking changes

## Script Compatibility

| Script | docker-compose | docker compose v2 |
|--------|----------------|-------------------|
| zus_setup.sh | ✅ Recommended | ⚠️ Works |
| zus_start.sh | ✅ Recommended | ⚠️ Works |
| zus_stop.sh | ✅ Recommended | ⚠️ Works |
| zus_restart.sh | ✅ Recommended | ⚠️ Works |
| build.sharders.sh | ✅ Recommended | ⚠️ Works |

## Next Steps

Once Docker is properly set up:
1. Run `./zus_setup.sh` to set up the blockchain
2. Run `./zus_start.sh` to start services
3. Run `./zus_stop.sh` to stop services
4. Run `./zus_restart.sh` to restart services
