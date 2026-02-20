#!/bin/bash

# View Change Loop Script
# Continuously tests miner/sharder deletion and addition with verification

# Miner and sharder pools for random selection each iteration
MINER_IDS=(
    "31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929"
    "585732eb076d07455fbebcf3388856b6fd00449a25c47c0f72d961c7c4e7e7c2"
    "8877e3da19b4cb51e59b4646ec7c0cf4849bc7b860257d69ddbf753b9a981e1b"
    "bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d8"
)
SHARDER_IDS=(
    "57b416fcda1cf82b8a7e1fc3a47c68a94e617be873b5383ea2606bda757d3ce4"
    "b098d2d56b087ee910f3ee2d2df173630566babb69f0be0e2e9a0c98d63f0b0b"
)
# Selected randomly at the start of each iteration
MINER_ID=""
SHARDER_ID=""
ZWALLET_DIR="/Users/saswatabasu/Code/zwalletcli"
ZWALLET_PATH="$ZWALLET_DIR/zwallet"
WALLET="owner_wallet.json"
CONFIG="local.yaml"
SLEEP_TIME=30
VC_WAIT_TIME=120  # Max time to wait for view change
CHAIN_PROGRESS_CHECKS=5  # Number of progress checks before moving to next test
CHAIN_PROGRESS_INTERVAL=3  # Seconds between progress checks
VC_CYCLES_TO_WAIT=2  # Number of view change cycles to wait for add/delete to take effect

# Nonce management
CURRENT_NONCE=0
NONCE_INITIALIZED=false

# Log file
LOG_FILE="/tmp/view_change_loop.log"

# Diagnostics URLs
MINER_DIAG="http://localhost:7071/_diagnostics"
MINER4_DIAG="http://localhost:7074/_diagnostics"
SHARDER_DIAG="http://localhost:7171/_diagnostics"
DNS_NETWORK="http://localhost:9091/network"
DNS_MAGIC_BLOCK="http://localhost:9091/magic_block"

# Miner SC state endpoints (from sharder2)
SHARDER2_REST="http://localhost:7172"
MINER_SC="6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
DELETE_MINERS_KEY="${MINER_SC}7eedce10df401053add4456541c5aadcc7644c150078435810fa5990a20fae2e"
DELETE_SHARDERS_KEY="${MINER_SC}ec6aa81faf0ce0bbcdfbd76014433c9d12ae10552c64ae3b70dc3bcda8a204e7"
REGISTER_MINERS_KEY="${MINER_SC}5e6c409eac3d9b2ec0fc0294926c00d91323e9348d30f9a6a0c340272f2c03a4"
REGISTER_SHARDERS_KEY="${MINER_SC}67e5fb97a908fdd92ffe340cb3e39c25194d4c633e7401908f175bea5b76314a"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

ITERATION=0
FAILURES=0
PREV_ROUND=0
PREV_MB=0

# Results tracking
declare -a TEST_RESULTS
declare -a TEST_NAMES

# Get client ID from wallet
get_client_id() {
    cat "$ZWALLET_DIR/$WALLET" 2>/dev/null | python3 -c "import json,sys; print(json.load(sys.stdin).get('client_id',''))" 2>/dev/null
}

# Get nonce from sharder API
get_nonce_from_sharder() {
    local sharder_url=$1
    local client_id=$(get_client_id)
    if [ -z "$client_id" ]; then
        echo "0"
        return
    fi
    local response=$(curl -s "${sharder_url}/v1/client/get?id=${client_id}" 2>/dev/null)
    local nonce=$(echo "$response" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('nonce', 0))" 2>/dev/null)
    echo "${nonce:-0}"
}

# Get highest nonce from multiple sharders
get_current_nonce() {
    local nonce1=$(get_nonce_from_sharder "http://localhost:7171")
    local nonce2=$(get_nonce_from_sharder "http://localhost:7172")

    # Return the higher nonce
    if [ "$nonce1" -gt "$nonce2" ] 2>/dev/null; then
        echo "$nonce1"
    else
        echo "$nonce2"
    fi
}

# Initialize nonce by querying sharders
initialize_nonce() {
    if [ "$NONCE_INITIALIZED" = true ]; then
        return
    fi

    echo -e "  ${CYAN}Initializing nonce...${NC}"
    local sharder_nonce=$(get_current_nonce)

    if [ "$sharder_nonce" -gt 0 ] 2>/dev/null; then
        CURRENT_NONCE=$sharder_nonce
        echo -e "  ${GREEN}Got nonce from sharder: $CURRENT_NONCE${NC}"
    else
        CURRENT_NONCE=0
        echo -e "  ${YELLOW}Starting with nonce: $CURRENT_NONCE${NC}"
    fi
    NONCE_INITIALIZED=true
}

# Advance nonce: refresh from sharder first, then increment
# Updates CURRENT_NONCE directly — do NOT call via $() subshell
advance_nonce() {
    local sharder_nonce=$(get_current_nonce)
    if [ "$sharder_nonce" -gt "$CURRENT_NONCE" ] 2>/dev/null; then
        CURRENT_NONCE=$sharder_nonce
    fi
    CURRENT_NONCE=$((CURRENT_NONCE + 1))
}

