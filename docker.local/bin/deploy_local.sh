#!/bin/bash

# Local Chain Deploy Script
# Automates: loopback -> clean -> build -> start -> setup transactions -> vc.sh -> chaos.sh
# Usage: ./docker.local/bin/deploy_local.sh [--no-clean] [--no-build] [--no-vc] [--no-chaos]

set -e

CHAIN_DIR="/Users/saswatabasu/Code/0chain"
DNS_DIR="/Users/saswatabasu/Code/0dns"
ZWALLET_DIR="/Users/saswatabasu/Code/zwalletcli"
ZWALLET_PATH="$ZWALLET_DIR/zwallet"
WALLET="local.json"
OWNER_WALLET="local_owner.json"
CONFIG="local.yaml"

# Options
SKIP_CLEAN=false
SKIP_BUILD=false
SKIP_VC=false
SKIP_CHAOS=false
for arg in "$@"; do
    case $arg in
        --no-clean) SKIP_CLEAN=true ;;
        --no-build) SKIP_BUILD=true ;;
        --no-vc)    SKIP_VC=true ;;
        --no-chaos) SKIP_CHAOS=true ;;
    esac
done

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'

log() { echo -e "${CYAN}[deploy]${NC} $1"; }
ok()  { echo -e "${GREEN}[deploy]${NC} $1"; }
warn() { echo -e "${YELLOW}[deploy]${NC} $1"; }
err() { echo -e "${RED}[deploy]${NC} $1"; }

# Get current round from diagnostics (macOS-compatible)
get_round() {
    curl -s http://localhost:7071/_diagnostics 2>/dev/null | grep -oE '</span>[0-9]+</a>' | grep -oE '[0-9]+' | head -1
}

# Get current MB number from diagnostics
get_mb() {
    curl -s http://localhost:7071/_diagnostics 2>/dev/null | grep -oE 'LFMB.*[0-9]+' | grep -oE '[0-9]+' | head -1
}

# ============================================================
# Step 0: Check loopback aliases
# ============================================================
log "Checking loopback aliases..."
MISSING_ALIASES=false
for ip in 198.18.0.71 198.18.0.72 198.18.0.73 198.18.0.74 198.18.0.81 198.18.0.82 198.18.0.100; do
    if ! ifconfig lo0 2>/dev/null | grep -q "$ip"; then
        MISSING_ALIASES=true
        break
    fi
done

if [ "$MISSING_ALIASES" = true ]; then
    warn "Setting up loopback aliases (requires sudo)..."
    sudo ifconfig lo0 alias 198.18.0.71
    sudo ifconfig lo0 alias 198.18.0.72
    sudo ifconfig lo0 alias 198.18.0.73
    sudo ifconfig lo0 alias 198.18.0.74
    sudo ifconfig lo0 alias 198.18.0.81
    sudo ifconfig lo0 alias 198.18.0.82
    sudo ifconfig lo0 alias 198.18.0.100
    sudo ifconfig lo0 alias 198.18.0.97
    sudo ifconfig lo0 alias 198.18.0.98
    sudo ifconfig lo0 alias 198.18.0.99
    sudo ifconfig lo0 alias 198.18.0.110
    sudo ifconfig lo0 alias 198.18.0.111
    sudo ifconfig lo0 alias 198.18.0.112
    ok "Loopback aliases configured"
else
    ok "Loopback aliases already configured"
fi

# ============================================================
# Step 1: Kill stale background scripts and stop containers
# ============================================================
log "Killing stale background scripts..."
for script in vc.sh chaos.sh monitor.sh; do
    pids=$(ps aux | grep "[/]$script" | awk '{print $2}')
    if [ -n "$pids" ]; then
        warn "Killing $script (PIDs: $pids)"
        echo "$pids" | xargs kill 2>/dev/null
    fi
done

log "Stopping all containers..."
docker stop $(docker ps -a -q) 2>/dev/null || true
ok "All containers stopped"

