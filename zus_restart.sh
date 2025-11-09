#!/bin/bash

# Flexible Quick Start Script
# Usage examples:
#   ./zus_start.sh                # defaults to 3 miners, 1 sharder
#   ./zus_start.sh -m 4 -s 2      # start 4 miners and 2 sharders
#   ./zus_start.sh --miners=4 --sharders=2

set -euo pipefail

MINERS=3
SHARDERS=1

# Verify Docker is working
if ! ./check_docker.sh; then
  echo "❌ Docker check failed. Please fix Docker issues before continuing." >&2
  exit 1
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--miners)
      if [[ $# -lt 2 ]]; then echo "❌ missing value for $1" >&2; exit 1; fi
      MINERS="$2"; shift 2 ;;
    --miners=*)
      MINERS="${1#*=}"; shift 1 ;;
    -s|--sharders)
      if [[ $# -lt 2 ]]; then echo "❌ missing value for $1" >&2; exit 1; fi
      SHARDERS="$2"; shift 2 ;;
    --sharders=*)
      SHARDERS="${1#*=}"; shift 1 ;;
    -h|--help)
      echo "Usage: $0 [-m <miners>] [-s <sharders>]" && exit 0 ;;
    *)
      # Ignore unknown args to be forgiving
      shift 1 ;;
  esac
done

if ! [[ "$MINERS" =~ ^[0-9]+$ ]] || ! [[ "$SHARDERS" =~ ^[0-9]+$ ]]; then
  echo "❌ miners and sharders must be integers" >&2
  exit 1
fi

if [ "$MINERS" -lt 1 ] || [ "$SHARDERS" -lt 1 ]; then
  echo "❌ miners and sharders must be >= 1" >&2
  exit 1
fi

echo "🚀 Quick starting Züs blockchain with $MINERS miners and $SHARDERS sharders..."

# Select appropriate DKG configuration
echo "🔧 Configuring DKG files for $MINERS miners and $SHARDERS sharders..."

# Determine DKG config directory
# DKG files are organized by miner count, so we match by miner count
# Strategy: First try exact match (miners + sharders), then fallback to miner count only
DKG_CONFIG_DIR=""
CONFIG_BASE="docker.local/config"

# Strategy 1: Try exact match (miners + sharders)
if [ "$SHARDERS" -eq 1 ]; then
  # Try patterns: ${MINERS}miners_${SHARDERS}_sharder and ${MINERS}_miners_${SHARDERS}_sharder
  if [ -d "$CONFIG_BASE/${MINERS}miners_${SHARDERS}_sharder" ]; then
    DKG_CONFIG_DIR="$CONFIG_BASE/${MINERS}miners_${SHARDERS}_sharder"
  elif [ -d "$CONFIG_BASE/${MINERS}_miners_${SHARDERS}_sharder" ]; then
    DKG_CONFIG_DIR="$CONFIG_BASE/${MINERS}_miners_${SHARDERS}_sharder"
  fi
else
  # Try patterns: ${MINERS}miners_${SHARDERS}_sharders and ${MINERS}_miners_${SHARDERS}_sharders
  if [ -d "$CONFIG_BASE/${MINERS}miners_${SHARDERS}_sharders" ]; then
    DKG_CONFIG_DIR="$CONFIG_BASE/${MINERS}miners_${SHARDERS}_sharders"
  elif [ -d "$CONFIG_BASE/${MINERS}_miners_${SHARDERS}_sharders" ]; then
    DKG_CONFIG_DIR="$CONFIG_BASE/${MINERS}_miners_${SHARDERS}_sharders"
  fi
fi

# Strategy 2: Fallback to match by miner count only (since DKG files are organized by miner count)
if [ -z "$DKG_CONFIG_DIR" ]; then
  echo "🔍 Exact match not found, looking for folder with $MINERS miners..."
  # Find any folder that matches the miner count pattern
  for dir in "$CONFIG_BASE"/${MINERS}_miners_* "$CONFIG_BASE"/${MINERS}miners_*; do
    if [ -d "$dir" ]; then
      # Verify it has the required DKG files
      if [ -f "$dir/b0mnode1_dkg.json" ]; then
        DKG_CONFIG_DIR="$dir"
        echo "   Found folder: $dir (using for $MINERS miners, $SHARDERS sharders)"
        break
      fi
    fi
  done