# Execute zwallet command with nonce management and retry
run_zwallet_cmd() {
    local cmd=$1
    local max_retries=${2:-5}
    local retry=0
    local backoff=3

    while [ $retry -lt $max_retries ]; do
        advance_nonce
        local nonce=$CURRENT_NONCE
        local full_cmd="$cmd --withNonce $nonce"

        echo -e "    ${CYAN}> $cmd (nonce=$nonce)${NC}"
        local output=$(eval "$full_cmd" 2>&1)
        local exit_code=$?

        # Log to file
        echo "Command: $full_cmd" >> "$LOG_FILE"
        echo "Output: $output" >> "$LOG_FILE"

        # Check for success
        if echo "$output" | grep -qiE "success|Execute faucet|confirmed"; then
            echo -e "    ${GREEN}SUCCESS${NC}"
            echo "$output" | grep -E "Hash|transaction" || true
            return 0
        fi

        # Check for "already" conditions
        if echo "$output" | grep -qiE "already"; then
            echo -e "    ${YELLOW}Already in desired state${NC}"
            return 0
        fi

        # Check for insufficient balance - fund and retry
        if echo "$output" | grep -qiE "insufficient balance"; then
            retry=$((retry + 1))
            echo -e "    ${YELLOW}Insufficient balance, funding wallet with 100 tokens (retry $retry/$max_retries)...${NC}"
            advance_nonce
            cd "$ZWALLET_DIR" && $ZWALLET_PATH faucet --methodName pour --input "{}" --tokens 100 --wallet $WALLET --config $CONFIG --withNonce $CURRENT_NONCE 2>&1 | grep -E "success|error" || true
            sleep 3
            continue
        fi

        # Check for nonce errors — re-sync from sharder instead of guessing
        if echo "$output" | grep -qiE "nonce"; then
            retry=$((retry + 1))
            local fresh_nonce=$(get_current_nonce)
            echo -e "    ${YELLOW}Nonce error, refreshing from sharder: $CURRENT_NONCE -> $fresh_nonce (retry $retry/$max_retries)${NC}"
            if [ "$fresh_nonce" -gt "$CURRENT_NONCE" ] 2>/dev/null; then
                CURRENT_NONCE=$fresh_nonce
            else
                # Sharder nonce didn't help — nudge forward by 1
                CURRENT_NONCE=$((CURRENT_NONCE + 1))
            fi
            sleep 2
            continue
        fi

        # Check for network errors
        if echo "$output" | grep -qiE "connection refused|timeout|too less sharders|unexpected end"; then
            retry=$((retry + 1))
            echo -e "    ${YELLOW}Network error, retrying in ${backoff}s ($retry/$max_retries)...${NC}"
            sleep $backoff
            backoff=$((backoff * 2))
            [ $backoff -gt 20 ] && backoff=20
            continue
        fi

        # Command completed (may have succeeded)
        if [ $exit_code -eq 0 ]; then
            echo "$output" | grep -E "Hash|error" || true
            return 0
        fi

        # Failed - retry
        retry=$((retry + 1))
        if [ $retry -lt $max_retries ]; then
            echo -e "    ${YELLOW}Retrying in ${backoff}s ($retry/$max_retries)...${NC}"
            sleep $backoff
            backoff=$((backoff * 2))
            [ $backoff -gt 20 ] && backoff=20
        fi
    done

    echo -e "    ${RED}Max retries reached - chain may be stuck${NC}"
    echo -e "    ${RED}[FAILURE] Transaction failed after $max_retries retries${NC}"
    FAILURES=$((FAILURES + 1))

    # Pause and wait for chain recovery before continuing
    echo -e "    ${YELLOW}Pausing test - waiting for chain to recover...${NC}"
    pause_for_chain_recovery
    return 1
}

# Pause test and wait for chain recovery
pause_for_chain_recovery() {
    local max_wait=600  # 10 minutes max wait
    local check_interval=15
    local start_time=$(date +%s)

    echo -e "  ${RED}========================================${NC}"
    echo -e "  ${RED}  TEST PAUSED - CHAIN LIKELY STUCK     ${NC}"
    echo -e "  ${RED}========================================${NC}"
    echo -e "  ${YELLOW}Waiting for chain to show progress before resuming tests...${NC}"
    echo -e "  ${YELLOW}Check miner logs: docker logs miner-1 2>&1 | tail -50${NC}"
    echo ""

    local last_round=$(get_current_round)
    local consecutive_progress=0
    local required_progress=3  # Need 3 consecutive progress checks

    while true; do
        local elapsed=$(($(date +%s) - start_time))
        sleep $check_interval
        local current_round=$(get_current_round)

        if [ "$current_round" -gt "$last_round" ]; then
            consecutive_progress=$((consecutive_progress + 1))
            echo -e "  ${GREEN}Chain progressing: $last_round -> $current_round (+$consecutive_progress/$required_progress)${NC}"
            last_round=$current_round

            if [ $consecutive_progress -ge $required_progress ]; then
                echo -e "  ${GREEN}Chain recovered! Resuming tests...${NC}"
                echo -e "  ${GREEN}========================================${NC}"
                return 0
            fi
        else
            consecutive_progress=0
            echo -e "  ${YELLOW}Chain still stuck at round $current_round (${elapsed}s/${max_wait}s)${NC}"
        fi

        if [ $elapsed -ge $max_wait ]; then
            echo -e "  ${RED}Chain still stuck after ${max_wait}s - resuming tests anyway${NC}"
            echo -e "  ${RED}WARNING: Further failures are likely!${NC}"
            echo -e "  ${RED}========================================${NC}"
            return 1
        fi
    done
}