# ============================================================
# Step 2: Clean (unless --no-clean)
# ============================================================
if [ "$SKIP_CLEAN" = false ]; then
    log "Cleaning chain data..."
    cd "$CHAIN_DIR"
    ./docker.local/bin/clean.sh
    ./docker.local/bin/init.setup.sh
    ok "Chain data cleaned and directories initialized"
else
    warn "Skipping clean (--no-clean)"
fi

# ============================================================
# Step 3: Build (unless --no-build)
# ============================================================
if [ "$SKIP_BUILD" = false ]; then
    log "Building miner and sharder images..."
    cd "$CHAIN_DIR"
    ./docker.local/bin/build.sharders.sh &
    ./docker.local/bin/build.miners.sh &
    wait
    ok "Images built"
else
    warn "Skipping build (--no-build)"
fi

# ============================================================
# Step 4: Start sharders
# ============================================================
log "Starting sharders..."
cd "$CHAIN_DIR/docker.local/sharder1" && ../bin/start.b0sharder.sh &
cd "$CHAIN_DIR/docker.local/sharder2" && ../bin/start.b0sharder.sh &
wait
sleep 3
if docker ps | grep -q "sharder-1" && docker ps | grep -q "sharder-2"; then
    ok "Both sharders running"
else
    err "Sharder startup failed!"
    docker ps -a --format "table {{.Names}}\t{{.Status}}" | grep sharder
    exit 1
fi

# ============================================================
# Step 5: Start 0dns
# ============================================================
log "Starting 0dns..."
cd "$DNS_DIR"
./docker.local/bin/start.sh
sleep 3
if docker ps | grep -q "0dns"; then
    ok "0dns running"
else
    err "0dns startup failed!"
    exit 1
fi

# ============================================================
# Step 6: Start all miners simultaneously
# ============================================================
log "Starting all 4 miners simultaneously..."
cd "$CHAIN_DIR/docker.local/miner1" && ../bin/start.b0miner.sh &
cd "$CHAIN_DIR/docker.local/miner2" && ../bin/start.b0miner.sh &
cd "$CHAIN_DIR/docker.local/miner3" && ../bin/start.b0miner.sh &
cd "$CHAIN_DIR/docker.local/miner4" && ../bin/start.b0miner.sh &
wait
ok "All miners started"

# ============================================================
# Step 7: Wait for chain to produce blocks
# ============================================================
log "Waiting for chain to start producing blocks (up to 300s)..."
MAX_WAIT=300
ELAPSED=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
    ROUND=$(get_round)
    if [ -n "$ROUND" ] && [ "$ROUND" -gt 5 ] 2>/dev/null; then
        echo ""
        ok "Chain producing blocks (round $ROUND)"
        break
    fi
    sleep 5
    ELAPSED=$((ELAPSED + 5))
    echo -n "."
done

if [ $ELAPSED -ge $MAX_WAIT ]; then
    echo ""
    err "Chain failed to start producing blocks within ${MAX_WAIT}s"
    echo "Check logs: docker logs miner-1 2>&1 | tail -50"
    exit 1
fi

# Wait a bit more for the chain to stabilize
log "Waiting 15s for chain to stabilize..."
sleep 15

# ============================================================
# Step 8: Run setup transactions
# ============================================================
log "Running setup transactions..."

