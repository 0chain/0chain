#!/bin/bash

# Setup View Change for Local Development
# This script configures loopback interfaces, funds wallet, sets hardforks,
# and enables view change with faster round durations.
#
# Usage: ./docker.local/bin/setup_vc.sh
#
# Prerequisites:
# - Local chain running (miners + sharders + 0dns)
# - zwallet binary at ~/Code/zwalletcli/zwallet
# - local.json and local.yaml in ~/Code/zwalletcli/

set -e

ZWALLET_DIR="/Users/saswatabasu/Code/zwalletcli"
ZWALLET="$ZWALLET_DIR/zwallet"
WALLET="local.json"
CONFIG="local.yaml"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${CYAN}[$(date '+%H:%M:%S')]${NC} $1"; }
ok()  { echo -e "${GREEN}  OK${NC}"; }
fail() { echo -e "${RED}  FAILED: $1${NC}"; }

# ─── Step 1: Setup loopback interfaces ───────────────────────────────────────

if ifconfig lo0 2>/dev/null | grep -q "198.18.0.71"; then
    log "Loopback interfaces already configured, skipping..."
    ok
else
    log "Setting up loopback interfaces (requires sudo)..."

    # Miners
    ifconfig lo0 alias 198.18.0.71
    ifconfig lo0 alias 198.18.0.72
    ifconfig lo0 alias 198.18.0.73
    ifconfig lo0 alias 198.18.0.74

    # Sharders
    ifconfig lo0 alias 198.18.0.81
    ifconfig lo0 alias 198.18.0.82

    # 0dns
    ifconfig lo0 alias 198.18.0.100

    # Blobbers
    ifconfig lo0 alias 198.18.0.97
    ifconfig lo0 alias 198.18.0.98
    ifconfig lo0 alias 198.18.0.99
    ifconfig lo0 alias 198.18.0.110
    ifconfig lo0 alias 198.18.0.111
    ifconfig lo0 alias 198.18.0.112

    ok
fi

# ─── Step 2: Wait for chain to be ready ──────────────────────────────────────