# Log function with timestamp (like chaos script)
log() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Initialize log file
echo "=== View Change Loop Started at $(date) ===" > "$LOG_FILE"
echo "Log file: $LOG_FILE"
echo "Monitor with: tail -f $LOG_FILE"
echo ""

# Redirect all output to both terminal and log file
exec > >(tee -a "$LOG_FILE") 2>&1

# Get current round from diagnostics
get_current_round() {
    local best=0
    for port in 7071 7072 7073 7074 7171 7172; do
        local r=$(curl -s --connect-timeout 2 "http://localhost:${port}/_diagnostics" 2>/dev/null | grep -oE "Latest Finalized Round</td><td[^>]*>([0-9]+)" | grep -oE "[0-9]+")
        if [ -n "$r" ] && [ "$r" -gt "$best" ] 2>/dev/null; then
            best=$r
        fi
    done
    echo "${best:-0}"
}

# Get current magic block number from 0dns (accurate) with diagnostics fallback
get_current_mb() {
    # Primary: 0dns magic_block endpoint has the actual MB number
    local mb=$(get_0dns_magic_block_number)
    if [ -n "$mb" ] && [ "$mb" != "null" ] && [ "$mb" -gt 0 ] 2>/dev/null; then
        echo "$mb"
        return
    fi
    # Fallback: diagnostics page (WARNING: LFMB field shows starting round, not MB number)
    local round=$(curl -s "$MINER_DIAG" 2>/dev/null | grep -oE "LFMB</td><td[^>]*>([0-9]+)" | grep -oE "[0-9]+" | head -1)
    echo "${round:-0}"
}

# Get number of miners in current magic block
get_miners_count() {
    local count=$(curl -s "$MINER_DIAG" 2>/dev/null | grep -oE "Miners \([0-9]+\)" | grep -oE "[0-9]+")
    echo "${count:-0}"
}

# Get number of sharders in current magic block
get_sharders_count() {
    local count=$(curl -s "$MINER_DIAG" 2>/dev/null | grep -oE "Sharders \([0-9]+\)" | grep -oE "[0-9]+")
    echo "${count:-0}"
}

# Get miners count from 0dns network endpoint
get_0dns_miners_count() {
    local miners=$(curl -s "$DNS_NETWORK" 2>/dev/null | jq -r '.miners | length' 2>/dev/null)
    echo "${miners:-0}"
}

# Get sharders count from 0dns network endpoint
get_0dns_sharders_count() {
    local sharders=$(curl -s "$DNS_NETWORK" 2>/dev/null | jq -r '.sharders | length' 2>/dev/null)
    echo "${sharders:-0}"
}

# Get miners list from 0dns
get_0dns_miners() {
    curl -s "$DNS_NETWORK" 2>/dev/null | grep -oE '"miners":\[[^\]]*\]' | grep -oE 'http[^"]+' | sort
}

# Get sharders list from 0dns
get_0dns_sharders() {
    curl -s "$DNS_NETWORK" 2>/dev/null | grep -oE '"sharders":\[[^\]]*\]' | grep -oE 'http[^"]+' | sort
}

# Get magic block info from 0dns
get_0dns_magic_block_number() {
    curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r '.magic_block_number' 2>/dev/null
}

get_0dns_magic_block_starting_round() {
    curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r '.starting_round' 2>/dev/null
}

# Get miner IDs from 0dns magic block (short format)
get_0dns_mb_miner_ids() {
    curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r '.miners.nodes | keys[]' 2>/dev/null | cut -c1-16 | sort
}

# Get sharder IDs from 0dns magic block (short format)
get_0dns_mb_sharder_ids() {
    curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r '.sharders.nodes | keys[]' 2>/dev/null | cut -c1-16 | sort
}

# Check if miner ID is in 0dns magic block
is_miner_in_0dns_mb() {
    local miner_id=$1
    local found=$(curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r ".miners.nodes[\"$miner_id\"]" 2>/dev/null)
    [ "$found" != "null" ] && [ -n "$found" ] && return 0 || return 1
}

# Check if sharder ID is in 0dns magic block
is_sharder_in_0dns_mb() {
    local sharder_id=$1
    local found=$(curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r ".sharders.nodes[\"$sharder_id\"]" 2>/dev/null)
    [ "$found" != "null" ] && [ -n "$found" ] && return 0 || return 1
}

# Verify view change completed correctly (simple check - returns 1 if not yet complete)
verify_view_change_check() {
    local expected_miner_in=$1  # "true" or "false" - should miner be in MB?
    local expected_sharder_in=$2  # "true" or "false" - should sharder be in MB?

    local miner_state_ok=true
    local sharder_state_ok=true

    if [ "$expected_miner_in" = "true" ]; then
        if ! is_miner_in_0dns_mb "$MINER_ID"; then
            miner_state_ok=false
        fi
    elif [ "$expected_miner_in" = "false" ]; then
        if is_miner_in_0dns_mb "$MINER_ID"; then
            miner_state_ok=false
        fi
    fi

    if [ "$expected_sharder_in" = "true" ]; then
        if ! is_sharder_in_0dns_mb "$SHARDER_ID"; then
            sharder_state_ok=false
        fi
    elif [ "$expected_sharder_in" = "false" ]; then
        if is_sharder_in_0dns_mb "$SHARDER_ID"; then
            sharder_state_ok=false
        fi
    fi

    if [ "$miner_state_ok" = true ] && [ "$sharder_state_ok" = true ]; then
        return 0
    else
        return 1
    fi
}

