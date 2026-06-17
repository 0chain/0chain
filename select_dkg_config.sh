#!/bin/bash

# DKG Configuration Selector
# Automatically selects the correct DKG files based on miner/sharder count
# Usage: ./select_dkg_config.sh <miners> <sharders>

set -euo pipefail

MINERS="$1"
SHARDERS="$2"
CONFIG_DIR="docker.local/config"

if [ ! -d "$CONFIG_DIR" ]; then
    echo "❌ Config directory not found: $CONFIG_DIR" >&2
    exit 1
fi

echo "🔧 Selecting DKG configuration for $MINERS miners and $SHARDERS sharders..."

# Clean up any existing DKG files that might have wrong share counts
echo "🧹 Cleaning up existing DKG files..."
for i in $(seq 1 8); do
    if [ -f "$CONFIG_DIR/b0mnode${i}_dkg.json" ]; then
        share_count=$(grep -A 10 '"secret_shares"' "$CONFIG_DIR/b0mnode${i}_dkg.json" | grep -c '"[a-f0-9]*":' 2>/dev/null || echo 0)
        if [ "$share_count" -ne "$MINERS" ] && [ "$share_count" -gt 0 ]; then
            echo "  🗑️  Removing b0mnode${i}_dkg.json (has $share_count shares, need $MINERS)"
            rm -f "$CONFIG_DIR/b0mnode${i}_dkg.json"
        fi
    fi
done

# Function to find and copy DKG files
select_dkg_files() {
    local miners="$1"
    local sharders="$2"
    local found_config=false
    
    # Strategy 1: Look for directory-based config (e.g., 3_miners_1_sharder/)
    local dir_pattern="${miners}_miners_${sharders}_sharder"
    if [ "$sharders" -ne 1 ]; then
        dir_pattern="${miners}_miners_${sharders}_sharders"
    fi
    
    if [ -d "$CONFIG_DIR/$dir_pattern" ]; then
        echo "📁 Found directory-based config: $dir_pattern"
        for i in $(seq 1 "$miners"); do
            local src_file="$CONFIG_DIR/$dir_pattern/b0mnode${i}_dkg.json"
            local dst_file="$CONFIG_DIR/b0mnode${i}_dkg.json"
            if [ -f "$src_file" ]; then
                cp "$src_file" "$dst_file"
                echo "  ✅ Copied b0mnode${i}_dkg.json"
            else
                echo "  ⚠️  Missing: $src_file"
            fi
        done
        found_config=true
    fi
    
    # Strategy 2: Look for individual files with naming pattern
    if [ "$found_config" = false ]; then
        echo "🔍 Looking for individual DKG files..."
        local file_pattern="b0mnode.*_${miners}_miners_${sharders}_sharder.*_dkg.json"
        if [ "$sharders" -ne 1 ]; then
            file_pattern="b0mnode.*_${miners}_miners_${sharders}_sharders.*_dkg.json"
        fi
        
        local found_files=0
        for i in $(seq 1 "$miners"); do
            local src_file=$(find "$CONFIG_DIR" -name "b0mnode${i}_${miners}_miners_${sharders}_sharder*_dkg.json" | head -1)
            if [ -n "$src_file" ]; then
                local dst_file="$CONFIG_DIR/b0mnode${i}_dkg.json"
                cp "$src_file" "$dst_file"
                echo "  ✅ Copied b0mnode${i}_dkg.json from $(basename "$src_file")"
                found_files=$((found_files + 1))
            fi
        done
        
        if [ "$found_files" -eq "$miners" ]; then
            found_config=true
        fi
    fi
    
    # Strategy 3: Check if current files already match (for common configurations)
    if [ "$found_config" = false ]; then
        echo "🔍 Checking if current DKG files match the configuration..."
        local current_miners=0
        for i in $(seq 1 8); do
            if [ -f "$CONFIG_DIR/b0mnode${i}_dkg.json" ]; then
                # Count secret shares in the DKG file
                local shares=$(grep -c '"secret_shares"' "$CONFIG_DIR/b0mnode${i}_dkg.json" 2>/dev/null || echo 0)
                if [ "$shares" -gt 0 ]; then
                    local share_count=$(grep -A 10 '"secret_shares"' "$CONFIG_DIR/b0mnode${i}_dkg.json" | grep -c '"[a-f0-9]*":' 2>/dev/null || echo 0)
                    if [ "$share_count" -eq "$miners" ]; then
                        current_miners=$((current_miners + 1))
                    fi
                fi
            fi
        done
        
        if [ "$current_miners" -eq "$miners" ]; then
            echo "  ✅ Current DKG files already match $miners miners configuration"
            found_config=true
        fi
    fi
    
    # Strategy 4: Fallback - try to generate or use default
    if [ "$found_config" = false ]; then
        echo "⚠️  No specific DKG configuration found for $miners miners and $sharders sharders"
        echo "   Available configurations:" >&2
        find "$CONFIG_DIR" -name "*dkg*.json" -path "*/${miners}_miners_${sharders}_*" 2>/dev/null | head -5 >&2 || true
        find "$CONFIG_DIR" -type d -name "*${miners}_miners_${sharders}_*" 2>/dev/null | head -5 >&2 || true
        
        # Try to use the most recent DKG files that have the right number of shares
        echo "🔄 Attempting to use existing DKG files with correct share count..."
        local used_files=0
        for i in $(seq 1 "$miners"); do
            if [ -f "$CONFIG_DIR/b0mnode${i}_dkg.json" ]; then
                local share_count=$(grep -A 10 '"secret_shares"' "$CONFIG_DIR/b0mnode${i}_dkg.json" | grep -c '"[a-f0-9]*":' 2>/dev/null || echo 0)
                if [ "$share_count" -eq "$miners" ]; then
                    echo "  ✅ Using existing b0mnode${i}_dkg.json (has $share_count shares)"
                    used_files=$((used_files + 1))
                else
                    echo "  ❌ b0mnode${i}_dkg.json has $share_count shares, need $miners"
                fi
            else
                echo "  ❌ Missing b0mnode${i}_dkg.json"
            fi
        done
        
        if [ "$used_files" -eq "$miners" ]; then
            found_config=true
        else
            echo "❌ Could not find suitable DKG configuration" >&2
            echo "   Need $miners DKG files with $miners shares each" >&2
            return 1
        fi
    fi
    
    echo "✅ DKG configuration selected successfully!"
    return 0
}

# Validate inputs
if ! [[ "$MINERS" =~ ^[0-9]+$ ]] || ! [[ "$SHARDERS" =~ ^[0-9]+$ ]]; then
    echo "❌ miners and sharders must be integers" >&2
    exit 1
fi

if [ "$MINERS" -lt 1 ] || [ "$SHARDERS" -lt 1 ]; then
    echo "❌ miners and sharders must be >= 1" >&2
    exit 1
fi

# Select DKG configuration
select_dkg_files "$MINERS" "$SHARDERS"