# Run a zwallet command with retry (zwallet handles nonces automatically)
run_cmd() {
    local desc=$1
    shift
    local cmd="$@"
    local max_retries=5
    local retry=0
    local backoff=3

    echo -e "  ${CYAN}$desc${NC}"
    while [ $retry -lt $max_retries ]; do
        local output
        output=$(eval "$cmd --configDir $ZWALLET_DIR" 2>&1)
        local exit_code=$?

        # Success indicators
        if echo "$output" | grep -qiE "success|confirmed|Execute faucet|already|Hash:"; then
            echo -e "    ${GREEN}OK${NC}"
            return 0
        fi

        # Check for authorization error
        if echo "$output" | grep -qiE "unauthorized"; then
            err "    Authorization error — wrong wallet?"
            err "    Output: $(echo "$output" | tail -1)"
            return 1
        fi

        # Network error — retry with backoff
        if echo "$output" | grep -qiE "connection refused|timeout|too less sharders|unexpected end"; then
            retry=$((retry + 1))
            warn "    Network error, retrying in ${backoff}s ($retry/$max_retries)"
            sleep $backoff
            backoff=$((backoff * 2))
            [ $backoff -gt 20 ] && backoff=20
            continue
        fi

        # zwallet returned 0 — treat as success
        if [ $exit_code -eq 0 ]; then
            echo -e "    ${GREEN}OK${NC}"
            return 0
        fi

        retry=$((retry + 1))
        warn "    Failed, retrying in ${backoff}s ($retry/$max_retries)"
        warn "    Output: $(echo "$output" | tail -1)"
        sleep $backoff
        backoff=$((backoff * 2))
        [ $backoff -gt 20 ] && backoff=20
    done

    err "    FAILED after $max_retries retries: $desc"
    return 1
}

# 1. Fund wallets
run_cmd "Funding main wallet (10M tokens)..." \
    "$ZWALLET_PATH faucet --methodName pour --input '{Pay day}' --tokens 10000000 --wallet $WALLET --config $CONFIG"

run_cmd "Funding owner wallet (100 tokens)..." \
    "$ZWALLET_PATH faucet --methodName pour --input '{}' --tokens 100 --wallet $OWNER_WALLET --config $CONFIG"

# 2. Set cost.vc_add
run_cmd "Setting cost.vc_add=361..." \
    "$ZWALLET_PATH mn-update-config --keys 'cost.vc_add' --values 361 --config $CONFIG --network network.yaml --wallet $OWNER_WALLET --configDir $ZWALLET_DIR"

# 3. Add all hardforks at round 0 (including Nyx) — requires owner wallet
run_cmd "Adding hardforks at round 0..." \
    "$ZWALLET_PATH add-hardfork --names 'apollo,ares,artemis,athena,demeter,electra,hercules,hermes,Medea,Jason,Nyx' --rounds '0,0,0,0,0,0,0,0,0,0,0' --wallet $OWNER_WALLET --config $CONFIG"

# 4. Set VC round durations — requires owner wallet
run_cmd "Setting VC round durations..." \
    "$ZWALLET_PATH mn-update-config --keys 'vc_rounds.start,vc_rounds.contribute,vc_rounds.share,vc_rounds.publish,vc_rounds.wait' --values 10,20,10,10,20 --config $CONFIG --wallet $OWNER_WALLET"

# 5. Set k_percent and x_percent — requires owner wallet
run_cmd "Setting k_percent=0.6, x_percent=0.6..." \
    "$ZWALLET_PATH mn-update-config --keys 'k_percent,x_percent' --values '0.6,0.6' --wallet $OWNER_WALLET --config $CONFIG"

# 6. Set minimum miners and sharders — requires owner wallet
run_cmd "Setting min_n=2, min_s=1..." \
    "$ZWALLET_PATH mn-update-config --keys 'min_n,min_s' --values 2,1 --wallet $OWNER_WALLET --config $CONFIG"

# 7. Enable view change — requires owner wallet
run_cmd "Enabling view change..." \
    "$ZWALLET_PATH global-update-config --keys 'server_chain.view_change' --values true --wallet $OWNER_WALLET --config $CONFIG"

# 8. Set block proposal max wait time — requires owner wallet
run_cmd "Setting block proposal max_wait_time=500ms..." \
    "$ZWALLET_PATH global-update-config --keys 'server_chain.block.proposal.max_wait_time' --values '500ms' --config $CONFIG --wallet $OWNER_WALLET"

# ============================================================
# Step 9: Wait for setup to take effect and verify chain stability
# ============================================================
log "Waiting 30s for transactions to be processed..."
sleep 30