# Verify view change completed correctly - waits until state is confirmed
verify_view_change() {
    local operation=$1  # "delete_miner", "add_miner", "delete_sharder", "add_sharder", etc.
    local expected_miner_in=$2  # "true" or "false" - should miner be in MB?
    local expected_sharder_in=$3  # "true" or "false" - should sharder be in MB?
    local max_wait=${4:-300}  # Maximum wait time in seconds (default 5 minutes)

    echo -e "  ${CYAN}Verifying View Change:${NC}"

    local start_time=$(date +%s)
    local last_mb=""

    while true; do
        # Get 0dns magic block info
        local mb_num=$(get_0dns_magic_block_number)
        local mb_round=$(get_0dns_magic_block_starting_round)
        local mb_miners=$(curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r '.miners.nodes | keys | length' 2>/dev/null)
        local mb_sharders=$(curl -s "$DNS_MAGIC_BLOCK" 2>/dev/null | jq -r '.sharders.nodes | keys | length' 2>/dev/null)

        # Only print full status if MB changed or first check
        if [ "$mb_num" != "$last_mb" ]; then
            echo -e "    0dns Magic Block: #${WHITE}$mb_num${NC} (starting round: $mb_round)"
            echo -e "    MB Miners ($mb_miners): $(get_0dns_mb_miner_ids | tr '\n' ' ')"
            echo -e "    MB Sharders ($mb_sharders): $(get_0dns_mb_sharder_ids | tr '\n' ' ')"
            last_mb=$mb_num
        fi

        # Check if target miner is in/out of MB as expected
        local miner_short=${MINER_ID:0:16}
        local sharder_short=${SHARDER_ID:0:16}
        local miner_state_ok=true
        local sharder_state_ok=true

        if [ "$expected_miner_in" = "true" ]; then
            if is_miner_in_0dns_mb "$MINER_ID"; then
                echo -e "    Miner $miner_short: ${GREEN}IN MB (expected)${NC}"
            else
                echo -e "    Miner $miner_short: ${YELLOW}NOT in MB yet${NC}"
                miner_state_ok=false
            fi
        elif [ "$expected_miner_in" = "false" ]; then
            if ! is_miner_in_0dns_mb "$MINER_ID"; then
                echo -e "    Miner $miner_short: ${GREEN}NOT in MB (expected)${NC}"
            else
                echo -e "    Miner $miner_short: ${YELLOW}Still in MB${NC}"
                miner_state_ok=false
            fi
        fi

        if [ "$expected_sharder_in" = "true" ]; then
            if is_sharder_in_0dns_mb "$SHARDER_ID"; then
                echo -e "    Sharder $sharder_short: ${GREEN}IN MB (expected)${NC}"
            else
                echo -e "    Sharder $sharder_short: ${YELLOW}NOT in MB yet${NC}"
                sharder_state_ok=false
            fi
        elif [ "$expected_sharder_in" = "false" ]; then
            if ! is_sharder_in_0dns_mb "$SHARDER_ID"; then
                echo -e "    Sharder $sharder_short: ${GREEN}NOT in MB (expected)${NC}"
            else
                echo -e "    Sharder $sharder_short: ${YELLOW}Still in MB${NC}"
                sharder_state_ok=false
            fi
        fi

        # Print current status
        local round=$(get_current_round)
        local elapsed=$(($(date +%s) - start_time))
        echo -e "    ${CYAN}Status:${NC} Round=$round, MB=#$mb_num"

        if [ "$miner_state_ok" = true ] && [ "$sharder_state_ok" = true ]; then
            echo -e "    ${GREEN}VC VERIFIED OK${NC} (after ${elapsed}s)"
            return 0
        fi

        # Check timeout
        if [ $elapsed -ge $max_wait ]; then
            echo -e "    ${RED}VC VERIFICATION TIMEOUT${NC} (${elapsed}s) - state not as expected"
            echo -e "    ${RED}[FAILURE] View change did not complete in expected state${NC}"
            return 1
        fi

        # Wait and retry
        echo -e "    ${YELLOW}VC state not yet as expected, waiting for next MB... (${elapsed}s/${max_wait}s)${NC}"
        sleep 15
    done
}

# Get miners count from diagnostics page table (miner4)
get_diag_miners_count() {
    local count=$(curl -s "$MINER4_DIAG" 2>/dev/null | grep -oE "Miner[0-9]+" | wc -l | tr -d ' ')
    echo "${count:-0}"
}

# Get sharders count from diagnostics page table (miner4)
get_diag_sharders_count() {
    local count=$(curl -s "$MINER4_DIAG" 2>/dev/null | grep -oE "Sharder[0-9]+" | wc -l | tr -d ' ')
    echo "${count:-0}"
}

# Check if specific miner ID is in diagnostics miners table
is_miner_in_diag_table() {
    local miner_id=$1
    local short_id=${miner_id:0:8}
    local found=$(curl -s "$MINER4_DIAG" 2>/dev/null | grep -c "$short_id")
    [ "$found" -gt 0 ] && return 0 || return 1
}

