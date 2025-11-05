#!/bin/bash

# Quick DKG Configuration Helper
# Usage: ./configure_dkg.sh <miners> <sharders>
# Example: ./configure_dkg.sh 3 1

set -euo pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 <miners> <sharders>"
    echo "Examples:"
    echo "  $0 3 1    # Configure for 3 miners, 1 sharder"
    echo "  $0 4 2    # Configure for 4 miners, 2 sharders"
    exit 1
fi

MINERS="$1"
SHARDERS="$2"

echo "🔧 Configuring DKG for $MINERS miners and $SHARDERS sharders..."

if ./select_dkg_config.sh "$MINERS" "$SHARDERS"; then
    echo ""
    echo "✅ DKG configuration complete!"
    echo "🚀 You can now run: ./zus_start.sh -m $MINERS -s $SHARDERS"
else
    echo ""
    echo "❌ DKG configuration failed!"
    echo "💡 Make sure you have the correct DKG files for this configuration"
    exit 1
fi
