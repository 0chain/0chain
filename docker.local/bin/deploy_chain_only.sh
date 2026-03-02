#!/bin/bash

# Local 0Chain Deploy Script
#
# One-command setup for a local dev/test chain with view change, chaos testing,
# and monitoring. Merges the best of the former setup_chain.sh and deploy_local.sh.
#
# Steps: loopback → clean → build → start sharders/miners/0dns → wait for blocks
#        → fund wallet → configure VC → launch vc.sh, chaos.sh, monitor.sh
#
# Usage: ./docker.local/bin/deploy_local.sh [OPTIONS]
#
# Options:
#   --no-clean      Skip chain data cleanup (preserve existing data)
#   --no-build      Skip Docker image builds
#   --no-loopback   Skip Mac loopback interface setup
#   --no-start      Skip container startup (just run VC config + scripts)
#   --no-vc         Skip view change configuration transactions
#   --no-chaos      Skip launching chaos.sh
#   --no-monitor    Skip launching monitor.sh
#   --no-scripts    Skip launching all scripts (vc.sh, chaos.sh, monitor.sh)
#   --help          Show this help message
#
# Examples:
#   ./docker.local/bin/deploy_local.sh                          # Full deploy from scratch
#   ./docker.local/bin/deploy_local.sh --no-clean --no-build    # Restart existing chain
#   ./docker.local/bin/deploy_local.sh --no-clean --no-build --no-start  # Just VC config + scripts
#
# Environment variables (override defaults):
#   DNS_DIR       Path to 0dns repo (default: ../0dns relative to chain repo)
#   ZWALLET_DIR   Path to zwalletcli repo (default: ../zwalletcli relative to chain repo)

set -e

# ─── Paths (portable) ────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_LOCAL="$REPO_ROOT/docker.local"
BIN_DIR="$DOCKER_LOCAL/bin"

# Overridable via environment variables
: "${DNS_DIR:=$(cd "$REPO_ROOT/.." && pwd)/0dns}"
: "${ZWALLET_DIR:=$(cd "$REPO_ROOT/.." && pwd)/zwalletcli}"
ZWALLET_PATH="$ZWALLET_DIR/zwallet"
WALLET="local.json"
OWNER_WALLET="local_owner.json"
CONFIG="local.yaml"

NUM_MINERS=4
NUM_SHARDERS=2

# ─── Options ──────────────────────────────────────────────────────────────────

SKIP_CLEAN=false
SKIP_BUILD=false
SKIP_LOOPBACK=false
SKIP_START=false
SKIP_VC=false
SKIP_CHAOS=false
SKIP_MONITOR=false
SKIP_SCRIPTS=false

for arg in "$@"; do
    case $arg in
        --no-clean)    SKIP_CLEAN=true ;;
        --no-build)    SKIP_BUILD=true ;;
        --no-loopback) SKIP_LOOPBACK=true ;;
        --no-start)    SKIP_START=true ;;
        --no-vc)       SKIP_VC=true ;;
        --no-chaos)    SKIP_CHAOS=true ;;
        --no-monitor)  SKIP_MONITOR=true ;;
        --no-scripts)  SKIP_SCRIPTS=true ;;
        --help|-h)
            sed -n '3,/^$/p' "$0" | sed 's/^# \?//'
            exit 0
            ;;
        *) echo "Unknown option: $arg (use --help)"; exit 1 ;;
    esac
done

if [ "$SKIP_SCRIPTS" = true ]; then
    SKIP_CHAOS=true
    SKIP_MONITOR=true
fi

# ─── Colors ───────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log()  { echo -e "${CYAN}[deploy]${NC} $1"; }
ok()   { echo -e "${GREEN}[deploy]${NC} $1"; }
warn() { echo -e "${YELLOW}[deploy]${NC} $1"; }
err()  { echo -e "${RED}[deploy]${NC} $1"; }

# ─── Helpers ──────────────────────────────────────────────────────────────────

# Get highest LFB round across all miners and sharders
get_round() {
    local best=0
    for port in 7171 7172 7071 7072 7073 7074; do
        local r=$(curl -s --connect-timeout 2 "http://localhost:${port}/v1/block/get/latest_finalized" 2>/dev/null | \
            python3 -c "import json,sys; print(json.load(sys.stdin).get('round',0))" 2>/dev/null)
        if [ -n "$r" ] && [ "$r" -gt "$best" ] 2>/dev/null; then
            best=$r
        fi
    done
    echo "${best:-0}"
}

