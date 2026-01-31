#!/bin/bash

# Local Container Chaos Test Script
# Randomly stops and starts miner/sharder containers to test resilience and view change
# For local development environment with 4 miners and 3 sharders

set -e

# Configuration
MIN_STOP_TIME=10    # Minimum time container stays stopped (seconds)
MAX_STOP_TIME=30    # Maximum time container stays stopped (seconds)
MIN_RUN_TIME=20     # Minimum time before next chaos operation (seconds)
MAX_RUN_TIME=60     # Maximum time before next chaos operation (seconds)
ITERATIONS=0        # Number of test iterations (0 = infinite)

# Local container names (matching docker-compose naming)
MINERS=("miner-1" "miner-2" "miner-3" "miner-4")
SHARDERS=("sharder-1" "sharder-2")

# Consensus requirements: T=3 for N=4 miners, need at least 3 miners for consensus
MIN_MINERS_RUNNING=3
MIN_SHARDERS_RUNNING=1

# Diagnostics URL for monitoring
MINER_DIAG="http://localhost:7071/_diagnostics"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

random_range() {
    local min=$1
    local max=$2
    echo $((RANDOM % (max - min + 1) + min))
}

random_element() {
    local arr=("$@")
    local len=${#arr[@]}
    echo "${arr[$((RANDOM % len))]}"
}

stop_container() {
    local container=$1
    log "${YELLOW}Stopping container: $container${NC}"
    if docker stop "$container" 2>/dev/null; then
        log "${GREEN}Container $container stopped${NC}"
        return 0
    else
        log "${RED}Failed to stop $container (may already be stopped)${NC}"
        return 1
    fi
}

start_container() {
    local container=$1
    log "${YELLOW}Starting container: $container${NC}"
    if docker start "$container" 2>/dev/null; then
        log "${GREEN}Container $container started${NC}"
        return 0
    else
        log "${RED}Failed to start $container${NC}"
        return 1
    fi
}

is_running() {
    local container=$1
    local status=$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || echo "false")
    [ "$status" = "true" ]
}

count_running_miners() {
    local count=0
    for m in "${MINERS[@]}"; do
        if is_running "$m"; then
            count=$((count + 1))
        fi
    done
    echo $count
}

count_running_sharders() {
    local count=0
    for s in "${SHARDERS[@]}"; do
        if is_running "$s"; then
            count=$((count + 1))
        fi
    done
    echo $count
}

check_status() {
    log "${BLUE}=== Container Status ===${NC}"
    echo -e "  ${CYAN}Miners:${NC}"
    for m in "${MINERS[@]}"; do
        if is_running "$m"; then
            echo -e "    $m: ${GREEN}running${NC}"
        else
            echo -e "    $m: ${RED}stopped${NC}"
        fi
    done
    echo -e "  ${CYAN}Sharders:${NC}"
    for s in "${SHARDERS[@]}"; do
        if is_running "$s"; then
            echo -e "    $s: ${GREEN}running${NC}"
        else
            echo -e "    $s: ${RED}stopped${NC}"
        fi
    done
    echo -e "  Running: ${GREEN}$(count_running_miners)${NC} miners, ${GREEN}$(count_running_sharders)${NC} sharders"
}

check_chain_status() {
    log "${BLUE}=== Chain Status ===${NC}"
    local status=$(curl -s "$MINER_DIAG" 2>/dev/null | grep -oE "(Round</td><td[^>]*><[^>]*>[^<]*</span>[0-9]+|VRF|Latest Finalized Round</td><td[^>]*>[0-9]+)" | head -5 || echo "Unable to fetch")

    # Extract round and LFB from diagnostics
    local round=$(curl -s "$MINER_DIAG" 2>/dev/null | grep -oE "Round</td>.*>([0-9]+)</a>" | grep -oE "[0-9]+" | tail -1 || echo "?")
    local lfb=$(curl -s "$MINER_DIAG" 2>/dev/null | grep -oE "Latest Finalized Round</td><td[^>]*>([0-9]+)" | grep -oE "[0-9]+" || echo "?")
    local vrfs=$(curl -s "$MINER_DIAG" 2>/dev/null | grep -oE "VRFs</td><td[^>]*>\([0-9]+/[0-9]+\)" | grep -oE "\([0-9]+/[0-9]+\)" || echo "?")

    echo -e "  Round: ${GREEN}$round${NC}, LFB: ${GREEN}$lfb${NC}, VRFs: ${GREEN}$vrfs${NC}"
}