log "Checking chain is running..."
MAX_WAIT=30
for i in $(seq 1 $MAX_WAIT); do
    ROUND=$(curl -s http://localhost:7171/v1/block/get/latest_finalized 2>/dev/null | python3 -c "import json,sys; print(json.load(sys.stdin).get('round',0))" 2>/dev/null || echo "0")
    if [ "$ROUND" -gt 0 ] 2>/dev/null; then
        echo -e "${GREEN}  Chain running at round $ROUND${NC}"
        break
    fi
    if [ "$i" -eq "$MAX_WAIT" ]; then
        fail "Chain not running. Start miners and sharders first."
        exit 1
    fi
    sleep 1
done

# ─── Step 3: Fund wallet ─────────────────────────────────────────────────────

log "Funding wallet from faucet (10M tokens)..."
cd "$ZWALLET_DIR"
OUTPUT=$($ZWALLET faucet --methodName pour --input "{Pay day}" --tokens 10000000 --config $CONFIG --wallet $WALLET 2>&1)
if echo "$OUTPUT" | grep -qiE "success|Execute faucet|confirmed|Hash"; then
    ok
else
    echo "$OUTPUT" | tail -3
    fail "Faucet failed"
    exit 1
fi
sleep 3

# ─── Step 4: Set cost for vc_add ─────────────────────────────────────────────

log "Setting cost.vc_add..."
OUTPUT=$($ZWALLET mn-update-config --keys 'cost.vc_add' --values 361 --config $CONFIG --wallet $WALLET 2>&1)
if echo "$OUTPUT" | grep -qiE "success|confirmed|updated"; then
    ok
else
    echo "$OUTPUT" | tail -2
    echo -e "${YELLOW}  (may already be set, continuing)${NC}"
fi
sleep 3

# ─── Step 5: Add hardforks ───────────────────────────────────────────────────

log "Adding hardforks (all at round 0)..."
OUTPUT=$($ZWALLET add-hardfork --names 'apollo,ares,artemis,athena,demeter,electra,hercules,hermes,Medea,Jason' --rounds '0,0,0,0,0,0,0,0,0,0' --wallet $WALLET --config $CONFIG 2>&1)
if echo "$OUTPUT" | grep -qiE "success|confirmed|updated|Hash"; then
    ok
else
    echo "$OUTPUT" | tail -2
    echo -e "${YELLOW}  (may already be set, continuing)${NC}"
fi
sleep 3

# ─── Step 6: Set VC round durations ──────────────────────────────────────────

log "Setting VC round durations (10,20,10,10,20)..."
OUTPUT=$($ZWALLET mn-update-config --keys 'vc_rounds.start,vc_rounds.contribute,vc_rounds.share,vc_rounds.publish,vc_rounds.wait' --values '10,20,10,10,20' --config $CONFIG --wallet $WALLET 2>&1)
if echo "$OUTPUT" | grep -qiE "success|confirmed|updated|Hash"; then
    ok
else
    echo "$OUTPUT" | tail -2
    fail "Failed to set VC round durations"
fi
sleep 3

# ─── Step 7: Set k_percent and x_percent ─────────────────────────────────────

log "Setting k_percent=0.6, x_percent=0.6..."
OUTPUT=$($ZWALLET mn-update-config --keys 'k_percent,x_percent' --values '0.6,0.6' --wallet $WALLET --config $CONFIG 2>&1)
if echo "$OUTPUT" | grep -qiE "success|confirmed|updated|Hash"; then
    ok
else
    echo "$OUTPUT" | tail -2
    fail "Failed to set k/x percent"
fi
sleep 3

# ─── Step 8: Set min_n and min_s ─────────────────────────────────────────────

log "Setting min_n=2, min_s=1..."
OUTPUT=$($ZWALLET mn-update-config --keys 'min_n,min_s' --values '2,1' --wallet $WALLET --config $CONFIG 2>&1)
if echo "$OUTPUT" | grep -qiE "success|confirmed|updated|Hash"; then
    ok
else
    echo "$OUTPUT" | tail -2
    fail "Failed to set min_n/min_s"
fi
sleep 3

# ─── Step 9: Enable view change ──────────────────────────────────────────────

log "Enabling view change..."
OUTPUT=$($ZWALLET global-update-config --keys 'server_chain.view_change' --values true --wallet $WALLET --config $CONFIG 2>&1)
if echo "$OUTPUT" | grep -qiE "success|confirmed|updated|Hash"; then
    ok
else
    echo "$OUTPUT" | tail -2
    fail "Failed to enable view change"
fi
sleep 3

# ─── Step 10: Set block proposal max wait time ───────────────────────────────

log "Setting block proposal max_wait_time=500ms..."
OUTPUT=$($ZWALLET global-update-config --keys "server_chain.block.proposal.max_wait_time" --values "500ms" --config $CONFIG --wallet $WALLET 2>&1)
if echo "$OUTPUT" | grep -qiE "success|confirmed|updated|Hash"; then
    ok
else
    echo "$OUTPUT" | tail -2
    fail "Failed to set max_wait_time"
fi

# ─── Done ─────────────────────────────────────────────────────────────────────

echo ""
log "${GREEN}View change setup complete!${NC}"
log "Chain should start view change cycles within ~70 rounds."
echo ""

# Verify settings
log "Verifying settings..."
curl -s "http://localhost:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs" 2>/dev/null | python3 -c "
import json, sys
d = json.load(sys.stdin).get('fields', {})
keys = ['vc_rounds.start','vc_rounds.contribute','vc_rounds.share','vc_rounds.publish','vc_rounds.wait','k_percent','x_percent','min_n','min_s']
for k in keys:
    print(f'  {k}: {d.get(k, \"?\")}')
" 2>/dev/null || echo "  (could not fetch config)"

curl -s "http://localhost:7171/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings" 2>/dev/null | python3 -c "
import json, sys
d = json.load(sys.stdin).get('fields', {})
print(f'  server_chain.view_change: {d.get(\"server_chain.view_change\", \"?\")}')
print(f'  server_chain.block.proposal.max_wait_time: {d.get(\"server_chain.block.proposal.max_wait_time\", \"?\")}')
" 2>/dev/null || echo "  (could not fetch global settings)"

echo ""
