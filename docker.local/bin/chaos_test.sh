#!/bin/bash

#############################################################################
# Chaos Testing Script for 0chain Blockchain
#
# This script randomly stops/starts miner and sharder containers to test
# the robustness of the blockchain. It waits indefinitely for the chain
# to progress before continuing with more chaos.
#
# Requirements:
#   - Docker and docker-compose installed
#   - 4 miners and 2 sharders running (configurable)
#   - jq installed for JSON parsing
#
# Usage:
#   ./chaos_test.sh [options]
#
# Options:
#   -d, --duration       Total duration of chaos test in seconds (default: 3600)
#   -i, --interval       Base interval between chaos events in seconds (default: 60)
#   --min-down-time      Minimum time a node stays down in seconds (default: 300)
#   --max-down-time      Maximum time a node stays down in seconds (default: 7200)
#   --dry-run            Show what would happen without executing
#############################################################################

set -e

# Configuration
TOTAL_MINERS=4
TOTAL_SHARDERS=2
MIN_MINERS_FOR_CONSENSUS=3   # Minimum miners needed for consensus
MIN_SHARDERS_FOR_CONSENSUS=1 # Minimum sharders needed for consensus
TEST_DURATION=3600      # 1 hour default
CHAOS_INTERVAL=60       # Base time between chaos events
MIN_DOWN_TIME=300       # 5 minutes minimum down time
MAX_DOWN_TIME=7200      # 120 minutes maximum down time
DRY_RUN=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Tracking arrays for down times
declare -A NODE_DOWN_UNTIL  # When each node should come back up
declare -a EVENT_LOG        # Log of all events for final report

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--duration)
            TEST_DURATION="$2"
            shift 2
            ;;
        -i|--interval)
            CHAOS_INTERVAL="$2"
            shift 2
            ;;
        --min-down-time)
            MIN_DOWN_TIME="$2"
            shift 2
            ;;
        --max-down-time)
            MAX_DOWN_TIME="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            head -24 "$0" | tail -20
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[$(date '+%H:%M:%S')] ✓${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[$(date '+%H:%M:%S')] ⚠${NC} $1"
}

log_error() {
    echo -e "${RED}[$(date '+%H:%M:%S')] ✗${NC} $1"
}

log_chaos() {
    echo -e "${MAGENTA}[$(date '+%H:%M:%S')] 🔥${NC} $1"
}

# Get container name for a miner
get_miner_container() {
    local miner_num=$1
    echo "miner-${miner_num}"
}

# Get container name for a sharder
get_sharder_container() {
    local sharder_num=$1
    echo "sharder-${sharder_num}"
}

# Check if a container is running
is_container_running() {
    local container=$1
    docker ps --format '{{.Names}}' | grep -q "^${container}$"
}

# Get list of running miners
get_running_miners() {
    local running=()
    for i in $(seq 1 $TOTAL_MINERS); do
        if is_container_running "miner-${i}"; then
            running+=($i)
        fi
    done
    echo "${running[@]}"
}

# Get list of running sharders
get_running_sharders() {
    local running=()
    for i in $(seq 1 $TOTAL_SHARDERS); do
        if is_container_running "sharder-${i}"; then
            running+=($i)
        fi
    done
    echo "${running[@]}"
}

# Get list of stopped miners
get_stopped_miners() {
    local stopped=()
    for i in $(seq 1 $TOTAL_MINERS); do
        if ! is_container_running "miner-${i}"; then
            stopped+=($i)
        fi
    done
    echo "${stopped[@]}"
}

# Get list of stopped sharders
get_stopped_sharders() {
    local stopped=()
    for i in $(seq 1 $TOTAL_SHARDERS); do
        if ! is_container_running "sharder-${i}"; then
            stopped+=($i)
        fi
    done
    echo "${stopped[@]}"
}

# Format seconds to human readable
format_duration() {
    local seconds=$1
    local mins=$((seconds / 60))
    local secs=$((seconds % 60))
    if [ $mins -gt 0 ]; then
        echo "${mins}m ${secs}s"
    else
        echo "${secs}s"
    fi
}

