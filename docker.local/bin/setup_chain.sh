#!/bin/bash

# Setup Local 0Chain for Development & Testing
#
# This script does everything needed to get a local chain running with view change,
# chaos testing, and monitoring:
#   1. Docker network + init directories
#   2. Start sharders and miners
#   3. Start 0dns (network discovery service)
#   4. Setup Mac loopback interfaces (for Docker testnet0 network)
#   5. Wait for chain to produce blocks
#   6. Fund wallet and configure view change settings
#   7. Launch vc.sh, chaos.sh, and monitor.sh in background
#
# Usage: ./docker.local/bin/setup_chain.sh [--skip items]
#
# Options:
#   --skip items   Comma-separated list of items to skip:
#                    loopback  - Mac loopback interface setup
#                    chain     - Container startup (Docker network, sharders, miners, 0dns)
#                    vc        - View change configuration
#                    scripts   - Launching vc.sh, chaos.sh, monitor.sh
#
# Examples:
#   ./docker.local/bin/setup_chain.sh                        # Full setup
#   ./docker.local/bin/setup_chain.sh --skip vc,scripts      # Start chain only
#   ./docker.local/bin/setup_chain.sh --skip chain,loopback  # VC config + scripts only
#
# Prerequisites:
#   - Docker Desktop running
#   - Miner and sharder images built (build.miners.sh, build.sharders.sh)
#   - 0dns image built (cd ~/Code/0dns && make build)
#   - zwallet binary at ~/Code/zwalletcli/zwallet (for VC config)

set -e

# ─── Config ──────────────────────────────────────────────────────────────────

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DOCKER_LOCAL="$REPO_ROOT/docker.local"
BIN_DIR="$DOCKER_LOCAL/bin"
DNS_DIR="/Users/saswatabasu/Code/0dns/docker.local"
ZWALLET_DIR="/Users/saswatabasu/Code/zwalletcli"
ZWALLET="$ZWALLET_DIR/zwallet"
WALLET="local.json"
CONFIG="local.yaml"

NUM_MINERS=4
NUM_SHARDERS=2

SKIP_LOOPBACK=false
SKIP_CHAIN=false
SKIP_VC=false
SKIP_SCRIPTS=false

# Parse --skip flag
while [ $# -gt 0 ]; do
    case "$1" in
        --skip)
            shift
            IFS=',' read -ra ITEMS <<< "$1"
            for item in "${ITEMS[@]}"; do
                case "$(echo "$item" | tr '[:upper:]' '[:lower:]' | xargs)" in
                    loopback)  SKIP_LOOPBACK=true ;;
                    chain)     SKIP_CHAIN=true ;;
                    vc)        SKIP_VC=true ;;
                    scripts)   SKIP_SCRIPTS=true ;;
                    *) echo "Unknown skip item: $item (valid: loopback, chain, vc, scripts)"; exit 1 ;;
                esac
            done
            ;;
    esac
    shift
done

# ─── Colors ──────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log()  { echo -e "${CYAN}[$(date '+%H:%M:%S')]${NC} $1"; }
ok()   { echo -e "  ${GREEN}OK${NC}"; }
warn() { echo -e "  ${YELLOW}$1${NC}"; }
fail() { echo -e "  ${RED}FAILED: $1${NC}"; }

run_zwallet() {
    local desc="$1"
    shift
    log "$desc"
    local output
    output=$(cd "$ZWALLET_DIR" && "$ZWALLET" "$@" 2>&1)
    if echo "$output" | grep -qiE "success|confirmed|updated|Hash|Execute faucet"; then
        ok
    else
        echo "$output" | tail -3
        warn "(may already be set, continuing)"
    fi
    sleep 3
}

get_round() {
    for port in 7171 7172; do
        local r=$(curl -s "http://localhost:${port}/v1/block/get/latest_finalized" 2>/dev/null | \
            python3 -c "import json,sys; print(json.load(sys.stdin).get('round',0))" 2>/dev/null)
        if [ -n "$r" ] && [ "$r" != "0" ]; then
            echo "$r"
            return
        fi
    done
    echo "0"
}