# Check if specific sharder ID is in diagnostics sharders table
is_sharder_in_diag_table() {
    local sharder_id=$1
    local short_id=${sharder_id:0:8}
    local found=$(curl -s "$MINER4_DIAG" 2>/dev/null | grep -c "$short_id")
    [ "$found" -gt 0 ] && return 0 || return 1
}

# Monitor chain progress - returns 0 if chain progressed, 1 if stuck
# Also outputs progress info
monitor_chain_progress() {
    local test_name=$1
    local max_checks=${2:-$CHAIN_PROGRESS_CHECKS}
    local start_round=$(get_current_round)
    local progress_count=0
    local last_round=$start_round

    echo -e "  ${CYAN}Monitoring chain progress for: $test_name${NC}"
    echo -n "    Checks: "

    for ((i=1; i<=max_checks; i++)); do
        sleep $CHAIN_PROGRESS_INTERVAL
        local current_round=$(get_current_round)

        if [ "$current_round" -gt "$last_round" ]; then
            progress_count=$((progress_count + 1))
            echo -n -e "${GREEN}+${NC}"
            last_round=$current_round
        else
            echo -n -e "${YELLOW}.${NC}"
        fi
    done

    local rounds_gained=$((last_round - start_round))

    if [ $progress_count -ge 3 ]; then
        echo -e " ${GREEN}PASS${NC} (rounds: $start_round -> $last_round, +$rounds_gained)"
        return 0
    elif [ $progress_count -ge 1 ]; then
        echo -e " ${YELLOW}SLOW${NC} (rounds: $start_round -> $last_round, +$rounds_gained, only $progress_count/$max_checks checks passed)"
        return 0
    else
        echo -e " ${RED}STUCK${NC} (rounds: $start_round -> $last_round, no progress)"
        return 1
    fi
}

# Wait for chain to recover - blocks until chain starts progressing
wait_for_chain_recovery() {
    local max_wait=${1:-300}  # Default 5 minutes max wait
    local check_interval=10
    local start_time=$(date +%s)

    echo -e "  ${YELLOW}Chain stuck - waiting for recovery...${NC}"

    while true; do
        local elapsed=$(($(date +%s) - start_time))
        local start_round=$(get_current_round)
        sleep $check_interval
        local current_round=$(get_current_round)

        if [ "$current_round" -gt "$start_round" ]; then
            echo -e "  ${GREEN}Chain recovered! Round advancing: $start_round -> $current_round${NC}"
            return 0
        fi

        if [ $elapsed -ge $max_wait ]; then
            echo -e "  ${RED}Chain still stuck after ${elapsed}s - continuing anyway${NC}"
            return 1
        fi

        echo -e "  ${YELLOW}Still waiting for chain recovery... (${elapsed}s/${max_wait}s)${NC}"
    done
}

# Verify 0dns network reflects view change
verify_0dns_network() {
    local expected_miners=$1
    local expected_sharders=$2
    local dns_miners=$(get_0dns_miners_count)
    local dns_sharders=$(get_0dns_sharders_count)

    echo -n "  0dns network check: "
    if [ "$dns_miners" -eq "$expected_miners" ] && [ "$dns_sharders" -eq "$expected_sharders" ]; then
        echo -e "${GREEN}OK${NC} (miners=$dns_miners, sharders=$dns_sharders)"
        return 0
    else
        echo -e "${YELLOW}MISMATCH${NC} (expected m=$expected_miners/s=$expected_sharders, got m=$dns_miners/s=$dns_sharders)"
        return 1
    fi
}

# Verify specific miner is in/out of the network (non-blocking check)
check_miner_in_network() {
    local miner_id=$1
    # Check 0dns for port 7071 (miner1)
    local dns_data=$(curl -s "$DNS_NETWORK" 2>/dev/null)
    local in_dns=$(echo "$dns_data" | grep -c "7071")
    [ "$in_dns" -gt 0 ] && return 0 || return 1
}

# Wait for miner to be in/out of network with timeout
wait_for_miner_state() {
    local miner_id=$1
    local expected=$2  # "in" or "out"
    local timeout=${3:-60}
    local short_id=${miner_id:0:16}
    local start_time=$(date +%s)

    echo -n "  Waiting for miner $short_id to be $expected "
    while true; do
        local elapsed=$(($(date +%s) - start_time))

        if [ "$expected" = "in" ]; then
            if check_miner_in_network "$miner_id"; then
                echo -e " ${GREEN}OK${NC} (${elapsed}s)"
                return 0
            fi
        else
            if ! check_miner_in_network "$miner_id"; then
                echo -e " ${GREEN}OK${NC} (${elapsed}s)"
                return 0
            fi
        fi

        if [ $elapsed -ge $timeout ]; then
            echo -e " ${RED}TIMEOUT${NC} (${elapsed}s)"
            return 1
        fi

        echo -n "."
        sleep 3
    done
}