# Get a random stopped miner
get_stopped_miner() {
    for m in "${MINERS[@]}"; do
        if ! is_running "$m"; then
            echo "$m"
            return 0
        fi
    done
    echo ""
}

# Get a random running miner (that can be safely stopped)
get_stoppable_miner() {
    local running=$(count_running_miners)
    if [ $running -le $MIN_MINERS_RUNNING ]; then
        echo ""
        return
    fi

    # Build list of running miners and pick random one
    local running_miners=()
    for m in "${MINERS[@]}"; do
        if is_running "$m"; then
            running_miners+=("$m")
        fi
    done

    if [ ${#running_miners[@]} -gt 0 ]; then
        echo "${running_miners[$((RANDOM % ${#running_miners[@]}))]}"
    fi
}

# Get a random running sharder (that can be safely stopped)
get_stoppable_sharder() {
    local running=$(count_running_sharders)
    if [ $running -le $MIN_SHARDERS_RUNNING ]; then
        echo ""
        return
    fi

    local running_sharders=()
    for s in "${SHARDERS[@]}"; do
        if is_running "$s"; then
            running_sharders+=("$s")
        fi
    done

    if [ ${#running_sharders[@]} -gt 0 ]; then
        echo "${running_sharders[$((RANDOM % ${#running_sharders[@]}))]}"
    fi
}

# Get a random stopped sharder
get_stopped_sharder() {
    for s in "${SHARDERS[@]}"; do
        if ! is_running "$s"; then
            echo "$s"
            return 0
        fi
    done
    echo ""
}

# Operation: Stop a random miner (if safe)
op_stop_random_miner() {
    log "${CYAN}Operation: STOP RANDOM MINER${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        stop_container "$miner"
        local stop_time=$MIN_STOP_TIME
        log "${BLUE}Miner $miner will restart in $stop_time seconds${NC}"
        sleep $stop_time
        start_container "$miner"
    else
        log "${YELLOW}Cannot stop more miners - minimum $MIN_MINERS_RUNNING required for consensus${NC}"
    fi
}

# Operation: Stop a random sharder (if safe)
op_stop_random_sharder() {
    log "${CYAN}Operation: STOP RANDOM SHARDER${NC}"
    local sharder=$(get_stoppable_sharder)
    if [ -n "$sharder" ]; then
        stop_container "$sharder"
        local stop_time=$MIN_STOP_TIME
        log "${BLUE}Sharder $sharder will restart in $stop_time seconds${NC}"
        sleep $stop_time
        start_container "$sharder"
    else
        log "${YELLOW}Cannot stop more sharders - minimum $MIN_SHARDERS_RUNNING required${NC}"
    fi
}

# Operation: Stop miner and sharder together
op_stop_miner_and_sharder() {
    log "${CYAN}Operation: STOP ONE MINER AND ONE SHARDER${NC}"
    local miner=$(get_stoppable_miner)
    local sharder=$(get_stoppable_sharder)

    if [ -n "$miner" ]; then
        stop_container "$miner" || true
    fi
    if [ -n "$sharder" ]; then
        stop_container "$sharder" || true
    fi

    if [ -n "$miner" ] || [ -n "$sharder" ]; then
        local stop_time=$MIN_STOP_TIME
        log "${BLUE}Containers down for $stop_time seconds${NC}"
        sleep $stop_time

        [ -n "$sharder" ] && start_container "$sharder"
        sleep 2
        [ -n "$miner" ] && start_container "$miner"
    else
        log "${YELLOW}Cannot stop any more containers safely${NC}"
    fi
}

# Operation: Rolling restart of miners
op_rolling_restart_miners() {
    log "${CYAN}Operation: ROLLING RESTART MINERS${NC}"

    for m in "${MINERS[@]}"; do
        if is_running "$m"; then
            stop_container "$m" || true
            local stop_time=10
            log "${BLUE}$m stopped, waiting $stop_time seconds${NC}"
            sleep $stop_time
            start_container "$m"
            sleep 3
        fi
    done

    log "${GREEN}Rolling restart of miners complete${NC}"
}

# Operation: Restart a stopped container
op_restart_stopped() {
    log "${CYAN}Operation: RESTART STOPPED CONTAINERS${NC}"

    local stopped_miner=$(get_stopped_miner)
    local stopped_sharder=$(get_stopped_sharder)

    if [ -n "$stopped_sharder" ]; then
        start_container "$stopped_sharder"
        sleep 2
    fi
    if [ -n "$stopped_miner" ]; then
        start_container "$stopped_miner"
    fi

    if [ -z "$stopped_miner" ] && [ -z "$stopped_sharder" ]; then
        log "${GREEN}All containers already running${NC}"
    fi
}

# Operation: Quick stop-start of random miner
op_quick_bounce_miner() {
    log "${CYAN}Operation: QUICK BOUNCE MINER${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        stop_container "$miner"
        sleep 3
        start_container "$miner"
    else
        log "${YELLOW}Cannot bounce - minimum miners required${NC}"
    fi
}

# Sequential operations list
OPERATIONS=(
    "op_stop_random_miner"
    "op_stop_random_sharder"
    "op_stop_miner_and_sharder"
    "op_rolling_restart_miners"
    "op_restart_stopped"
    "op_quick_bounce_miner"
)
CURRENT_OP=0

sequential_operation() {
    local op_name=${OPERATIONS[$CURRENT_OP]}

    # Execute the operation
    $op_name

    # Move to next operation, wrap around
    CURRENT_OP=$(( (CURRENT_OP + 1) % ${#OPERATIONS[@]} ))
}

ensure_all_running() {
    log "${YELLOW}Ensuring all containers are running...${NC}"
    for s in "${SHARDERS[@]}"; do
        if ! is_running "$s"; then
            start_container "$s" || true
            sleep 2
        fi
    done
    for m in "${MINERS[@]}"; do
        if ! is_running "$m"; then
            start_container "$m" || true
            sleep 1
        fi
    done
    log "${GREEN}All containers should be running${NC}"
}

cleanup() {
    echo ""
    log "${YELLOW}Cleaning up - starting all containers...${NC}"
    ensure_all_running
    check_status
    check_chain_status
    log "${GREEN}Cleanup complete${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Main
echo ""
log "${GREEN}=== Local Container Chaos Test ===${NC}"
log "Miners: ${MINERS[*]}"
log "Sharders: ${SHARDERS[*]}"
log "Stop time range: ${MIN_STOP_TIME}-${MAX_STOP_TIME} seconds"
log "Run time range: ${MIN_RUN_TIME}-${MAX_RUN_TIME} seconds"
log "Min miners for consensus: $MIN_MINERS_RUNNING"
log "Press Ctrl+C to stop and restore containers"
echo ""

# Ensure all running at start
ensure_all_running
sleep 5
check_status
check_chain_status

iteration=0
while true; do
    iteration=$((iteration + 1))

    if [ $ITERATIONS -gt 0 ] && [ $iteration -gt $ITERATIONS ]; then
        log "${GREEN}Completed $ITERATIONS iterations${NC}"
        cleanup
    fi

    echo ""
    log "${MAGENTA}=== Iteration $iteration ===${NC}"

    sequential_operation

    sleep 3
    check_status
    check_chain_status

    # Fixed sleep before next operation
    run_time=$MIN_RUN_TIME
    log "${BLUE}Next operation in $run_time seconds...${NC}"
    sleep $run_time
done
