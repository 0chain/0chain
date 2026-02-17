#!/bin/bash

# Local Container Chaos Test Script
# Randomly stops and starts miner/sharder containers to test resilience and view change
# For local development environment with 4 miners and 3 sharders

# Do NOT use set -e: expressions like [ -n "$var" ] && cmd return 1 when $var
# is empty, which causes set -e to kill the script unexpectedly.

# Configuration
MIN_STOP_TIME=10    # Minimum time container stays stopped (seconds)
MAX_STOP_TIME=30    # Maximum time container stays stopped (seconds)
MIN_RUN_TIME=20     # Minimum time before next chaos operation (seconds)
MAX_RUN_TIME=60     # Maximum time before next chaos operation (seconds)
ITERATIONS=0        # Number of test iterations (0 = infinite)
VC_PAUSE_TIME=300   # Time to pause after restoring all containers (seconds) - allows view change to execute

# Local container names (matching docker-compose naming)
MINERS=("miner-1" "miner-2" "miner-3" "miner-4")
SHARDERS=("sharder-1" "sharder-2")

# Consensus requirements: T=3 for N=4 miners, need at least 3 miners for consensus
MIN_MINERS_RUNNING=0
MIN_SHARDERS_RUNNING=0

# Diagnostics URL for monitoring
MINER_DIAG="http://localhost:7071/_diagnostics"

# Colors (disabled when not running in a terminal, e.g. nohup to file)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    MAGENTA='\033[0;35m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' CYAN='' MAGENTA='' NC=''
fi

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
    local lfb=$(get_lfb)
    echo -e "  LFB: ${GREEN}$lfb${NC} (highest across all nodes)"
}

# Get current LFB round — checks all miners and sharders, returns the highest
get_lfb() {
    local best=0
    for port in 7071 7072 7073 7074 7171 7172; do
        local r=$(curl -s --connect-timeout 2 "http://localhost:${port}/_diagnostics" 2>/dev/null | grep -oE "Latest Finalized Round</td><td[^>]*>([0-9]+)" | grep -oE "[0-9]+")
        if [ -n "$r" ] && [ "$r" -gt "$best" ] 2>/dev/null; then
            best=$r
        fi
    done
    echo "${best:-0}"
}

# Wait for chain to make progress (LFB to advance)
# Returns 0 if chain progressed, 1 if timeout
wait_for_chain_progress() {
    local min_rounds=${1:-5}         # Minimum rounds to advance
    local timeout=${2:-300}          # Timeout in seconds (default 5 min)
    local check_interval=${3:-10}    # Check interval in seconds

    local start_lfb=$(get_lfb)
    local start_time=$(date +%s)
    local target_lfb=$((start_lfb + min_rounds))

    log "${CYAN}Waiting for chain progress: LFB $start_lfb -> $target_lfb (min +$min_rounds rounds)${NC}"

    while true; do
        local current_lfb=$(get_lfb)
        local elapsed=$(($(date +%s) - start_time))
        local progress=$((current_lfb - start_lfb))

        if [ "$current_lfb" -ge "$target_lfb" ]; then
            log "${GREEN}Chain progressed: LFB $start_lfb -> $current_lfb (+$progress rounds in ${elapsed}s)${NC}"
            return 0
        fi

        if [ $elapsed -ge $timeout ]; then
            log "${RED}TIMEOUT waiting for chain progress: LFB stuck at $current_lfb (started at $start_lfb, +$progress rounds)${NC}"
            return 1
        fi

        # Show progress every check
        echo -n -e "  ${YELLOW}LFB: $current_lfb (+$progress), ${elapsed}s elapsed...${NC}\r"
        sleep $check_interval
    done
}