# Verify specific miner is in/out of the network (non-blocking display)
verify_miner_in_network() {
    local miner_id=$1
    local expected=$2  # "in" or "out"
    local short_id=${miner_id:0:16}

    # Check 0dns
    local dns_data=$(curl -s "$DNS_NETWORK" 2>/dev/null)
    local in_dns=$(echo "$dns_data" | grep -c "7071")  # miner1 is on port 7071

    echo -n "  Miner $short_id... "
    if [ "$expected" = "in" ]; then
        if [ "$in_dns" -gt 0 ]; then
            echo -e "${GREEN}IN 0dns${NC}"
            return 0
        else
            echo -e "${RED}NOT in 0dns (expected IN)${NC}"
            return 1
        fi
    else
        if [ "$in_dns" -eq 0 ]; then
            echo -e "${GREEN}NOT in 0dns (as expected)${NC}"
            return 0
        else
            echo -e "${YELLOW}Still in 0dns (expected OUT - may take time)${NC}"
            return 0
        fi
    fi
}

# Verify specific sharder is in/out of the network (non-blocking check)
check_sharder_in_network() {
    local sharder_id=$1
    # Check 0dns for port 7171 (sharder1)
    local dns_data=$(curl -s "$DNS_NETWORK" 2>/dev/null)
    local in_dns=$(echo "$dns_data" | grep -c "7171")
    [ "$in_dns" -gt 0 ] && return 0 || return 1
}

# Wait for sharder to be in/out of network with timeout
wait_for_sharder_state() {
    local sharder_id=$1
    local expected=$2  # "in" or "out"
    local timeout=${3:-60}
    local short_id=${sharder_id:0:16}
    local start_time=$(date +%s)

    echo -n "  Waiting for sharder $short_id to be $expected "
    while true; do
        local elapsed=$(($(date +%s) - start_time))

        if [ "$expected" = "in" ]; then
            if check_sharder_in_network "$sharder_id"; then
                echo -e " ${GREEN}OK${NC} (${elapsed}s)"
                return 0
            fi
        else
            if ! check_sharder_in_network "$sharder_id"; then
                echo -e " ${GREEN}OK${NC} (${elapsed}s)"
                return 0
            fi
        fi

        if [ $elapsed -ge $timeout ]; then
            echo -e " ${RED}TIMEOUT${NC} (${elapsed}s)"
            return 1
        fi

        echo -n "."
        sleep 3
    done
}

# Verify specific sharder is in/out of the network (non-blocking display)
verify_sharder_in_network() {
    local sharder_id=$1
    local expected=$2  # "in" or "out"
    local short_id=${sharder_id:0:16}

    # Check 0dns
    local dns_data=$(curl -s "$DNS_NETWORK" 2>/dev/null)
    local in_dns=$(echo "$dns_data" | grep -c "7171")  # sharder1 is on port 7171

    echo -n "  Sharder $short_id... "
    if [ "$expected" = "in" ]; then
        if [ "$in_dns" -gt 0 ]; then
            echo -e "${GREEN}IN 0dns${NC}"
            return 0
        else
            echo -e "${RED}NOT in 0dns (expected IN)${NC}"
            return 1
        fi
    else
        if [ "$in_dns" -eq 0 ]; then
            echo -e "${GREEN}NOT in 0dns (as expected)${NC}"
            return 0
        else
            echo -e "${YELLOW}Still in 0dns (expected OUT - may take time)${NC}"
            return 0
        fi
    fi
}

# Wait for view change to complete
wait_for_view_change() {
    local expected_mb=$1
    local start_time=$(date +%s)
    local timeout=$VC_WAIT_TIME

    echo -n "  Waiting for view change to MB #$expected_mb "
    while true; do
        local current_mb=$(get_current_mb)
        local elapsed=$(($(date +%s) - start_time))

        if [ "$current_mb" -ge "$expected_mb" ]; then
            echo -e " ${GREEN}OK${NC} (MB #$current_mb after ${elapsed}s)"
            return 0
        fi

        if [ $elapsed -ge $timeout ]; then
            echo -e " ${RED}TIMEOUT${NC} (stuck at MB #$current_mb)"
            return 1
        fi

        echo -n "."
        sleep 5
    done
}

# Print status summary
print_status() {
    local round=$(get_current_round)
    local mb=$(get_current_mb)
    local miners=$(get_miners_count)
    local sharders=$(get_sharders_count)
    local dns_miners=$(get_0dns_miners_count)
    local dns_sharders=$(get_0dns_sharders_count)
    echo "  Status: Round=$round, MB=#$mb"
    echo "    Chain: Miners=$miners, Sharders=$sharders"
    echo "    0dns:  Miners=$dns_miners, Sharders=$dns_sharders"
}