# Generate random down time between MIN and MAX
get_random_down_time() {
    local range=$((MAX_DOWN_TIME - MIN_DOWN_TIME))
    local random_offset=$((RANDOM % range))
    echo $((MIN_DOWN_TIME + random_offset))
}

# Stop a miner container
stop_miner() {
    local miner_num=$1
    local down_time=$2
    local container=$(get_miner_container $miner_num)
    
    if $DRY_RUN; then
        log_chaos "[DRY-RUN] Would stop miner-${miner_num} for $(format_duration $down_time)"
        return
    fi
    
    log_chaos "Stopping miner-${miner_num} for $(format_duration $down_time)..."
    docker stop "${container}" >/dev/null 2>&1 || true
    
    # Also stop redis containers for this miner
    docker stop "miner-redis-${miner_num}" >/dev/null 2>&1 || true
    docker stop "miner-redis-txns-${miner_num}" >/dev/null 2>&1 || true
    
    # Record when it should come back up
    NODE_DOWN_UNTIL["miner-${miner_num}"]=$(($(date +%s) + down_time))
}

# Start a miner container
start_miner() {
    local miner_num=$1
    local container=$(get_miner_container $miner_num)
    
    if $DRY_RUN; then
        log_chaos "[DRY-RUN] Would start miner-${miner_num}"
        return
    fi
    
    log_chaos "Starting miner-${miner_num}..."
    
    # Start redis containers first
    docker start "miner-redis-${miner_num}" >/dev/null 2>&1 || true
    docker start "miner-redis-txns-${miner_num}" >/dev/null 2>&1 || true
    sleep 2
    
    # Start miner
    docker start "${container}" >/dev/null 2>&1 || true
    
    # Clear down time
    unset NODE_DOWN_UNTIL["miner-${miner_num}"]
}

# Stop a sharder container
stop_sharder() {
    local sharder_num=$1
    local down_time=$2
    local container=$(get_sharder_container $sharder_num)
    
    if $DRY_RUN; then
        log_chaos "[DRY-RUN] Would stop sharder-${sharder_num} for $(format_duration $down_time)"
        return
    fi
    
    log_chaos "Stopping sharder-${sharder_num} for $(format_duration $down_time)..."
    docker stop "${container}" >/dev/null 2>&1 || true
    
    # Also stop postgres container for this sharder
    docker stop "sharder-postgres-${sharder_num}" >/dev/null 2>&1 || true
    
    # Record when it should come back up
    NODE_DOWN_UNTIL["sharder-${sharder_num}"]=$(($(date +%s) + down_time))
}

# Start a sharder container
start_sharder() {
    local sharder_num=$1
    local container=$(get_sharder_container $sharder_num)
    
    if $DRY_RUN; then
        log_chaos "[DRY-RUN] Would start sharder-${sharder_num}"
        return
    fi
    
    log_chaos "Starting sharder-${sharder_num}..."
    
    # Start postgres first
    docker start "sharder-postgres-${sharder_num}" >/dev/null 2>&1 || true
    sleep 3
    
    # Start sharder
    docker start "${container}" >/dev/null 2>&1 || true
    
    # Clear down time
    unset NODE_DOWN_UNTIL["sharder-${sharder_num}"]
}