# Pause for view change transactions with chain progress verification
vc_pause() {
    log "${MAGENTA}=== Waiting for View Change Window (20 seconds) ===${NC}"

    # Ensure all containers are running before waiting
    ensure_all_running

    # Wait minimum 20 seconds for view change to complete
    local min_wait=20
    local start_time=$(date +%s)
    local start_lfb=$(get_lfb)

    log "${CYAN}Waiting ${min_wait}s for view change to complete (LFB: $start_lfb)${NC}"

    while true; do
        local elapsed=$(($(date +%s) - start_time))
        local current_lfb=$(get_lfb)
        local progress=$((current_lfb - start_lfb))

        if [ $elapsed -ge $min_wait ]; then
            log "${GREEN}View change window complete: ${elapsed}s elapsed, LFB $start_lfb -> $current_lfb (+$progress rounds)${NC}"
            break
        fi

        # Show progress every 15 seconds
        echo -n -e "  ${YELLOW}LFB: $current_lfb (+$progress), ${elapsed}s/${min_wait}s elapsed...${NC}\r"
        sleep 15
    done

    # Verify chain is still progressing
    if ! wait_for_chain_progress 5 60 10; then
        log "${RED}WARNING: Chain did not progress after VC window!${NC}"
        log "${YELLOW}Chain may be stuck - check miner logs for errors${NC}"
        check_status
        check_chain_status
        return 1
    fi

    log "${GREEN}=== Chain Progress Verified ===${NC}"
    check_status
    check_chain_status
    return 0
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

# Operation: Quick stop-start of random sharder
op_quick_bounce_sharder() {
    log "${CYAN}Operation: QUICK BOUNCE SHARDER${NC}"
    local sharder=$(get_stoppable_sharder)
    if [ -n "$sharder" ]; then
        stop_container "$sharder"
        sleep 3
        start_container "$sharder"
    else
        log "${YELLOW}Cannot bounce - minimum sharders required${NC}"
    fi
}

# Operation: Stop 2 miners simultaneously
op_stop_two_miners() {
    log "${CYAN}Operation: STOP TWO MINERS${NC}"
    local running=$(count_running_miners)
    if [ $running -le $((MIN_MINERS_RUNNING + 1)) ]; then
        log "${YELLOW}Cannot stop 2 miners - need at least $((MIN_MINERS_RUNNING + 2)) running${NC}"
        return
    fi

    local stopped=()
    for m in "${MINERS[@]}"; do
        if [ ${#stopped[@]} -ge 2 ]; then
            break
        fi
        if is_running "$m"; then
            stop_container "$m" || true
            stopped+=("$m")
        fi
    done

    if [ ${#stopped[@]} -gt 0 ]; then
        local stop_time=$MIN_STOP_TIME
        log "${BLUE}${#stopped[@]} miners down for $stop_time seconds${NC}"
        sleep $stop_time
        for m in "${stopped[@]}"; do
            start_container "$m"
            sleep 2
        done
    fi
}

# Operation: Rolling restart of sharders
op_rolling_restart_sharders() {
    log "${CYAN}Operation: ROLLING RESTART SHARDERS${NC}"

    for s in "${SHARDERS[@]}"; do
        if is_running "$s"; then
            stop_container "$s" || true
            local stop_time=10
            log "${BLUE}$s stopped, waiting $stop_time seconds${NC}"
            sleep $stop_time
            start_container "$s"
            sleep 5
        fi
    done

    log "${GREEN}Rolling restart of sharders complete${NC}"
}

# Operation: Long stop miner (60 seconds)
op_long_stop_miner() {
    log "${CYAN}Operation: LONG STOP MINER (60s)${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        stop_container "$miner"
        local stop_time=60
        log "${BLUE}Miner $miner will restart in $stop_time seconds${NC}"
        sleep $stop_time
        start_container "$miner"
    else
        log "${YELLOW}Cannot stop - minimum miners required${NC}"
    fi
}

# Operation: Long stop sharder (60 seconds)
op_long_stop_sharder() {
    log "${CYAN}Operation: LONG STOP SHARDER (60s)${NC}"
    local sharder=$(get_stoppable_sharder)
    if [ -n "$sharder" ]; then
        stop_container "$sharder"
        local stop_time=60
        log "${BLUE}Sharder $sharder will restart in $stop_time seconds${NC}"
        sleep $stop_time
        start_container "$sharder"
    else
        log "${YELLOW}Cannot stop - minimum sharders required${NC}"
    fi
}

# Operation: Very long stop miner (120 seconds)
op_very_long_stop_miner() {
    log "${CYAN}Operation: VERY LONG STOP MINER (120s)${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        stop_container "$miner"
        local stop_time=120
        log "${BLUE}Miner $miner will restart in $stop_time seconds${NC}"
        sleep $stop_time
        start_container "$miner"
    else
        log "${YELLOW}Cannot stop - minimum miners required${NC}"
    fi
}

# Operation: Staggered miner stops (stop one, wait, stop another)
op_staggered_miner_stops() {
    log "${CYAN}Operation: STAGGERED MINER STOPS${NC}"
    local running=$(count_running_miners)
    if [ $running -le $((MIN_MINERS_RUNNING + 1)) ]; then
        log "${YELLOW}Cannot do staggered stops - need more miners running${NC}"
        return
    fi

    local first_miner=$(get_stoppable_miner)
    if [ -n "$first_miner" ]; then
        stop_container "$first_miner"
        log "${BLUE}First miner $first_miner stopped, waiting 15s before stopping another${NC}"
        sleep 15

        local second_miner=$(get_stoppable_miner)
        if [ -n "$second_miner" ]; then
            stop_container "$second_miner"
            log "${BLUE}Second miner $second_miner stopped, waiting 20s${NC}"
            sleep 20
            start_container "$second_miner"
            sleep 2
        fi
        start_container "$first_miner"
    fi
}

# Operation: Stop all stoppable miners (down to minimum)
op_stop_all_stoppable_miners() {
    log "${CYAN}Operation: STOP ALL STOPPABLE MINERS${NC}"
    local stopped=()

    while true; do
        local miner=$(get_stoppable_miner)
        if [ -z "$miner" ]; then
            break
        fi
        stop_container "$miner" || true
        stopped+=("$miner")
        sleep 1
    done

    if [ ${#stopped[@]} -gt 0 ]; then
        log "${BLUE}Stopped ${#stopped[@]} miners, waiting $MIN_STOP_TIME seconds${NC}"
        sleep $MIN_STOP_TIME
        for m in "${stopped[@]}"; do
            start_container "$m"
            sleep 2
        done
    else
        log "${YELLOW}No miners could be stopped${NC}"
    fi
}

# Operation: Stop all stoppable sharders (down to minimum)
op_stop_all_stoppable_sharders() {
    log "${CYAN}Operation: STOP ALL STOPPABLE SHARDERS${NC}"
    local stopped=()

    while true; do
        local sharder=$(get_stoppable_sharder)
        if [ -z "$sharder" ]; then
            break
        fi
        stop_container "$sharder" || true
        stopped+=("$sharder")
        sleep 1
    done

    if [ ${#stopped[@]} -gt 0 ]; then
        log "${BLUE}Stopped ${#stopped[@]} sharders, waiting $MIN_STOP_TIME seconds${NC}"
        sleep $MIN_STOP_TIME
        for s in "${stopped[@]}"; do
            start_container "$s"
            sleep 2
        done
    else
        log "${YELLOW}No sharders could be stopped${NC}"
    fi
}

# Operation: Kill (force stop) random miner
op_kill_random_miner() {
    log "${CYAN}Operation: KILL RANDOM MINER (force stop)${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        log "${RED}Force killing $miner${NC}"
        docker kill "$miner" 2>/dev/null || true
        local stop_time=$MIN_STOP_TIME
        log "${BLUE}Miner $miner will restart in $stop_time seconds${NC}"
        sleep $stop_time
        start_container "$miner"
    else
        log "${YELLOW}Cannot kill - minimum miners required${NC}"
    fi
}

# Operation: Kill (force stop) random sharder
op_kill_random_sharder() {
    log "${CYAN}Operation: KILL RANDOM SHARDER (force stop)${NC}"
    local sharder=$(get_stoppable_sharder)
    if [ -n "$sharder" ]; then
        log "${RED}Force killing $sharder${NC}"
        docker kill "$sharder" 2>/dev/null || true
        local stop_time=$MIN_STOP_TIME
        log "${BLUE}Sharder $sharder will restart in $stop_time seconds${NC}"
        sleep $stop_time
        start_container "$sharder"
    else
        log "${YELLOW}Cannot kill - minimum sharders required${NC}"
    fi
}

# Operation: Stop ALL miners (chain will halt!)
op_stop_all_miners() {
    log "${RED}Operation: STOP ALL MINERS (chain will halt!)${NC}"
    local stopped=()

    for m in "${MINERS[@]}"; do
        if is_running "$m"; then
            stop_container "$m" || true
            stopped+=("$m")
        fi
    done

    if [ ${#stopped[@]} -gt 0 ]; then
        local stop_time=$MIN_STOP_TIME
        log "${RED}ALL ${#stopped[@]} miners stopped! Chain halted. Waiting $stop_time seconds${NC}"
        sleep $stop_time
        # Start sharders first, then miners
        for m in "${stopped[@]}"; do
            start_container "$m"
            sleep 2
        done
        log "${GREEN}All miners restarted${NC}"
    fi
}

# Operation: Stop ALL sharders
op_stop_all_sharders() {
    log "${RED}Operation: STOP ALL SHARDERS${NC}"
    local stopped=()

    for s in "${SHARDERS[@]}"; do
        if is_running "$s"; then
            stop_container "$s" || true
            stopped+=("$s")
        fi
    done

    if [ ${#stopped[@]} -gt 0 ]; then
        local stop_time=$MIN_STOP_TIME
        log "${RED}ALL ${#stopped[@]} sharders stopped! Waiting $stop_time seconds${NC}"
        sleep $stop_time
        for s in "${stopped[@]}"; do
            start_container "$s"
            sleep 3
        done
        log "${GREEN}All sharders restarted${NC}"
    fi
}

# Operation: Long stop miner and sharder together (60s)
op_long_stop_miner_and_sharder() {
    log "${CYAN}Operation: LONG STOP MINER AND SHARDER (60s)${NC}"
    local miner=$(get_stoppable_miner)
    local sharder=$(get_stoppable_sharder)

    if [ -n "$miner" ]; then
        stop_container "$miner" || true
    fi
    if [ -n "$sharder" ]; then
        stop_container "$sharder" || true
    fi

    if [ -n "$miner" ] || [ -n "$sharder" ]; then
        local stop_time=60
        log "${BLUE}Containers down for $stop_time seconds${NC}"
        sleep $stop_time

        [ -n "$sharder" ] && start_container "$sharder"
        sleep 2
        [ -n "$miner" ] && start_container "$miner"
    else
        log "${YELLOW}Cannot stop any containers${NC}"
    fi
}

# Operation: Rapid bounce miner (1 second stop)
op_rapid_bounce_miner() {
    log "${CYAN}Operation: RAPID BOUNCE MINER (1s)${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        stop_container "$miner"
        sleep 1
        start_container "$miner"
    else
        log "${YELLOW}Cannot bounce - minimum miners required${NC}"
    fi
}

# Operation: Rapid bounce sharder (1 second stop)
op_rapid_bounce_sharder() {
    log "${CYAN}Operation: RAPID BOUNCE SHARDER (1s)${NC}"
    local sharder=$(get_stoppable_sharder)
    if [ -n "$sharder" ]; then
        stop_container "$sharder"
        sleep 1
        start_container "$sharder"
    else
        log "${YELLOW}Cannot bounce - minimum sharders required${NC}"
    fi
}

# Operation: Extended stop miner (3 minutes — misses multiple VC cycles)
op_extended_stop_miner() {
    log "${RED}Operation: EXTENDED STOP MINER (180s)${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        stop_container "$miner"
        local stop_time=180
        log "${RED}Miner $miner will be DOWN for $stop_time seconds (3 min)${NC}"
        sleep $stop_time
        start_container "$miner"
    else
        log "${YELLOW}Cannot stop - minimum miners required${NC}"
    fi
}

# Operation: Extended stop sharder (3 minutes)
op_extended_stop_sharder() {
    log "${RED}Operation: EXTENDED STOP SHARDER (180s)${NC}"
    local sharder=$(get_stoppable_sharder)
    if [ -n "$sharder" ]; then
        stop_container "$sharder"
        local stop_time=180
        log "${RED}Sharder $sharder will be DOWN for $stop_time seconds (3 min)${NC}"
        sleep $stop_time
        start_container "$sharder"
    else
        log "${YELLOW}Cannot stop - minimum sharders required${NC}"
    fi
}

# Operation: Extended stop miner AND sharder (3 minutes)
op_extended_stop_miner_and_sharder() {
    log "${RED}Operation: EXTENDED STOP MINER AND SHARDER (180s)${NC}"
    local miner=$(get_stoppable_miner)
    local sharder=$(get_stoppable_sharder)

    if [ -n "$miner" ]; then
        stop_container "$miner" || true
    fi
    if [ -n "$sharder" ]; then
        stop_container "$sharder" || true
    fi

    if [ -n "$miner" ] || [ -n "$sharder" ]; then
        local stop_time=180
        log "${RED}Containers down for $stop_time seconds (3 min)${NC}"
        sleep $stop_time

        [ -n "$sharder" ] && start_container "$sharder"
        sleep 2
        [ -n "$miner" ] && start_container "$miner"
    else
        log "${YELLOW}Cannot stop any containers${NC}"
    fi
}

# Operation: Very extended stop miner (5 minutes — misses many VC cycles)
op_very_extended_stop_miner() {
    log "${RED}Operation: VERY EXTENDED STOP MINER (300s)${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        stop_container "$miner"
        local stop_time=300
        log "${RED}Miner $miner will be DOWN for $stop_time seconds (5 min)${NC}"
        sleep $stop_time
        start_container "$miner"
    else
        log "${YELLOW}Cannot stop - minimum miners required${NC}"
    fi
}

# Operation: Kill miner and leave down for 3 minutes (simulates hard crash + slow recovery)
op_kill_extended_miner() {
    log "${RED}Operation: KILL + EXTENDED DOWN MINER (180s)${NC}"
    local miner=$(get_stoppable_miner)
    if [ -n "$miner" ]; then
        log "${RED}Force killing $miner${NC}"
        docker kill "$miner" 2>/dev/null || true
        local stop_time=180
        log "${RED}Miner $miner killed, will restart in $stop_time seconds (3 min)${NC}"
        sleep $stop_time
        start_container "$miner"
    else
        log "${YELLOW}Cannot kill - minimum miners required${NC}"
    fi
}

# --- Multi-node long-stop combo operations ---
# These intentionally go below MIN_MINERS_RUNNING to test recovery.
# Chain WILL halt during these but must recover after restart.

# Helper: stop N random running miners, returns names in STOPPED_MINERS array
stop_n_miners() {
    local n=$1
    STOPPED_MINERS=()
    for m in "${MINERS[@]}"; do
        if [ ${#STOPPED_MINERS[@]} -ge $n ]; then break; fi
        if is_running "$m"; then
            stop_container "$m" || true
            STOPPED_MINERS+=("$m")
        fi
    done
}

# Helper: stop N random running sharders, returns names in STOPPED_SHARDERS array
stop_n_sharders() {
    local n=$1
    STOPPED_SHARDERS=()
    for s in "${SHARDERS[@]}"; do
        if [ ${#STOPPED_SHARDERS[@]} -ge $n ]; then break; fi
        if is_running "$s"; then
            stop_container "$s" || true
            STOPPED_SHARDERS+=("$s")
        fi
    done
}

# Helper: restart stopped miners and sharders (sharders first)
restart_stopped_combo() {
    for s in "${STOPPED_SHARDERS[@]}"; do
        start_container "$s"
        sleep 2
    done
    for m in "${STOPPED_MINERS[@]}"; do
        start_container "$m"
        sleep 2
    done
}

# --- 2 miners down ---

op_combo_2miners_2m() {
    log "${RED}Operation: 2 MINERS DOWN (120s) — chain may lose consensus${NC}"
    stop_n_miners 2
    if [ ${#STOPPED_MINERS[@]} -ge 2 ]; then
        log "${RED}${STOPPED_MINERS[*]} down for 120s (2 min)${NC}"
        sleep 120
    fi
    restart_stopped_combo
}

op_combo_2miners_3m() {
    log "${RED}Operation: 2 MINERS DOWN (180s) — chain may lose consensus${NC}"
    stop_n_miners 2
    if [ ${#STOPPED_MINERS[@]} -ge 2 ]; then
        log "${RED}${STOPPED_MINERS[*]} down for 180s (3 min)${NC}"
        sleep 180
    fi
    restart_stopped_combo
}

op_combo_2miners_4m() {
    log "${RED}Operation: 2 MINERS DOWN (240s) — chain may lose consensus${NC}"
    stop_n_miners 2
    if [ ${#STOPPED_MINERS[@]} -ge 2 ]; then
        log "${RED}${STOPPED_MINERS[*]} down for 240s (4 min)${NC}"
        sleep 240
    fi
    restart_stopped_combo
}

# --- 2 miners + 1 sharder down ---

op_combo_2miners_1sharder_2m() {
    log "${RED}Operation: 2 MINERS + 1 SHARDER DOWN (120s)${NC}"
    stop_n_miners 2
    stop_n_sharders 1
    log "${RED}Miners: ${STOPPED_MINERS[*]}, Sharders: ${STOPPED_SHARDERS[*]} — down for 120s (2 min)${NC}"
    sleep 120
    restart_stopped_combo
}

op_combo_2miners_1sharder_3m() {
    log "${RED}Operation: 2 MINERS + 1 SHARDER DOWN (180s)${NC}"
    stop_n_miners 2
    stop_n_sharders 1
    log "${RED}Miners: ${STOPPED_MINERS[*]}, Sharders: ${STOPPED_SHARDERS[*]} — down for 180s (3 min)${NC}"
    sleep 180
    restart_stopped_combo
}

op_combo_2miners_1sharder_4m() {
    log "${RED}Operation: 2 MINERS + 1 SHARDER DOWN (240s)${NC}"
    stop_n_miners 2
    stop_n_sharders 1
    log "${RED}Miners: ${STOPPED_MINERS[*]}, Sharders: ${STOPPED_SHARDERS[*]} — down for 240s (4 min)${NC}"
    sleep 240
    restart_stopped_combo
}

# --- 2 sharders down (all sharders) ---

op_combo_2sharders_2m() {
    log "${RED}Operation: 2 SHARDERS DOWN (120s) — no sharders available${NC}"
    stop_n_sharders 2
    if [ ${#STOPPED_SHARDERS[@]} -ge 2 ]; then
        log "${RED}${STOPPED_SHARDERS[*]} down for 120s (2 min)${NC}"
        sleep 120
    fi
    restart_stopped_combo
}

op_combo_2sharders_3m() {
    log "${RED}Operation: 2 SHARDERS DOWN (180s) — no sharders available${NC}"
    stop_n_sharders 2
    if [ ${#STOPPED_SHARDERS[@]} -ge 2 ]; then
        log "${RED}${STOPPED_SHARDERS[*]} down for 180s (3 min)${NC}"
        sleep 180
    fi
    restart_stopped_combo
}

op_combo_2sharders_4m() {
    log "${RED}Operation: 2 SHARDERS DOWN (240s) — no sharders available${NC}"
    stop_n_sharders 2
    if [ ${#STOPPED_SHARDERS[@]} -ge 2 ]; then
        log "${RED}${STOPPED_SHARDERS[*]} down for 240s (4 min)${NC}"
        sleep 240
    fi
    restart_stopped_combo
}

# --- 2 sharders + 1 miner down ---

op_combo_2sharders_1miner_2m() {
    log "${RED}Operation: 2 SHARDERS + 1 MINER DOWN (120s)${NC}"
    stop_n_sharders 2
    stop_n_miners 1
    log "${RED}Sharders: ${STOPPED_SHARDERS[*]}, Miners: ${STOPPED_MINERS[*]} — down for 120s (2 min)${NC}"
    sleep 120
    restart_stopped_combo
}

op_combo_2sharders_1miner_3m() {
    log "${RED}Operation: 2 SHARDERS + 1 MINER DOWN (180s)${NC}"
    stop_n_sharders 2
    stop_n_miners 1
    log "${RED}Sharders: ${STOPPED_SHARDERS[*]}, Miners: ${STOPPED_MINERS[*]} — down for 180s (3 min)${NC}"
    sleep 180
    restart_stopped_combo
}

op_combo_2sharders_1miner_4m() {
    log "${RED}Operation: 2 SHARDERS + 1 MINER DOWN (240s)${NC}"
    stop_n_sharders 2
    stop_n_miners 1
    log "${RED}Sharders: ${STOPPED_SHARDERS[*]}, Miners: ${STOPPED_MINERS[*]} — down for 240s (4 min)${NC}"
    sleep 240
    restart_stopped_combo
}

# Sequential operations list — multi-node long-stop combos placed EARLY
# to stress-test DKG/MB recovery after missing multiple VC cycles
OPERATIONS=(
    # --- Phase 1: Multi-node long-stop combos (2+ min) ---
    "op_stop_random_miner"              #  1: warm-up — quick stop
    "op_combo_2miners_2m"               #  2: 2 miners down 2 min
    "op_combo_2sharders_2m"             #  3: 2 sharders down 2 min
    "op_combo_2miners_1sharder_2m"      #  4: 2 miners + 1 sharder down 2 min
    "op_combo_2sharders_1miner_2m"      #  5: 2 sharders + 1 miner down 2 min
    "op_combo_2miners_3m"               #  6: 2 miners down 3 min
    "op_combo_2sharders_3m"             #  7: 2 sharders down 3 min
    "op_combo_2miners_1sharder_3m"      #  8: 2 miners + 1 sharder down 3 min
    "op_combo_2sharders_1miner_3m"      #  9: 2 sharders + 1 miner down 3 min
    "op_combo_2miners_4m"               # 10: 2 miners down 4 min
    "op_combo_2miners_1sharder_4m"      # 11: 2 miners + 1 sharder down 4 min
    "op_combo_2sharders_4m"             # 12: 2 sharders down 4 min
    "op_combo_2sharders_1miner_4m"      # 13: 2 sharders + 1 miner down 4 min
    # --- Phase 2: Single-node extended stops ---
    "op_extended_stop_miner"            # 14: 3 min miner down
    "op_extended_stop_sharder"          # 15: 3 min sharder down
    "op_extended_stop_miner_and_sharder" # 16: 3 min miner+sharder down
    "op_very_extended_stop_miner"       # 17: 5 min miner down
    "op_kill_extended_miner"            # 18: hard kill + 3 min down
    # --- Phase 3: Medium-length stops ---
    "op_very_long_stop_miner"           # 19: 120s miner
    "op_long_stop_miner_and_sharder"    # 20: 60s miner+sharder
    "op_long_stop_miner"               # 21: 60s miner
    "op_long_stop_sharder"             # 22: 60s sharder
    # --- Phase 4: Short stops and misc ---
    "op_stop_two_miners"                # 23: two miners at once (short)
    "op_staggered_miner_stops"          # 24: staggered stops
    "op_rolling_restart_miners"         # 25: rolling restart all miners
    "op_rolling_restart_sharders"       # 26: rolling restart sharders
    "op_stop_miner_and_sharder"         # 27: quick miner+sharder
    "op_stop_random_sharder"            # 28: quick sharder stop
    "op_stop_all_stoppable_miners"     # 29: max miners down
    "op_stop_all_stoppable_sharders"   # 30: max sharders down
    "op_kill_random_miner"             # 31: force kill miner
    "op_kill_random_sharder"           # 32: force kill sharder
    "op_stop_all_miners"               # 33: all miners down (chain halts)
    "op_stop_all_sharders"             # 34: all sharders down
    "op_quick_bounce_miner"            # 35: quick bounce
    "op_quick_bounce_sharder"          # 36: quick bounce
    "op_rapid_bounce_miner"            # 37: 1s bounce
    "op_rapid_bounce_sharder"          # 38: 1s bounce
    "op_restart_stopped"               # 39: restart any stopped
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
log "VC pause time: ${VC_PAUSE_TIME} seconds (5 minutes)"
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

    # Wait for chain to progress before next operation
    if ! vc_pause; then
        log "${RED}=== CHAIN STUCK - PAUSING CHAOS OPERATIONS ===${NC}"
        log "${YELLOW}Chain did not progress after view change window.${NC}"
        log "${YELLOW}Ensuring all containers are running and waiting for manual recovery...${NC}"
        ensure_all_running
        check_status
        check_chain_status

        # Pause and wait - don't do more chaos operations while chain is stuck
        log "${MAGENTA}Chaos operations PAUSED. Monitoring chain every 60s...${NC}"
        log "${MAGENTA}Press Ctrl+C to stop, or wait for chain to recover naturally.${NC}"

        pause_start=$(date +%s)
        while true; do
            sleep 60
            pause_duration=$(( ($(date +%s) - pause_start) / 60 ))

            # Check if chain is progressing
            if wait_for_chain_progress 5 120 10; then
                log "${GREEN}=== CHAIN RECOVERED after ${pause_duration} minutes ===${NC}"
                check_status
                check_chain_status
                log "${GREEN}Resuming chaos operations...${NC}"
                break
            fi

            log "${YELLOW}[Paused ${pause_duration}m] Chain still stuck - waiting... (Ctrl+C to stop)${NC}"
            check_chain_status
        done
    fi

    # Fixed 1-minute gap between iterations to let chain stabilize
    run_time=60
    log "${BLUE}Next operation in $run_time seconds (1 min)...${NC}"
    sleep $run_time
done