# Verify view change with transaction retry if needed
# Returns 0 on success, 1 on failure
verify_with_retry() {
    local operation=$1
    local expected_miner_in=$2
    local expected_sharder_in=$3
    local max_txn_retries=3
    local txn_retry=0

    while [ $txn_retry -lt $max_txn_retries ]; do
        verify_view_change "$operation" "$expected_miner_in" "$expected_sharder_in"
        local result=$?

        if [ $result -eq 0 ]; then
            return 0  # Success
        elif [ $result -eq 2 ]; then
            # Transaction failed - retry the transaction
            txn_retry=$((txn_retry + 1))
            echo -e "  ${YELLOW}Transaction failed, retrying ($txn_retry/$max_txn_retries)...${NC}"

            # Re-execute the appropriate transaction based on expected state
            if [ "$expected_miner_in" = "true" ]; then
                echo -e "  ${CYAN}Re-submitting: vc-add miner${NC}"
                $ZWALLET_PATH vc-add --id $MINER_ID --provider-type miner --config $CONFIG --wallet $WALLET 2>&1 | grep -E "success|error|Hash"
            elif [ "$expected_miner_in" = "false" ]; then
                echo -e "  ${CYAN}Re-submitting: mn-delete${NC}"
                $ZWALLET_PATH mn-delete --id $MINER_ID --wallet $WALLET --config $CONFIG 2>&1 | grep -E "success|error|Hash"
            fi

            if [ "$expected_sharder_in" = "true" ]; then
                echo -e "  ${CYAN}Re-submitting: vc-add sharder${NC}"
                $ZWALLET_PATH vc-add --id $SHARDER_ID --provider-type sharder --wallet $WALLET --config $CONFIG 2>&1 | grep -E "success|error|Hash"
            elif [ "$expected_sharder_in" = "false" ]; then
                echo -e "  ${CYAN}Re-submitting: sh-delete${NC}"
                $ZWALLET_PATH sh-delete --id $SHARDER_ID --config $CONFIG --wallet $WALLET 2>&1 | grep -E "success|error|Hash"
            fi

            echo -e "  ${YELLOW}Waiting 30s for transaction to process...${NC}"
            sleep 30
        else
            # VC timeout (result=1) - don't retry transaction
            return 1
        fi
    done

    echo -e "  ${RED}Transaction retry limit reached${NC}"
    return 1
}

# Run a test step with transaction checking, verification, and chain progress monitoring
# Args: step_num step_name expected_mb expected_miner_in expected_sharder_in cmd1 [cmd2...]
#   expected_miner_in/expected_sharder_in: "true"/"false" = verify state, "skip" = don't check
run_test_step() {
    local step_num=$1
    local step_name=$2
    local expected_mb=$3
    local expected_miner_in=$4
    local expected_sharder_in=$5
    shift 5
    local commands=("$@")

    echo ""
    echo -e "${WHITE}[$step_num] $step_name${NC}"

    local start_round=$(get_current_round)
    local start_mb=$(get_current_mb)
    local test_passed=true
    local txn_failed=false

    # Execute commands and CHECK return values
    echo -e "  ${CYAN}Executing commands:${NC}"
    for cmd in "${commands[@]}"; do
        if ! run_zwallet_cmd "$cmd"; then
            txn_failed=true
        fi
        sleep 2  # Brief pause between commands
    done

    if [ "$txn_failed" = true ]; then
        echo -e "  ${RED}[FAIL] One or more transactions failed${NC}"
        test_passed=false
    fi

    echo "  Sleeping ${SLEEP_TIME}s..."
    sleep $SLEEP_TIME

    # Wait for view change (MB number must advance)
    if ! wait_for_view_change $expected_mb; then
        test_passed=false
        echo -e "  ${RED}[FAIL] View change did not complete${NC}"
    fi

    # Verify expected miner/sharder state via 0dns magic block
    if [ "$expected_miner_in" != "skip" ] || [ "$expected_sharder_in" != "skip" ]; then
        if ! verify_view_change "$step_name" "$expected_miner_in" "$expected_sharder_in"; then
            test_passed=false
        fi
    fi

    # Monitor chain progress - wait for recovery if stuck
    if ! monitor_chain_progress "$step_name"; then
        echo -e "  ${RED}[FAIL] Chain not progressing after $step_name${NC}"
        if wait_for_chain_recovery 300; then
            echo -e "  ${GREEN}Chain recovered - continuing tests${NC}"
        else
            test_passed=false
        fi
    fi

    local end_round=$(get_current_round)
    local end_mb=$(get_current_mb)
    local rounds_delta=$((end_round - start_round))
    local mb_delta=$((end_mb - start_mb))

    # Record result - FAILURES counted here ONLY
    if [ "$test_passed" = true ]; then
        TEST_RESULTS+=("PASS")
        echo -e "  ${GREEN}[RESULT] $step_name: PASS${NC} (rounds +$rounds_delta, MB +$mb_delta)"
    else
        TEST_RESULTS+=("FAIL")
        FAILURES=$((FAILURES + 1))
        echo -e "  ${RED}[RESULT] $step_name: FAIL${NC} (rounds +$rounds_delta, MB +$mb_delta)"
    fi
    TEST_NAMES+=("$step_name")

    return $([ "$test_passed" = true ] && echo 0 || echo 1)
}

# Print results table
print_results_table() {
    echo ""
    echo "=============================================="
    echo "              TEST RESULTS"
    echo "=============================================="
    printf "%-40s %s\n" "Test Name" "Result"
    echo "----------------------------------------------"

    local pass_count=0
    local fail_count=0

    for i in "${!TEST_NAMES[@]}"; do
        local name="${TEST_NAMES[$i]}"
        local result="${TEST_RESULTS[$i]}"

        if [ "$result" = "PASS" ]; then
            printf "%-40s ${GREEN}%s${NC}\n" "$name" "$result"
            pass_count=$((pass_count + 1))
        else
            printf "%-40s ${RED}%s${NC}\n" "$name" "$result"
            fail_count=$((fail_count + 1))
        fi
    done

    echo "----------------------------------------------"
    echo -e "Total: ${GREEN}$pass_count PASSED${NC}, ${RED}$fail_count FAILED${NC}"
    echo "=============================================="
}

