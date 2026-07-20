#!/bin/bash

set -euo pipefail

MODULE_BASE="0chain.net"

echo "🔍 Scanning for mock import paths..."

# Grep for mock imports (works on macOS)
IMPORT_PATHS=$(grep -rho "$MODULE_BASE" . --include='*.go' | grep '/mocks' | sort -u || true)

if [ -z "$IMPORT_PATHS" ]; then
  echo "❌ No mock imports found."
  exit 1
fi

echo "Found mock imports:"
echo "$IMPORT_PATHS"

echo "------"
for MOCK_PATH in $IMPORT_PATHS; do
  echo "🧩 Mock import: $MOCK_PATH"

  INTERFACE_PATH=$(echo "$MOCK_PATH" | sed 's|/mocks||')
  LOCAL_PATH="./$(echo "$INTERFACE_PATH" | sed "s|$MODULE_BASE/||")"

  echo "→ Expecting interface folder: $LOCAL_PATH"

  if [ -d "$LOCAL_PATH\" ]; then
    echo "✅ Found interface dir: $LOCAL_PATH"
    grep -r "type .* interface" "$LOCAL_PATH" || echo "⚠️  No interfaces found."
  else
    echo "❌ Directory not found: $LOCAL_PATH"
  fi

  echo "------"
done