# ─── Print what we're doing ──────────────────────────────────────────────────

echo -e "\n${BOLD}Local 0Chain Setup${NC}"
skipped=""
[ "$SKIP_CHAIN" = true ] && skipped="$skipped chain"
[ "$SKIP_LOOPBACK" = true ] && skipped="$skipped loopback"
[ "$SKIP_VC" = true ] && skipped="$skipped vc"
[ "$SKIP_SCRIPTS" = true ] && skipped="$skipped scripts"
if [ -n "$skipped" ]; then
    echo -e "${YELLOW}Skipping:${skipped}${NC}"
fi

# ═════════════════════════════════════════════════════════════════════════════
# Step 1: Docker Network + Init Directories + Start Containers + 0dns
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_CHAIN" = true ]; then
    echo -e "\n${YELLOW}Skipping chain startup${NC}"
else

# ─── Docker Network ──────────────────────────────────────────────────────────

echo -e "\n${BOLD}═══ Step 1: Docker Network ═══${NC}"

if docker network inspect testnet0 >/dev/null 2>&1; then
    log "testnet0 network already exists"
    ok
else
    log "Creating testnet0 network (198.18.0.0/16)..."
    docker network create --driver bridge --subnet 198.18.0.0/16 testnet0
    ok
fi

# ─── Init directories ───────────────────────────────────────────────────────

echo -e "\n${BOLD}═══ Step 2: Init Directories ═══${NC}"

log "Ensuring data/log directories exist..."
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
    mkdir -p "$DOCKER_LOCAL/sharder${i}/log"
done
ok

# ─── Start Sharders ──────────────────────────────────────────────────────────

echo -e "\n${BOLD}═══ Step 3: Start Sharders (first for genesis) ═══${NC}"

for i in $(seq 1 $NUM_SHARDERS); do
    container="sharder-${i}"
    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        log "sharder-${i} already running"
        ok
    else
        log "Starting sharder-${i}..."
        (cd "$DOCKER_LOCAL/sharder${i}" && SHARDER=$i docker compose -p sharder${i} -f ../build.sharder/b0docker-compose.yml up -d)
        ok
    fi
done

log "Waiting 10s for sharders to initialize..."
sleep 10

# ─── Start Miners ────────────────────────────────────────────────────────────

echo -e "\n${BOLD}═══ Step 4: Start Miners ═══${NC}"

for i in $(seq 1 $NUM_MINERS); do
    container="miner-${i}"
    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        log "miner-${i} already running"
        ok
    else
        log "Starting miner-${i}..."
        (cd "$DOCKER_LOCAL/miner${i}" && MINER=$i docker compose -p miner${i} -f ../build.miner/b0docker-compose.yml up -d)
        ok
    fi
done

# ─── Start 0dns ──────────────────────────────────────────────────────────────

echo -e "\n${BOLD}═══ Step 5: Start 0dns ═══${NC}"

if docker ps --format '{{.Names}}' | grep -q "0dns"; then
    log "0dns already running"
    ok
else
    if [ -f "$DNS_DIR/docker-compose.yml" ]; then
        log "Starting 0dns..."
        (cd "$DNS_DIR" && docker compose -p dockerlocal up -d)
        ok
    else
        warn "0dns not found at $DNS_DIR — skipping (VC script needs it)"
    fi
fi

fi  # end SKIP_CHAIN

# ═════════════════════════════════════════════════════════════════════════════
# Step 2: Mac Loopback Interfaces
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_LOOPBACK" = true ]; then
    echo -e "\n${YELLOW}Skipping loopback setup${NC}"
else

echo -e "\n${BOLD}═══ Step 6: Mac Loopback Interfaces ═══${NC}"

if ifconfig lo0 2>/dev/null | grep -q "198.18.0.71"; then
    log "Loopback interfaces already configured"
    ok