# Get current MB number from 0dns (accurate) with diagnostics fallback
get_mb() {
    # Primary: 0dns magic_block endpoint
    local mb=$(curl -s --connect-timeout 2 "http://localhost:9091/v1/block/get/latest_finalized_magic_block" 2>/dev/null | \
        python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('magic_block',{}).get('magic_block_number',0))" 2>/dev/null)
    if [ -n "$mb" ] && [ "$mb" != "null" ] && [ "$mb" -gt 0 ] 2>/dev/null; then
        echo "$mb"
        return
    fi
    # Fallback: diagnostics page
    local round=$(curl -s --connect-timeout 2 "http://localhost:7071/_diagnostics" 2>/dev/null | \
        grep -oE "LFMB</td><td[^>]*>([0-9]+)" | grep -oE "[0-9]+" | head -1)
    echo "${round:-0}"
}

# Run a zwallet command with retry and exponential backoff
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

        # Authorization error — don't retry
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

is_running() {
    docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${1}$"
}

# ─── Print plan ───────────────────────────────────────────────────────────────

echo -e "\n${BOLD}Local 0Chain Deploy${NC}"
echo -e "  Repo:    $REPO_ROOT"
echo -e "  0dns:    $DNS_DIR"
echo -e "  zwallet: $ZWALLET_DIR"
skipped=""
[ "$SKIP_CLEAN" = true ]    && skipped="$skipped clean"
[ "$SKIP_BUILD" = true ]    && skipped="$skipped build"
[ "$SKIP_LOOPBACK" = true ] && skipped="$skipped loopback"
[ "$SKIP_START" = true ]    && skipped="$skipped start"
[ "$SKIP_VC" = true ]       && skipped="$skipped vc"
[ "$SKIP_CHAOS" = true ]    && skipped="$skipped chaos"
[ "$SKIP_MONITOR" = true ]  && skipped="$skipped monitor"
[ -n "$skipped" ] && echo -e "  ${YELLOW}Skipping:${skipped}${NC}"

# ═════════════════════════════════════════════════════════════════════════════
# Step 1: Loopback aliases
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_LOOPBACK" = true ]; then
    warn "Skipping loopback setup (--no-loopback)"
else
    echo -e "\n${BOLD}═══ Step 1: Loopback Interfaces ═══${NC}"
    if ifconfig lo0 2>/dev/null | grep -q "198.18.0.71"; then
        ok "Loopback aliases already configured"
    else
        log "Setting up loopback aliases (requires sudo)..."
        # Miners
        for i in 1 2 3 4; do sudo ifconfig lo0 alias 198.18.0.$((70 + i)); done
        # Sharders
        for i in 1 2; do sudo ifconfig lo0 alias 198.18.0.$((80 + i)); done
        # 0dns
        sudo ifconfig lo0 alias 198.18.0.100
        # Blobbers
        for ip in 97 98 99 110 111 112; do sudo ifconfig lo0 alias 198.18.0.$ip; done
        ok "Loopback aliases configured (.71-.74 miners, .81-.82 sharders, .100 0dns, .97-.99/.110-.112 blobbers)"
    fi
fi

# ═════════════════════════════════════════════════════════════════════════════
# Step 2: Kill stale scripts + Stop containers + Clean
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_START" = true ]; then
    warn "Skipping container management (--no-start)"
else

echo -e "\n${BOLD}═══ Step 2: Stop + Clean ═══${NC}"