# Check and restart nodes whose down time has expired
check_scheduled_restarts() {
    local current_time=$(date +%s)
    
    for node in "${!NODE_DOWN_UNTIL[@]}"; do
        local restart_time=${NODE_DOWN_UNTIL[$node]}
        if [ $current_time -ge $restart_time ]; then
            if [[ $node == miner-* ]]; then
                local num=${node#miner-}
                start_miner $num
            elif [[ $node == sharder-* ]]; then
                local num=${node#sharder-}
                start_sharder $num
            fi
        fi
    done
}

# Get current chain round from a sharder
get_chain_round() {
    local sharder_num=$1
    local port=$((7170 + sharder_num))
    
    # Try to get chain stats
    local response=$(curl -s --connect-timeout 3 --max-time 5 \
        "http://localhost:${port}/v1/chain/get/stats" 2>/dev/null || echo "{}")
    
    # Extract round number using jq or grep
    if command -v jq &> /dev/null; then
        echo "$response" | jq -r '.round // .current_round // 0' 2>/dev/null || echo "0"
    else
        echo "$response" | grep -oP '"round":\s*\K\d+' 2>/dev/null | head -1 || echo "0"
    fi
}

# Get best round from any running sharder or miner
get_best_round() {
    local best_round=0
    
    # Try sharders first
    local running_sharders=($(get_running_sharders))
    for sharder in "${running_sharders[@]}"; do
        local round=$(get_chain_round $sharder)
        if [[ "$round" =~ ^[0-9]+$ ]] && [ "$round" -gt "$best_round" ]; then
            best_round=$round
        fi
    done
    
    # Also try miners if no sharder response
    if [ "$best_round" -eq 0 ]; then
        local running_miners=($(get_running_miners))
        for miner in "${running_miners[@]}"; do
            local port=$((7070 + miner))
            local response=$(curl -s --connect-timeout 3 --max-time 5 \
                "http://localhost:${port}/v1/chain/get/stats" 2>/dev/null || echo "{}")
            local round=0
            if command -v jq &> /dev/null; then
                round=$(echo "$response" | jq -r '.round // .current_round // 0' 2>/dev/null || echo "0")
            fi
            if [[ "$round" =~ ^[0-9]+$ ]] && [ "$round" -gt "$best_round" ]; then
                best_round=$round
            fi
        done
    fi
    
    echo $best_round
}

# Ensure minimum consensus nodes are running
ensure_minimum_consensus() {
    local running_miners=($(get_running_miners))
    local running_sharders=($(get_running_sharders))
    local running_miner_count=${#running_miners[@]}
    local running_sharder_count=${#running_sharders[@]}
    
    local restored_any=false
    
    # Check if we need more miners
    while [ $running_miner_count -lt $MIN_MINERS_FOR_CONSENSUS ]; do
        local stopped_miners=($(get_stopped_miners))
        if [ ${#stopped_miners[@]} -eq 0 ]; then
            break
        fi
        
        # Start the first stopped miner
        local miner_to_start=${stopped_miners[0]}
        log_warn "Restoring miner-${miner_to_start} for consensus (need ${MIN_MINERS_FOR_CONSENSUS} miners, have ${running_miner_count})"
        start_miner $miner_to_start
        restored_any=true
        
        # Update count
        running_miners=($(get_running_miners))
        running_miner_count=${#running_miners[@]}
    done
    
    # Check if we need more sharders
    while [ $running_sharder_count -lt $MIN_SHARDERS_FOR_CONSENSUS ]; do
        local stopped_sharders=($(get_stopped_sharders))
        if [ ${#stopped_sharders[@]} -eq 0 ]; then
            break
        fi
        
        # Start the first stopped sharder
        local sharder_to_start=${stopped_sharders[0]}
        log_warn "Restoring sharder-${sharder_to_start} for consensus (need ${MIN_SHARDERS_FOR_CONSENSUS} sharders, have ${running_sharder_count})"
        start_sharder $sharder_to_start
        restored_any=true
        
        # Update count
        running_sharders=($(get_running_sharders))
        running_sharder_count=${#running_sharders[@]}
    done
    
    if $restored_any; then
        log_info "Waiting 10s for restored nodes to initialize..."
        sleep 10
    fi
}

# Wait indefinitely for chain to progress
wait_for_progress() {
    local initial_round=$1
    local check_interval=10
    local wait_start=$(date +%s)
    
    log_info "Waiting for chain to progress from round ${initial_round}..."
    
    while true; do
        # Check for scheduled restarts
        check_scheduled_restarts
        
        # Ensure minimum consensus before checking progress
        ensure_minimum_consensus
        
        local current_round=$(get_best_round)
        local elapsed=$(($(date +%s) - wait_start))
        
        if [[ "$current_round" =~ ^[0-9]+$ ]] && [ "$current_round" -gt "$initial_round" ]; then
            local progress=$((current_round - initial_round))
            log_success "Chain progressed: ${initial_round} -> ${current_round} (+${progress} rounds) after $(format_duration $elapsed)"
            return 0
        fi
        
        # Show waiting status every minute
        if [ $((elapsed % 60)) -lt $check_interval ]; then
            local running_miners=($(get_running_miners))
            local running_sharders=($(get_running_sharders))
            log_warn "Still waiting... (${#running_miners[@]} miners, ${#running_sharders[@]} sharders up) - $(format_duration $elapsed) elapsed"
        fi
        
        sleep $check_interval
    done
}

# Get current status string for nodes
get_nodes_status_string() {
    local stopped_miners=($(get_stopped_miners))
    local stopped_sharders=($(get_stopped_sharders))
    
    local status=""
    
    if [ ${#stopped_miners[@]} -gt 0 ]; then
        status="M[${stopped_miners[*]}]"
    else
        status="M[none]"
    fi
    
    if [ ${#stopped_sharders[@]} -gt 0 ]; then
        status="${status} S[${stopped_sharders[*]}]"
    else
        status="${status} S[none]"
    fi
    
    echo "$status"
}

# Add event to log
add_event() {
    local event_num=$1
    local action=$2
    local target=$3
    local down_time=$4
    local round_before=$5
    local round_after=$6
    local wait_time=$7
    local nodes_down=$8
    local nodes_after_restore=$9
    
    EVENT_LOG+=("${event_num}|${action}|${target}|${down_time}|${round_before}|${round_after}|${wait_time}|${nodes_down}|${nodes_after_restore}")
}

# Print the results table
print_results_table() {
    echo ""
    echo -e "${CYAN}┌──────┬────────────┬──────────────┬─────────────┬─────────────┬─────────────┬─────────────┬─────────────────────┬─────────────────────┐${NC}"
    echo -e "${CYAN}│${WHITE} Evt# ${CYAN}│${WHITE}   Action   ${CYAN}│${WHITE}    Target    ${CYAN}│${WHITE}  Down Time  ${CYAN}│${WHITE} Round Start ${CYAN}│${WHITE}  Round End  ${CYAN}│${WHITE}  Wait Time  ${CYAN}│${WHITE}   After Action    ${CYAN}│${WHITE}  After Restore    ${CYAN}│${NC}"
    echo -e "${CYAN}├──────┼────────────┼──────────────┼─────────────┼─────────────┼─────────────┼─────────────┼─────────────────────┼─────────────────────┤${NC}"
    
    for event in "${EVENT_LOG[@]}"; do
        IFS='|' read -r num action target down_time round_before round_after wait_time nodes_down nodes_after_restore <<< "$event"
        printf "${CYAN}│${NC} %4s ${CYAN}│${NC} %-10s ${CYAN}│${NC} %-12s ${CYAN}│${NC} %11s ${CYAN}│${NC} %11s ${CYAN}│${NC} %11s ${CYAN}│${NC} %11s ${CYAN}│${NC} %-19s ${CYAN}│${NC} %-19s ${CYAN}│${NC}\n" \
            "$num" "$action" "$target" "$down_time" "$round_before" "$round_after" "$wait_time" "$nodes_down" "$nodes_after_restore"
    done
    
    echo -e "${CYAN}└──────┴────────────┴──────────────┴─────────────┴─────────────┴─────────────┴─────────────┴─────────────────────┴─────────────────────┘${NC}"
}

# Print live status table header
print_live_header() {
    echo ""
    echo -e "${CYAN}┌──────┬────────────┬──────────────┬─────────────┬─────────────┬─────────────┬─────────────┬─────────────────────┬─────────────────────┐${NC}"
    echo -e "${CYAN}│${WHITE} Evt# ${CYAN}│${WHITE}   Action   ${CYAN}│${WHITE}    Target    ${CYAN}│${WHITE}  Down Time  ${CYAN}│${WHITE} Round Start ${CYAN}│${WHITE}  Round End  ${CYAN}│${WHITE}  Wait Time  ${CYAN}│${WHITE}   After Action    ${CYAN}│${WHITE}  After Restore    ${CYAN}│${NC}"
    echo -e "${CYAN}├──────┼────────────┼──────────────┼─────────────┼─────────────┼─────────────┼─────────────┼─────────────────────┼─────────────────────┤${NC}"
}

# Print a single event row
print_event_row() {
    local num=$1
    local action=$2
    local target=$3
    local down_time=$4
    local round_before=$5
    local round_after=$6
    local wait_time=$7
    local nodes_down=$8
    local nodes_after_restore=$9
    
    printf "${CYAN}│${NC} %4s ${CYAN}│${NC} %-10s ${CYAN}│${NC} %-12s ${CYAN}│${NC} %11s ${CYAN}│${NC} %11s ${CYAN}│${NC} %11s ${CYAN}│${NC} %11s ${CYAN}│${NC} %-19s ${CYAN}│${NC} %-19s ${CYAN}│${NC}\n" \
        "$num" "$action" "$target" "$down_time" "$round_before" "$round_after" "$wait_time" "$nodes_down" "$nodes_after_restore"
}

# Cleanup function - restore all containers
cleanup() {
    echo ""
    log_warn "Cleaning up - restoring all containers..."
    
    for i in $(seq 1 $TOTAL_MINERS); do
        if ! is_container_running "miner-${i}"; then
            start_miner $i
        fi
    done
    
    for i in $(seq 1 $TOTAL_SHARDERS); do
        if ! is_container_running "sharder-${i}"; then
            start_sharder $i
        fi
    done
    
    log_success "All containers restored"
    
    # Print final results table
    if [ ${#EVENT_LOG[@]} -gt 0 ]; then
        echo ""
        echo -e "${WHITE}═══════════════════════════════════════════════════════════════════════════════════════════════════════════════════${NC}"
        echo -e "${WHITE}                                              FINAL RESULTS TABLE                                                    ${NC}"
        echo -e "${WHITE}═══════════════════════════════════════════════════════════════════════════════════════════════════════════════════${NC}"
        print_results_table
    fi
}

# Trap cleanup on exit
trap cleanup EXIT

# Random action - stop or start something
perform_chaos_action() {
    local running_miners=($(get_running_miners))
    local running_sharders=($(get_running_sharders))
    local stopped_miners=($(get_stopped_miners))
    local stopped_sharders=($(get_stopped_sharders))
    
    local running_miner_count=${#running_miners[@]}
    local running_sharder_count=${#running_sharders[@]}
    local stopped_miner_count=${#stopped_miners[@]}
    local stopped_sharder_count=${#stopped_sharders[@]}
    
    # Decide what to do randomly
    local action=$((RANDOM % 100))
    local down_time=$(get_random_down_time)
    
    # 40% stop miner, 20% stop sharder, 25% start miner, 15% start sharder
    if [ $action -lt 40 ] && [ $running_miner_count -gt 0 ]; then
        # Stop a random miner
        local idx=$((RANDOM % running_miner_count))
        local miner_to_stop=${running_miners[$idx]}
        stop_miner $miner_to_stop $down_time
        echo "STOP|miner-${miner_to_stop}|$(format_duration $down_time)"
        
    elif [ $action -lt 60 ] && [ $running_sharder_count -gt 0 ]; then
        # Stop a random sharder
        local idx=$((RANDOM % running_sharder_count))
        local sharder_to_stop=${running_sharders[$idx]}
        stop_sharder $sharder_to_stop $down_time
        echo "STOP|sharder-${sharder_to_stop}|$(format_duration $down_time)"
        
    elif [ $action -lt 85 ] && [ $stopped_miner_count -gt 0 ]; then
        # Start a random stopped miner (manual override of scheduled time)
        local idx=$((RANDOM % stopped_miner_count))
        local miner_to_start=${stopped_miners[$idx]}
        start_miner $miner_to_start
        echo "START|miner-${miner_to_start}|-"
        
    elif [ $stopped_sharder_count -gt 0 ]; then
        # Start a random stopped sharder (manual override of scheduled time)
        local idx=$((RANDOM % stopped_sharder_count))
        local sharder_to_start=${stopped_sharders[$idx]}
        start_sharder $sharder_to_start
        echo "START|sharder-${sharder_to_start}|-"
        
    elif [ $stopped_miner_count -gt 0 ]; then
        # Fallback: Start a random stopped miner
        local idx=$((RANDOM % stopped_miner_count))
        local miner_to_start=${stopped_miners[$idx]}
        start_miner $miner_to_start
        echo "START|miner-${miner_to_start}|-"
        
    elif [ $running_miner_count -gt 0 ]; then
        # Fallback: Stop a miner anyway
        local idx=$((RANDOM % running_miner_count))
        local miner_to_stop=${running_miners[$idx]}
        stop_miner $miner_to_stop $down_time
        echo "STOP|miner-${miner_to_stop}|$(format_duration $down_time)"
        
    else
        echo "NONE|-|-"
    fi
}

#############################################################################
# Main execution
#############################################################################

echo ""
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                      0chain Chaos Testing Script                               ║${NC}"
echo -e "${CYAN}╠════════════════════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║${NC} Miners: ${TOTAL_MINERS}              Sharders: ${TOTAL_SHARDERS}                                          ${CYAN}║${NC}"
echo -e "${CYAN}║${NC} Duration: $(format_duration $TEST_DURATION)          Chaos Interval: $(format_duration $CHAOS_INTERVAL)                          ${CYAN}║${NC}"
echo -e "${CYAN}║${NC} Down Time Range: $(format_duration $MIN_DOWN_TIME) - $(format_duration $MAX_DOWN_TIME)                                      ${CYAN}║${NC}"
echo -e "${CYAN}║${NC} Dry Run: ${DRY_RUN}                                                                 ${CYAN}║${NC}"
echo -e "${CYAN}║${NC}                                                                                ${CYAN}║${NC}"
echo -e "${CYAN}║${NC} ${YELLOW}NOTE: Nodes can be stopped below consensus, but will auto-restore${NC}              ${CYAN}║${NC}"
echo -e "${CYAN}║${NC} ${YELLOW}to ${MIN_MINERS_FOR_CONSENSUS} miners + ${MIN_SHARDERS_FOR_CONSENSUS} sharder before waiting for progress.${NC}                       ${CYAN}║${NC}"
echo -e "${CYAN}║${NC} ${YELLOW}Will wait FOREVER for chain progress before continuing.${NC}                       ${CYAN}║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    log_warn "jq not installed - using grep for JSON parsing (less reliable)"
fi

# Initial status check
log_info "Checking initial container status..."
running_miners=($(get_running_miners))
running_sharders=($(get_running_sharders))

log_success "Initial: ${#running_miners[@]} miners [${running_miners[*]}], ${#running_sharders[@]} sharders [${running_sharders[*]}] running"

# Get initial round
initial_round=$(get_best_round)
log_info "Initial chain round: ${initial_round}"

if [ "$initial_round" -eq 0 ]; then
    log_warn "Could not get chain round - chain might not be fully started"
    log_info "Waiting for chain to start..."
    while [ "$initial_round" -eq 0 ]; do
        sleep 5
        initial_round=$(get_best_round)
    done
    log_success "Chain started at round: ${initial_round}"
fi

# Tracking variables
start_time=$(date +%s)
chaos_events=0
last_known_round=$initial_round
highest_round=$initial_round

log_info "Starting chaos test for $(format_duration $TEST_DURATION)..."

# Print live table header
print_live_header

# Main chaos loop
while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    
    if [ $elapsed -ge $TEST_DURATION ]; then
        break
    fi
    
    # Check for scheduled restarts
    check_scheduled_restarts
    
    chaos_events=$((chaos_events + 1))
    
    # Get current round before chaos
    round_before=$(get_best_round)
    
    # Perform a chaos action
    action_result=$(perform_chaos_action)
    IFS='|' read -r action target down_time <<< "$action_result"
    
    if [ "$action" == "NONE" ]; then
        chaos_events=$((chaos_events - 1))
        sleep $CHAOS_INTERVAL
        continue
    fi
    
    # Wait a bit for action to take effect
    sleep 5
    
    # Get nodes down status
    nodes_down=$(get_nodes_status_string)
    
    # Ensure minimum consensus before waiting for progress
    ensure_minimum_consensus
    
    # Wait for chain to progress (indefinitely)
    wait_start=$(date +%s)
    
    while true; do
        # Check for scheduled restarts while waiting
        check_scheduled_restarts
        
        # Re-check consensus in case more nodes went down
        ensure_minimum_consensus
        
        current_round=$(get_best_round)
        
        if [[ "$current_round" =~ ^[0-9]+$ ]] && [ "$current_round" -gt "$last_known_round" ]; then
            break
        fi
        
        # Log status every 30 seconds
        local wait_elapsed=$(($(date +%s) - wait_start))
        if [ $((wait_elapsed % 30)) -lt 10 ] && [ $wait_elapsed -gt 0 ]; then
            local rm=($(get_running_miners))
            local rs=($(get_running_sharders))
            log_info "Waiting for progress... (${#rm[@]}M/${#rs[@]}S up, $(format_duration $wait_elapsed) elapsed)"
        fi
        
        sleep 10
    done
    
    wait_end=$(date +%s)
    wait_time=$((wait_end - wait_start))
    round_after=$current_round
    
    # Update tracking
    last_known_round=$current_round
    if [ "$current_round" -gt "$highest_round" ]; then
        highest_round=$current_round
    fi
    
    # Get nodes down status after restore (for consensus)
    nodes_after_restore=$(get_nodes_status_string)
    
    # Add to event log
    add_event "$chaos_events" "$action" "$target" "$down_time" "$round_before" "$round_after" "$(format_duration $wait_time)" "$nodes_down" "$nodes_after_restore"
    
    # Print row
    print_event_row "$chaos_events" "$action" "$target" "$down_time" "$round_before" "$round_after" "$(format_duration $wait_time)" "$nodes_down" "$nodes_after_restore"
    
    # Sleep before next chaos event
    sleep $CHAOS_INTERVAL
done

# Print table footer
echo -e "${CYAN}└──────┴────────────┴──────────────┴─────────────┴─────────────┴─────────────┴─────────────┴─────────────────────┴─────────────────────┘${NC}"

# Final summary
echo ""
echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                              CHAOS TEST SUMMARY                                ║${NC}"
echo -e "${CYAN}╠════════════════════════════════════════════════════════════════════════════════╣${NC}"
printf "${CYAN}║${NC} %-78s ${CYAN}║${NC}\n" "Duration:              $(format_duration $TEST_DURATION)"
printf "${CYAN}║${NC} %-78s ${CYAN}║${NC}\n" "Chaos Events:          ${chaos_events}"
printf "${CYAN}║${NC} %-78s ${CYAN}║${NC}\n" "Initial Round:         ${initial_round}"
printf "${CYAN}║${NC} %-78s ${CYAN}║${NC}\n" "Final Round:           ${highest_round}"
printf "${CYAN}║${NC} %-78s ${CYAN}║${NC}\n" "Rounds Progressed:     $((highest_round - initial_round))"
echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

log_success "🎉 CHAOS TEST COMPLETED - Chain survived all chaos events!"
exit 0
