# Network Configuration Guide

This guide explains how to configure the network setup for the Züs Blockchain on different platforms.

## Quick Setup

Run the interactive configuration helper:
```bash
./configure_network.sh
```

## Manual Configuration

Edit `blockchain.config` to set your network platform:

```bash
# Choose your platform
NETWORK_PLATFORM=wsl_ubuntu  # or macos, windows, custom
```

## Supported Platforms

### 1. macOS (`macos`)
- **Script**: `./macos_network.sh`
- **Method**: Uses `ifconfig lo0 alias` to create network aliases
- **Network**: `198.18.0.0/15` range
- **Requirements**: macOS with `ifconfig` command

### 2. WSL Ubuntu (`wsl_ubuntu`)
- **Script**: `./wsl_ubuntu_network_iptables.sh`
- **Method**: Uses `iptables` NAT rules for network routing
- **Network**: `198.18.0.0/15` range
- **Requirements**: WSL2 with `iptables` command

### 3. Windows (`windows`)
- **Script**: `./windows_network.ps1`
- **Method**: PowerShell network configuration
- **Network**: `198.18.0.0/15` range
- **Requirements**: Windows with PowerShell

### 4. Custom (`custom`)
- **Script**: Your own network setup script
- **Method**: Whatever you implement
- **Requirements**: Set `CUSTOM_NETWORK_SCRIPT` in config

## Configuration Examples

### For WSL Ubuntu (Default)
```bash
NETWORK_PLATFORM=wsl_ubuntu
```

### For macOS
```bash
NETWORK_PLATFORM=macos
```

### For Windows
```bash
NETWORK_PLATFORM=windows
```

### For Custom Script
```bash
NETWORK_PLATFORM=custom
CUSTOM_NETWORK_SCRIPT=./my_network_setup.sh
```

## Network Requirements

The blockchain requires specific network configuration:

- **Network Range**: `198.18.0.0/15`
- **Sharder IPs**: `198.18.0.81`, `198.18.0.82`, etc.
- **Miner IPs**: `198.18.0.91`, `198.18.0.92`, etc.
- **Ports**: Various ports for sharders (7171, 7172) and miners (7071, 7072, 7073)

## Troubleshooting

### Platform Detection
The `configure_network.sh` script automatically detects your platform:
- **macOS**: Detects `darwin` OS type
- **WSL**: Detects `Microsoft` in `/proc/version`
- **Windows**: Detects `msys` or `cygwin` OS type

### Common Issues

1. **Permission Denied**: Make sure you have sudo access for network configuration
2. **Script Not Found**: Ensure the network script exists and is executable
3. **Network Already Configured**: Some scripts may fail if network is already set up (this is usually OK)

### Manual Network Setup

If automatic setup fails, you can manually configure:

#### WSL Ubuntu
```bash
sudo iptables -t nat -I OUTPUT --dst 198.18.0.0/15 -p tcp -j REDIRECT
```

#### macOS
```bash
sudo ifconfig lo0 alias 198.18.0.81
sudo ifconfig lo0 alias 198.18.0.82
# ... add more aliases as needed
```

## Scripts That Use Network Configuration

- `./zus_setup.sh` - Full blockchain setup
- `./zus_start.sh` - Start blockchain services
- `./zus_restart.sh` - Restart blockchain services

All scripts automatically use the network configuration from `blockchain.config`.