ROUND=$(get_round)
ok "Chain running at round ${ROUND:-unknown}"

# ============================================================
# Step 10: Start vc.sh (unless --no-vc)
# ============================================================
if [ "$SKIP_VC" = false ]; then
    log "Starting vc.sh in background..."
    cd "$CHAIN_DIR"
    nohup ./docker.local/bin/vc.sh > /tmp/vc.log 2>&1 &
    VC_PID=$!
    ok "vc.sh started (PID=$VC_PID, log=/tmp/vc.log)"

    # Wait for vc.sh to complete at least one view change (MB >= 2)
    log "Waiting for first view change (MB >= 2, up to 300s)..."
    VC_WAIT=0
    VC_MAX=300
    VC_STABLE=false
    while [ $VC_WAIT -lt $VC_MAX ]; do
        MB=$(get_mb)
        ROUND=$(get_round)
        if [ -n "$MB" ] && [ "$MB" -ge 2 ] 2>/dev/null; then
            echo ""
            ok "View change successful! MB=#${MB}, round=${ROUND}"
            VC_STABLE=true
            break
        fi
        sleep 10
        VC_WAIT=$((VC_WAIT + 10))
        echo -n "."
    done

    if [ "$VC_STABLE" = false ]; then
        echo ""
        warn "View change not yet observed after ${VC_MAX}s (MB=$(get_mb))"
        warn "vc.sh is still running — check /tmp/vc.log"
    fi

    # Monitor chain stability for 60s after first VC
    if [ "$VC_STABLE" = true ]; then
        log "Monitoring chain stability for 60s after view change..."
        PREV_ROUND=$(get_round)
        STABLE_CHECKS=0
        for i in $(seq 1 6); do
            sleep 10
            CUR_ROUND=$(get_round)
            if [ -n "$CUR_ROUND" ] && [ "$CUR_ROUND" -gt "$PREV_ROUND" ] 2>/dev/null; then
                STABLE_CHECKS=$((STABLE_CHECKS + 1))
                echo -e "  ${GREEN}+${NC} round $PREV_ROUND -> $CUR_ROUND"
            else
                echo -e "  ${RED}-${NC} stuck at round ${CUR_ROUND:-unknown}"
            fi
            PREV_ROUND=$CUR_ROUND
        done

        if [ $STABLE_CHECKS -ge 4 ]; then
            ok "Chain is stable after view change ($STABLE_CHECKS/6 progress checks passed)"
        else
            warn "Chain instability detected ($STABLE_CHECKS/6 progress checks passed)"
        fi
    fi
else
    warn "Skipping vc.sh (--no-vc)"
fi

# ============================================================
# Step 11: Start chaos.sh (unless --no-chaos)
# ============================================================
if [ "$SKIP_CHAOS" = false ]; then
    log "Starting chaos.sh in background..."
    cd "$CHAIN_DIR"
    nohup ./docker.local/bin/chaos.sh > /tmp/chaos.log 2>&1 &
    CHAOS_PID=$!
    ok "chaos.sh started (PID=$CHAOS_PID, log=/tmp/chaos.log)"
else
    warn "Skipping chaos.sh (--no-chaos)"
fi

# ============================================================
# Done
# ============================================================
ROUND=$(get_round)
MB=$(get_mb)

echo ""
echo "============================================================"
echo -e "${GREEN}  Deploy complete!${NC}"
echo "============================================================"
echo ""
echo "  Chain:  round=${ROUND:-unknown}, MB=#${MB:-unknown}"
echo "  Diag:   http://localhost:7071/_diagnostics"
echo "  0dns:   http://localhost:9091/network"
echo ""
[ "$SKIP_VC" = false ] && echo "  vc.sh:    PID=${VC_PID:-unknown}, tail -f /tmp/vc.log"
[ "$SKIP_CHAOS" = false ] && echo "  chaos.sh: PID=${CHAOS_PID:-unknown}, tail -f /tmp/chaos.log"
echo ""