else
    log "Setting up loopback interfaces..."

    # Miners: 198.18.0.71-74
    for i in 1 2 3 4; do
        ifconfig lo0 alias 198.18.0.$((70 + i))
    done

    # Sharders: 198.18.0.81-82
    for i in 1 2; do
        ifconfig lo0 alias 198.18.0.$((80 + i))
    done

    # 0dns
    ifconfig lo0 alias 198.18.0.100

    # Blobbers: 198.18.0.97-99, 110-112
    for ip in 97 98 99 110 111 112; do
        ifconfig lo0 alias 198.18.0.$ip
    done

    ok
    log "IPs: miners .71-.74, sharders .81-.82, 0dns .100, blobbers .97-.99/.110-.112"
fi

fi  # end SKIP_LOOPBACK

# ═════════════════════════════════════════════════════════════════════════════
# Step 3: Wait for chain + Configure View Change
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_VC" = true ]; then
    echo -e "\n${YELLOW}Skipping view change configuration${NC}"
else

# ─── Wait for Chain ──────────────────────────────────────────────────────────

echo -e "\n${BOLD}═══ Step 7: Wait for Chain ═══${NC}"

log "Waiting for chain to produce blocks..."
MAX_WAIT=120
for i in $(seq 1 $MAX_WAIT); do
    ROUND=$(get_round)
    if [ "$ROUND" -gt 0 ] 2>/dev/null; then
        echo -e "  ${GREEN}Chain running at round $ROUND${NC}"
        break
    fi
    if [ "$i" -eq "$MAX_WAIT" ]; then
        fail "Chain not producing blocks after ${MAX_WAIT}s"
        echo "  Check logs: docker logs sharder-1, docker logs miner-1"
        exit 1
    fi
    [ $((i % 10)) -eq 0 ] && echo -e "  ${YELLOW}Still waiting... (${i}s)${NC}"
    sleep 1
done

# ─── Configure View Change ───────────────────────────────────────────────────

echo -e "\n${BOLD}═══ Step 8: Configure View Change ═══${NC}"

if [ ! -f "$ZWALLET" ]; then
    fail "zwallet not found at $ZWALLET"
    echo "  Build it: cd ~/Code/zwalletcli && make build"
    exit 1
fi

run_zwallet "Funding wallet from faucet (10M tokens)..." \
    faucet --methodName pour --input "{Pay day}" --tokens 10000000 --config $CONFIG --wallet $WALLET

run_zwallet "Setting cost.vc_add=361..." \
    mn-update-config --keys 'cost.vc_add' --values 361 --config $CONFIG --wallet $WALLET

run_zwallet "Adding hardforks (all at round 0)..." \
    add-hardfork --names 'apollo,ares,artemis,athena,demeter,electra,hercules,hermes,Medea,Jason' \
    --rounds '0,0,0,0,0,0,0,0,0,0' --wallet $WALLET --config $CONFIG

run_zwallet "Setting VC round durations (10,20,10,10,20)..." \
    mn-update-config --keys 'vc_rounds.start,vc_rounds.contribute,vc_rounds.share,vc_rounds.publish,vc_rounds.wait' \
    --values '10,20,10,10,20' --config $CONFIG --wallet $WALLET

run_zwallet "Setting k_percent=0.6, x_percent=0.6..." \
    mn-update-config --keys 'k_percent,x_percent' --values '0.6,0.6' --wallet $WALLET --config $CONFIG

run_zwallet "Setting min_n=2, min_s=1..." \
    mn-update-config --keys 'min_n,min_s' --values '2,1' --wallet $WALLET --config $CONFIG

run_zwallet "Enabling view change..." \
    global-update-config --keys 'server_chain.view_change' --values true --wallet $WALLET --config $CONFIG

run_zwallet "Setting block proposal max_wait_time=500ms..." \
    global-update-config --keys "server_chain.block.proposal.max_wait_time" --values "500ms" --config $CONFIG --wallet $WALLET

fi  # end SKIP_VC

# ═════════════════════════════════════════════════════════════════════════════
# Step 4: Launch Testing Scripts
# ═════════════════════════════════════════════════════════════════════════════

if [ "$SKIP_SCRIPTS" = true ]; then
    echo -e "\n${YELLOW}Skipping script launch${NC}"
else

