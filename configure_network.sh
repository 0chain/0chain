#!/bin/bash

# Network Configuration Helper Script
# Helps users choose the correct network setup for their platform

echo "🌐 Züs Blockchain Network Configuration"
echo "========================================"
echo ""

# Detect current platform
if [[ "$OSTYPE" == "darwin"* ]]; then
    DETECTED_PLATFORM="macos"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if grep -q Microsoft /proc/version 2>/dev/null; then
        DETECTED_PLATFORM="wsl_ubuntu"
    elif [ -n "$WSL_DISTRO_NAME" ] || [ -n "$WSLENV" ]; then
        DETECTED_PLATFORM="wsl_ubuntu"
    else
        DETECTED_PLATFORM="wsl_ubuntu"  # Default to wsl_ubuntu for Linux systems
    fi
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    DETECTED_PLATFORM="windows"
else
    DETECTED_PLATFORM="wsl_ubuntu"  # Default to wsl_ubuntu for unknown systems
fi

echo "🔍 Detected platform: $DETECTED_PLATFORM"
echo ""

# Show available options
echo "Available network platforms:"
echo "1) macos          - macOS with ifconfig aliases"
echo "2) wsl_ubuntu     - WSL Ubuntu with iptables NAT"
echo "3) windows        - Windows with PowerShell"
echo "4) custom         - Use your own network script"
echo "5) skip           - Don't change network configuration"
echo ""

# Get user choice
read -p "Choose your platform (1-5) [default: $DETECTED_PLATFORM]: " choice

case $choice in
    1)
        PLATFORM="macos"
        ;;
    2)
        PLATFORM="wsl_ubuntu"
        ;;
    3)
        PLATFORM="windows"
        ;;
    4)
        PLATFORM="custom"
        read -p "Enter path to your custom network script: " custom_script
        ;;
    5)
        echo "Skipping network configuration..."
        exit 0
        ;;
    "")
        PLATFORM="$DETECTED_PLATFORM"
        ;;
    *)
        echo "Invalid choice. Using detected platform: $DETECTED_PLATFORM"
        PLATFORM="$DETECTED_PLATFORM"
        ;;
esac

# Update blockchain.config
echo ""
echo "📝 Updating blockchain.config..."

# Create backup
cp blockchain.config blockchain.config.backup 2>/dev/null || true

# Update the configuration
if [ -f "blockchain.config" ]; then
    # Update existing NETWORK_PLATFORM
    if grep -q "^NETWORK_PLATFORM=" blockchain.config; then
        sed -i "s/^NETWORK_PLATFORM=.*/NETWORK_PLATFORM=$PLATFORM/" blockchain.config
    else
        echo "NETWORK_PLATFORM=$PLATFORM" >> blockchain.config
    fi
    
    # Update custom script if provided
    if [ "$PLATFORM" = "custom" ] && [ -n "$custom_script" ]; then
        if grep -q "^CUSTOM_NETWORK_SCRIPT=" blockchain.config; then
            sed -i "s|^CUSTOM_NETWORK_SCRIPT=.*|CUSTOM_NETWORK_SCRIPT=$custom_script|" blockchain.config
        else
            echo "CUSTOM_NETWORK_SCRIPT=$custom_script" >> blockchain.config
        fi
    fi
else
    # Create new config file
    cat > blockchain.config << EOF
# Blockchain Configuration
# Edit these values to change the number of nodes

NUM_SHARDERS=2
NUM_MINERS=3

# Network Configuration
# Choose your platform: macos, wsl_ubuntu, windows, or custom
NETWORK_PLATFORM=$PLATFORM
EOF
    if [ "$PLATFORM" = "custom" ] && [ -n "$custom_script" ]; then
        echo "CUSTOM_NETWORK_SCRIPT=$custom_script" >> blockchain.config
    fi
fi

echo "✅ Configuration updated!"
echo ""
echo "📋 Current configuration:"
echo "  Platform: $PLATFORM"
if [ "$PLATFORM" = "custom" ] && [ -n "$custom_script" ]; then
    echo "  Custom script: $custom_script"
fi
echo ""
echo "🚀 You can now run:"
echo "  ./zus_setup.sh    - Full setup"
echo "  ./zus_start.sh    - Start services"
echo "  ./zus_restart.sh  - Restart services"
echo ""
echo "🛠️  To change configuration later:"
echo "  ./configure_network.sh"
echo "  or edit blockchain.config directly"