# Cleanup on exit
cleanup() {
    echo ""
    echo ""
    print_results_table
    echo ""
    echo "================================"
    echo "Final Summary:"
    echo "  Iterations: $ITERATION"
    echo "  Total Failures: $FAILURES"
    echo "================================"
    exit 0
}

trap cleanup SIGINT SIGTERM

echo "Starting View Change Loop Test with Chain Progress Monitoring"
echo "Press Ctrl+C to stop and see results"
echo "=============================================================="
echo ""

# Initial status
echo "Initial Status:"
print_status
PREV_ROUND=$(get_current_round)
PREV_MB=$(get_current_mb)

# Initial chain progress check
echo ""
echo "Checking initial chain health..."
if ! monitor_chain_progress "Initial Health Check"; then
    echo -e "${RED}WARNING: Chain not progressing at start!${NC}"
fi

echo ""

while true; do
    ITERATION=$((ITERATION + 1))
    echo ""
    echo "=========================================================="
    echo -e "${WHITE}              ITERATION $ITERATION${NC}"
    echo "=========================================================="
    echo "$(date)"

    # Clear results for this iteration
    TEST_RESULTS=()
    TEST_NAMES=()

    # Randomly select a miner and sharder for this iteration
    MINER_ID=${MINER_IDS[$((RANDOM % ${#MINER_IDS[@]}))]}
    SHARDER_ID=${SHARDER_IDS[$((RANDOM % ${#SHARDER_IDS[@]}))]}
    echo -e "  ${CYAN}Selected miner:  ${MINER_ID:0:16}...${NC}"
    echo -e "  ${CYAN}Selected sharder: ${SHARDER_ID:0:16}...${NC}"

    # Initialize nonce on first iteration
    initialize_nonce

    # Fund wallet with faucet before each iteration (100 tokens once)
    echo -e "  ${CYAN}Funding wallet from faucet (100 tokens)...${NC}"
    run_zwallet_cmd "$ZWALLET_PATH faucet --methodName pour --input \"{}\" --tokens 100 --wallet $WALLET --config $CONFIG"
    sleep 3  # Wait for faucet transaction to be confirmed

    # Record starting MB for verification
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 1: Delete both miner and sharder
    run_test_step "1" "Delete miner + sharder" $EXPECTED_MB "false" "false" \
        "$ZWALLET_PATH mn-delete --id $MINER_ID --wallet $WALLET --config $CONFIG" \
        "$ZWALLET_PATH sh-delete --id $SHARDER_ID --config $CONFIG --wallet $WALLET"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 2: Add both miner and sharder
    run_test_step "2" "Add miner + sharder" $EXPECTED_MB "true" "true" \
        "$ZWALLET_PATH vc-add --id $MINER_ID --provider-type miner --config $CONFIG --wallet $WALLET" \
        "$ZWALLET_PATH vc-add --id $SHARDER_ID --provider-type sharder --wallet $WALLET --config $CONFIG"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 3: Delete miner only
    run_test_step "3" "Delete miner only" $EXPECTED_MB "false" "true" \
        "$ZWALLET_PATH mn-delete --id $MINER_ID --wallet $WALLET --config $CONFIG"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 4: Delete sharder only
    run_test_step "4" "Delete sharder only" $EXPECTED_MB "false" "false" \
        "$ZWALLET_PATH sh-delete --id $SHARDER_ID --config $CONFIG --wallet $WALLET"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 5: Add miner only
    run_test_step "5" "Add miner only" $EXPECTED_MB "true" "false" \
        "$ZWALLET_PATH vc-add --id $MINER_ID --provider-type miner --config $CONFIG --wallet $WALLET"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 6: Add sharder only
    run_test_step "6" "Add sharder only" $EXPECTED_MB "true" "true" \
        "$ZWALLET_PATH vc-add --id $SHARDER_ID --provider-type sharder --wallet $WALLET --config $CONFIG"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 7: Delete miner only
    run_test_step "7" "Delete miner (again)" $EXPECTED_MB "false" "true" \
        "$ZWALLET_PATH mn-delete --id $MINER_ID --wallet $WALLET --config $CONFIG"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 8: Add miner + Delete sharder
    run_test_step "8" "Add miner + Delete sharder" $EXPECTED_MB "true" "false" \
        "$ZWALLET_PATH vc-add --id $MINER_ID --provider-type miner --config $CONFIG --wallet $WALLET" \
        "$ZWALLET_PATH sh-delete --id $SHARDER_ID --config $CONFIG --wallet $WALLET"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 9: Add sharder + Delete miner
    run_test_step "9" "Add sharder + Delete miner" $EXPECTED_MB "false" "true" \
        "$ZWALLET_PATH vc-add --id $SHARDER_ID --provider-type sharder --wallet $WALLET --config $CONFIG" \
        "$ZWALLET_PATH mn-delete --id $MINER_ID --wallet $WALLET --config $CONFIG"
    EXPECTED_MB=$(($(get_current_mb) + 1))

    # Step 10: Add miner back (restore to initial state)
    run_test_step "10" "Add miner back (restore)" $EXPECTED_MB "true" "true" \
        "$ZWALLET_PATH vc-add --id $MINER_ID --provider-type miner --config $CONFIG --wallet $WALLET"

    # Print iteration results
    echo ""
    echo "=========================================================="
    echo -e "${WHITE}         ITERATION $ITERATION COMPLETE${NC}"
    echo "=========================================================="
    print_results_table
    print_status
    echo ""
done