# Kill stale background scripts
for script in vc.sh chaos.sh monitor.sh; do
    pids=$(pgrep -f "$script" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        warn "Killing $script (PIDs: $pids)"
        pkill -f "$script" 2>/dev/null || true
        sleep 1
    fi
done

# Stop all containers
log "Stopping all containers..."
docker stop $(docker ps -a -q) 2>/dev/null || true
ok "All containers stopped"

# Clean chain data
if [ "$SKIP_CLEAN" = false ]; then
    log "Cleaning chain data..."
    cd "$REPO_ROOT"
    "$BIN_DIR/clean.sh"
    "$BIN_DIR/init.setup.sh"
    ok "Chain data cleaned and directories initialized"
else
    warn "Skipping clean (--no-clean)"
fi

# ═════════════════════════════════════════════════════════════════════════════
# Step 3: Build images
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_BUILD" = false ]; then
    echo -e "\n${BOLD}═══ Step 3: Build Images ═══${NC}"
    log "Building miner and sharder images..."
    cd "$REPO_ROOT"
    "$BIN_DIR/build.sharders.sh" &
    "$BIN_DIR/build.miners.sh" &
    wait
    ok "Images built"
else
    warn "Skipping build (--no-build)"
fi

# ═════════════════════════════════════════════════════════════════════════════
# Step 4: Ensure directories exist
# ═════════════════════════════════════════════════════════════════════════════

echo -e "\n${BOLD}═══ Step 4: Init Directories ═══${NC}"
for i in $(seq 1 $NUM_MINERS); do
    mkdir -p "$DOCKER_LOCAL/miner${i}/data/redis/state"
    mkdir -p "$DOCKER_LOCAL/miner${i}/data/redis/transactions"
    mkdir -p "$DOCKER_LOCAL/miner${i}/data/rocksdb"
    mkdir -p "$DOCKER_LOCAL/miner${i}/log"
done
for i in $(seq 1 $NUM_SHARDERS); do
    mkdir -p "$DOCKER_LOCAL/sharder${i}/data/blocks"
    mkdir -p "$DOCKER_LOCAL/sharder${i}/data/rocksdb"
    mkdir -p "$DOCKER_LOCAL/sharder${i}/data/postgresql"
    mkdir -p "$DOCKER_LOCAL/sharder${i}/data/postgresql2"
    mkdir -p "$DOCKER_LOCAL/sharder${i}/log"
done
ok "Directories ready"

# ═════════════════════════════════════════════════════════════════════════════
# Step 5: Start sharders
# ═════════════════════════════════════════════════════════════════════════════

echo -e "\n${BOLD}═══ Step 5: Start Sharders ═══${NC}"
for i in $(seq 1 $NUM_SHARDERS); do
    if is_running "sharder-${i}"; then
        ok "sharder-${i} already running"
    else
        log "Starting sharder-${i}..."
        (cd "$DOCKER_LOCAL/sharder${i}" && ../bin/start.b0sharder.sh) &
    fi
done
wait
sleep 3

for i in $(seq 1 $NUM_SHARDERS); do
    if is_running "sharder-${i}"; then
        ok "sharder-${i} running"
    else
        err "sharder-${i} failed to start!"
    fi
done

# ═════════════════════════════════════════════════════════════════════════════
# Step 6: Start 0dns
# ═════════════════════════════════════════════════════════════════════════════

echo -e "\n${BOLD}═══ Step 6: Start 0dns ═══${NC}"
if is_running "0dns-0dns-1" || docker ps --format '{{.Names}}' 2>/dev/null | grep -q "0dns"; then
    ok "0dns already running"
else
    if [ -d "$DNS_DIR/docker.local" ]; then
        log "Starting 0dns..."
        cd "$DNS_DIR"
        ./docker.local/bin/start.sh
        sleep 3
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "0dns"; then
            ok "0dns running"
        else
            warn "0dns may have failed to start — VC script needs it"
        fi
    else
        warn "0dns not found at $DNS_DIR — skipping (VC script needs it)"
    fi
fi

# ═════════════════════════════════════════════════════════════════════════════
# Step 7: Start all miners simultaneously
# ═════════════════════════════════════════════════════════════════════════════

echo -e "\n${BOLD}═══ Step 7: Start Miners (all simultaneously) ═══${NC}"
NEED_MINERS=false
for i in $(seq 1 $NUM_MINERS); do
    if is_running "miner-${i}"; then
        ok "miner-${i} already running"
    else
        NEED_MINERS=true
        log "Starting miner-${i}..."
        (cd "$DOCKER_LOCAL/miner${i}" && ../bin/start.b0miner.sh) &
    fi
done
wait

for i in $(seq 1 $NUM_MINERS); do
    if is_running "miner-${i}"; then
        ok "miner-${i} running"
    else
        err "miner-${i} failed to start!"
    fi
done

# ═════════════════════════════════════════════════════════════════════════════
# Step 8: Wait for chain to produce blocks
# ═════════════════════════════════════════════════════════════════════════════

echo -e "\n${BOLD}═══ Step 8: Wait for Chain ═══${NC}"
log "Waiting for chain to produce blocks (up to 300s)..."
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
    echo "  Check logs: docker logs miner-1 2>&1 | tail -50"
    exit 1
fi

log "Waiting 15s for chain to stabilize..."
sleep 15

fi  # end SKIP_START

# ═════════════════════════════════════════════════════════════════════════════
# Step 9: Configure view change
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_VC" = true ]; then
    warn "Skipping VC configuration (--no-vc)"
else

echo -e "\n${BOLD}═══ Step 9: Configure View Change ═══${NC}"

if [ ! -f "$ZWALLET_PATH" ]; then
    err "zwallet not found at $ZWALLET_PATH"
    echo "  Build it: cd $ZWALLET_DIR && make build"
    exit 1
fi

# Fund wallets
run_cmd "Funding main wallet (10M tokens)..." \
    "$ZWALLET_PATH faucet --methodName pour --input '{Pay day}' --tokens 10000000 --wallet $WALLET --config $CONFIG"

run_cmd "Funding owner wallet (100 tokens)..." \
    "$ZWALLET_PATH faucet --methodName pour --input '{}' --tokens 100 --wallet $OWNER_WALLET --config $CONFIG"

# Set cost.vc_add (requires owner wallet)
run_cmd "Setting cost.vc_add=361..." \
    "$ZWALLET_PATH mn-update-config --keys 'cost.vc_add' --values 361 --config $CONFIG --network network.yaml --wallet $OWNER_WALLET --configDir $ZWALLET_DIR"

# Add all hardforks at round 0 (including Nyx)
run_cmd "Adding hardforks at round 0 (including Nyx)..." \
    "$ZWALLET_PATH add-hardfork --names 'apollo,ares,artemis,athena,demeter,electra,hercules,hermes,Medea,Jason,Nyx' --rounds '0,0,0,0,0,0,0,0,0,0,0' --wallet $OWNER_WALLET --config $CONFIG"

# Set VC round durations
run_cmd "Setting VC round durations (10,20,10,10,20)..." \
    "$ZWALLET_PATH mn-update-config --keys 'vc_rounds.start,vc_rounds.contribute,vc_rounds.share,vc_rounds.publish,vc_rounds.wait' --values 10,20,10,10,20 --config $CONFIG --wallet $OWNER_WALLET"

# Set k_percent and x_percent
run_cmd "Setting k_percent=0.6, x_percent=0.6..." \
    "$ZWALLET_PATH mn-update-config --keys 'k_percent,x_percent' --values '0.6,0.6' --wallet $OWNER_WALLET --config $CONFIG"

# Set minimum miners and sharders
run_cmd "Setting min_n=2, min_s=1..." \
    "$ZWALLET_PATH mn-update-config --keys 'min_n,min_s' --values 2,1 --wallet $OWNER_WALLET --config $CONFIG"

# Enable view change
run_cmd "Enabling view change..." \
    "$ZWALLET_PATH global-update-config --keys 'server_chain.view_change' --values true --wallet $OWNER_WALLET --config $CONFIG"

# Set block proposal max wait time
run_cmd "Setting block proposal max_wait_time=500ms..." \
    "$ZWALLET_PATH global-update-config --keys 'server_chain.block.proposal.max_wait_time' --values '500ms' --config $CONFIG --wallet $OWNER_WALLET"

log "Waiting 30s for transactions to be processed..."
sleep 30

fi  # end SKIP_VC

# ═════════════════════════════════════════════════════════════════════════════
# Step 10: Launch vc.sh
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_SCRIPTS" = true ]; then
    warn "Skipping all scripts (--no-scripts)"
else

echo -e "\n${BOLD}═══ Step 10: Launch Scripts ═══${NC}"

# Kill any existing instances
for script in vc.sh chaos.sh monitor.sh; do
    pids=$(pgrep -f "$script" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        log "Killing existing $script (PIDs: $pids)..."
        pkill -f "$script" 2>/dev/null || true
        sleep 1
    fi
done

# Wait for VC config to take effect
if [ "$SKIP_VC" != true ]; then
    log "Waiting 15s for VC config to take effect..."
    sleep 15
fi

# Start vc.sh
if [ -f "$BIN_DIR/vc.sh" ]; then
    log "Starting vc.sh → /tmp/vc.log"
    cd "$REPO_ROOT"
    nohup "$BIN_DIR/vc.sh" > /tmp/vc.log 2>&1 &
    VC_PID=$!
    ok "vc.sh started (PID=$VC_PID)"

    # Wait for first view change (MB >= 2)
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

    # Monitor stability for 60s after first VC
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
            ok "Chain stable after view change ($STABLE_CHECKS/6 progress checks passed)"
        else
            warn "Chain instability detected ($STABLE_CHECKS/6 progress checks passed)"
        fi
    fi
else
    warn "vc.sh not found at $BIN_DIR/vc.sh"
fi

# Start chaos.sh
if [ "$SKIP_CHAOS" = true ]; then
    warn "Skipping chaos.sh (--no-chaos)"
elif [ -f "$BIN_DIR/chaos.sh" ]; then
    log "Starting chaos.sh → /tmp/chaos.log"
    cd "$REPO_ROOT"
    nohup "$BIN_DIR/chaos.sh" > /tmp/chaos.log 2>&1 &
    CHAOS_PID=$!
    ok "chaos.sh started (PID=$CHAOS_PID)"
else
    warn "chaos.sh not found at $BIN_DIR/chaos.sh"
fi

# Start monitor.sh
if [ "$SKIP_MONITOR" = true ]; then
    warn "Skipping monitor.sh (--no-monitor)"
elif [ -f "$BIN_DIR/monitor.sh" ]; then
    log "Starting monitor.sh (60s interval) → /tmp/monitor.log"
    nohup "$BIN_DIR/monitor.sh" 60 > /tmp/monitor.log 2>&1 &
    MONITOR_PID=$!
    ok "monitor.sh started (PID=$MONITOR_PID)"
else
    warn "monitor.sh not found at $BIN_DIR/monitor.sh"
fi

fi  # end SKIP_SCRIPTS

# ═════════════════════════════════════════════════════════════════════════════
# Summary
# ═════════════════════════════════════════════════════════════════════════════

echo ""
echo -e "${BOLD}═══ Summary ═══${NC}"

echo -e "\n${CYAN}── Containers ──${NC}"
for c in sharder-1 sharder-2 miner-1 miner-2 miner-3 miner-4; do
    if is_running "$c"; then
        printf "  %-12s ${GREEN}UP${NC}\n" "$c"
    else
        printf "  %-12s ${RED}DOWN${NC}\n" "$c"
    fi
done
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "0dns"; then
    printf "  %-12s ${GREEN}UP${NC}\n" "0dns"
else
    printf "  %-12s ${RED}DOWN${NC}\n" "0dns"
fi

ROUND=$(get_round)
MB=$(get_mb)
echo -e "\n  Chain: round=${GREEN}${ROUND:-?}${NC}, MB=#${GREEN}${MB:-?}${NC}"

if [ "$SKIP_VC" != true ]; then
    echo -e "\n${CYAN}── VC Settings ──${NC}"
    for port in 7171 7172; do
        CONFIG_JSON=$(curl -s --connect-timeout 2 "http://localhost:${port}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs" 2>/dev/null)
        if [ -n "$CONFIG_JSON" ]; then
            echo "$CONFIG_JSON" | python3 -c "
import json, sys
d = json.load(sys.stdin).get('fields', {})
keys = ['vc_rounds.start','vc_rounds.contribute','vc_rounds.share','vc_rounds.publish','vc_rounds.wait','k_percent','x_percent','min_n','min_s']
for k in keys:
    print(f'  {k}: {d.get(k, \"?\")}')" 2>/dev/null
            break
        fi
    done
fi

if [ "$SKIP_SCRIPTS" != true ]; then
    echo -e "\n${CYAN}── Background Scripts ──${NC}"
    for script in vc.sh chaos.sh monitor.sh; do
        pid=$(pgrep -f "$script" 2>/dev/null | head -1 || true)
        logfile="/tmp/${script%.sh}.log"
        if [ -n "$pid" ]; then
            printf "  %-14s ${GREEN}running${NC} (PID %s) → %s\n" "$script" "$pid" "$logfile"
        else
            printf "  %-14s ${YELLOW}not running${NC}\n" "$script"
        fi
    done
fi

echo -e "\n${GREEN}${BOLD}Deploy complete!${NC}"
echo -e "  Diag:  http://localhost:7071/_diagnostics"
echo -e "  0dns:  http://localhost:9091/network"
echo -e "  Logs:  tail -f /tmp/vc.log /tmp/chaos.log /tmp/monitor.log"
echo ""