echo -e "\n${BOLD}═══ Step 9: Launch Testing Scripts ═══${NC}"

# Kill any existing instances
for script in vc.sh chaos.sh monitor.sh; do
    pids=$(pgrep -f "$script" 2>/dev/null || true)
    if [ -n "$pids" ]; then
        log "Killing existing $script (PIDs: $pids)..."
        pkill -f "$script" 2>/dev/null || true
        sleep 1
    fi
done

# Wait for a few rounds so VC config is active before starting scripts
if [ "$SKIP_VC" != true ]; then
    log "Waiting 15s for VC config to take effect..."
    sleep 15
fi

# Start VC script
if [ -f "$BIN_DIR/vc.sh" ]; then
    log "Starting vc.sh → /tmp/vc.log"
    nohup "$BIN_DIR/vc.sh" > /tmp/vc.log 2>&1 &
    echo -e "  ${GREEN}PID $!${NC}"
else
    warn "vc.sh not found, skipping"
fi

# Start chaos script
if [ -f "$BIN_DIR/chaos.sh" ]; then
    log "Starting chaos.sh → /tmp/chaos.log"
    nohup "$BIN_DIR/chaos.sh" > /tmp/chaos.log 2>&1 &
    echo -e "  ${GREEN}PID $!${NC}"
else
    warn "chaos.sh not found, skipping"
fi

# Start monitor script
if [ -f "$BIN_DIR/monitor.sh" ]; then
    log "Starting monitor.sh (60s interval) → /tmp/monitor.log"
    nohup "$BIN_DIR/monitor.sh" 60 > /tmp/monitor.log 2>&1 &
    echo -e "  ${GREEN}PID $!${NC}"
else
    warn "monitor.sh not found, skipping"
fi

fi  # end SKIP_SCRIPTS

# ═════════════════════════════════════════════════════════════════════════════
# Summary
# ═════════════════════════════════════════════════════════════════════════════

echo -e "\n${BOLD}═══ Summary ═══${NC}"

echo -e "\n${CYAN}── Containers ──${NC}"
for c in sharder-1 sharder-2 miner-1 miner-2 miner-3 miner-4; do
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${c}$"; then
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
echo -e "\n  Chain round: ${GREEN}$ROUND${NC}"

if [ "$SKIP_VC" != true ]; then
    echo -e "\n${CYAN}── VC Settings ──${NC}"
    for port in 7171 7172; do
        CONFIG_JSON=$(curl -s "http://localhost:${port}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs" 2>/dev/null)
        if [ -n "$CONFIG_JSON" ]; then
            echo "$CONFIG_JSON" | python3 -c "
import json, sys
d = json.load(sys.stdin).get('fields', {})
keys = ['vc_rounds.start','vc_rounds.contribute','vc_rounds.share','vc_rounds.publish','vc_rounds.wait','k_percent','x_percent','min_n','min_s']
for k in keys:
    print(f'  {k}: {d.get(k, \"?\")}')
" 2>/dev/null
            break
        fi
    done

    for port in 7171 7172; do
        GLOBAL_JSON=$(curl -s "http://localhost:${port}/v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings" 2>/dev/null)
        if [ -n "$GLOBAL_JSON" ]; then
            echo "$GLOBAL_JSON" | python3 -c "
import json, sys
d = json.load(sys.stdin).get('fields', {})
print(f'  server_chain.view_change: {d.get(\"server_chain.view_change\", \"?\")}')
print(f'  server_chain.block.proposal.max_wait_time: {d.get(\"server_chain.block.proposal.max_wait_time\", \"?\")}')
" 2>/dev/null
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
            printf "  %-14s ${RED}not running${NC}\n" "$script"
        fi
    done
fi

echo -e "\n${GREEN}${BOLD}Setup complete!${NC}"
if [ "$SKIP_VC" != true ]; then
    echo -e "View change should start within ~70 rounds."
fi
if [ "$SKIP_SCRIPTS" != true ]; then
    echo -e "Logs: tail -f /tmp/vc.log /tmp/chaos.log /tmp/monitor.log"
fi
echo ""