fi

if [ -n "$DKG_CONFIG_DIR" ]; then
  echo "📁 Using DKG config from: $DKG_CONFIG_DIR"
  for i in $(seq 1 "$MINERS"); do
    SRC_FILE="$DKG_CONFIG_DIR/b0mnode${i}_dkg.json"
    DST_FILE="$CONFIG_BASE/b0mnode${i}_dkg.json"
    if [ -f "$SRC_FILE" ]; then
      cp "$SRC_FILE" "$DST_FILE"
      echo "  ✅ Copied b0mnode${i}_dkg.json"
    else
      echo "  ❌ Missing $SRC_FILE" >&2
      exit 1
    fi
  done
else
  echo "❌ No DKG configuration found for $MINERS miners" >&2
  echo "   Expected folder: $CONFIG_BASE/${MINERS}_miners_* (or $CONFIG_BASE/${MINERS}miners_*)" >&2
  echo "   Available DKG folders:" >&2
  ls -1d "$CONFIG_BASE"/*miners*sharder* 2>/dev/null | sed 's|.*/|     |' || echo "     (none found)" >&2
  exit 1
fi

# Navigate to docker.local directory
cd docker.local

make cleanup

# Choose appropriate magic block for requested topology
# Strategy: First try exact match, then fallback to match by miner count only
MB_FILE=""
MB_SUFFIX="sharder"
if [ "$SHARDERS" -ne 1 ]; then MB_SUFFIX="sharders"; fi

# Strategy 1: Try exact match
MB_FILE="b0magicBlock_${MINERS}_miners_${SHARDERS}_${MB_SUFFIX}.json"
if [ ! -f "config/${MB_FILE}" ]; then
  # Strategy 2: Fallback - try to find any magic block with same miner count
  echo "🔍 Exact magic block not found, looking for one with $MINERS miners..."
  MB_FILE=""
  for mb in "config/b0magicBlock_${MINERS}_miners_"*.json; do
    if [ -f "$mb" ]; then
      MB_FILE=$(basename "$mb")
      echo "   Found: $MB_FILE (using for $MINERS miners, $SHARDERS sharders)"
      break
    fi
  done
  
  # Strategy 3: Final fallback - use default magic block if no match found
  if [ -z "$MB_FILE" ]; then
    if [ -f "config/b0magicBlock.json" ]; then
      MB_FILE="b0magicBlock.json"
      echo "   Using default magic block: $MB_FILE"
    fi
  fi
fi

if [ -z "$MB_FILE" ] || [ ! -f "config/${MB_FILE}" ]; then
  echo "❌ No suitable magic block file found for $MINERS miners" >&2
  echo "   Tried: b0magicBlock_${MINERS}_miners_${SHARDERS}_${MB_SUFFIX}.json" >&2
  echo "   Available options:" >&2
  ls -1 config/b0magicBlock_* 2>/dev/null || echo "     (none found)" >&2
  exit 1
fi

echo "🧩 Selecting magic block: ${MB_FILE}"

# Update 0chain.yaml to point to the selected magic block
# macOS-compatible in-place edit
sed -i '' "s|^\([[:space:]]*magic_block_file:[[:space:]]*\).*$|\1config/${MB_FILE}|" config/0chain.yaml

make sync_clock

echo "Starting sharders... ($SHARDERS)"
for i in $(seq 1 "$SHARDERS"); do
  echo "  Starting sharder $i..."
  make sharder num="$i" &
  sleep 5
done

echo "Starting miners... ($MINERS)"
for i in $(seq 1 "$MINERS"); do
  echo "  Starting miner $i..."
  make miner num="$i" &
  sleep 5
done

echo "⏳ Waiting for services to initialize..."
sleep 15

echo "✅ All sharders and miners started in background!"
echo ""
echo "📋 Check running processes:"
echo "  ps aux | grep -E \"(miner|sharder)\""
echo ""
echo "🔗 Blockchain Endpoints:"
echo "  Sharders start at: http://localhost:7171/_diagnostics"
echo "  Miners start at:   http://localhost:7071/_diagnostics"
echo ""
echo "🛑 To stop: ./zus_stop.sh"