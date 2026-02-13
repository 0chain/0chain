#!/bin/bash
# Monitor script for DKG chaos testing
# Outputs a table every INTERVAL seconds with chain stats, MB changes, DKG loading, chaos events
#
# Usage: ./docker.local/bin/monitor.sh [interval_seconds]
# Default interval: 300 (5 minutes)

INTERVAL=${1:-300}
LOG="/tmp/monitor.log"
CHAOS_LOG="/tmp/chaos.log"
VC_LOG="/tmp/vc.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

PREV_ROUND=0
PREV_TIME=$(date +%s)
PREV_MB_NUM=0
HIGHEST_MB_NUM=0
HIGHEST_MB_SR=0
HIGHEST_MB_N=0
HIGHEST_MB_T=0
HIGHEST_MB_S=0
REPORT_NUM=0

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

get_mb_info() {
    # Get MB info from sharder or miner API (try sharders first, then miners)
    local mb_num="" mb_sr="" miners="" sharders=""
    for port in 7171 7172 7071 7072 7073 7074; do
        local mb_json=$(curl -s --connect-timeout 2 "http://localhost:${port}/v1/block/get/latest_finalized_magic_block" 2>/dev/null)
        if [ -n "$mb_json" ]; then
            mb_num=$(echo "$mb_json" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('magic_block',{}).get('magic_block_number',0))" 2>/dev/null)
            mb_sr=$(echo "$mb_json" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('magic_block',{}).get('starting_round',0))" 2>/dev/null)
            miners=$(echo "$mb_json" | python3 -c "import json,sys; d=json.load(sys.stdin); mb=d.get('magic_block',{}); print(len(mb.get('miners',{}).get('nodes',[])))" 2>/dev/null)
            sharders=$(echo "$mb_json" | python3 -c "import json,sys; d=json.load(sys.stdin); mb=d.get('magic_block',{}); print(len(mb.get('sharders',{}).get('nodes',[])))" 2>/dev/null)
            if [ -n "$mb_num" ] && [ "$mb_num" != "0" ]; then
                break
            fi
        fi
    done
    [ -z "$mb_num" ] && mb_num="?"
    [ -z "$mb_sr" ] && mb_sr=0
    [ -z "$miners" ] && miners=4
    [ -z "$sharders" ] && sharders=2
    # T = ceil(N * 0.6) for default config
    local t=$(echo "($miners * 6 + 9) / 10" | bc 2>/dev/null || echo 3)
    echo "$mb_num $mb_sr $miners $t $sharders"
}

get_dkg_status() {
    local miner=$1
    # Single grep pass over current log file only (not rotated logs — much faster)
    local result=$(docker exec "$miner" sh -c '
        grep -h "stored fresh DKG from in-memory\|SetDKGSFromStore total\|clearing stale\|dkg_key_mismatch\|DKG is nil\|DKG finalized" /0chain/log/0chain.log 2>/dev/null | \
        awk "
            /stored fresh DKG from in-memory/{a++}
            /SetDKGSFromStore total/{b++}
            /clearing stale/{c++}
            /dkg_key_mismatch/{d++}
            /DKG is nil/{e++}
            /DKG finalized/{f++}
            END{printf \"%d/%d/%d/%d/%d/%d\",a+0,b+0,c+0,d+0,e+0,f+0}
        "
    ' 2>/dev/null)
    echo "${result:-0/0/0/0/0/0}"
}

get_recent_chaos() {
    if [ -f "$CHAOS_LOG" ]; then
        # Get chaos events since last report
        local since=$(date -v-${INTERVAL}S '+%H:%M' 2>/dev/null || date -d "-${INTERVAL} seconds" '+%H:%M' 2>/dev/null || echo "00:00")
        tail -50 "$CHAOS_LOG" | grep -E 'stop|start|restart|CHAOS' | tail -5
    fi
}

get_recent_vc() {
    if [ -f "$VC_LOG" ]; then
        tail -20 "$VC_LOG" | grep -E 'view change|add_miner|add_sharder|remove' | tail -3
    fi
}

print_header() {
    echo -e "\n${BOLD}╔══════════════════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║  DKG CHAOS TEST MONITOR — Report #${REPORT_NUM} at $(date '+%Y-%m-%d %H:%M:%S')${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════════════════════════════════════════════════════╝${NC}"
}

print_chain_stats() {
    local round=$1
    local now=$(date +%s)
    local elapsed=$((now - PREV_TIME))
    local blocks_produced=$((round - PREV_ROUND))
    local bps=0
    if [ "$elapsed" -gt 0 ] && [ "$PREV_ROUND" -gt 0 ]; then
        bps=$(echo "scale=2; $blocks_produced / $elapsed" | bc 2>/dev/null || echo "?")
    fi

    local mb_info=$(get_mb_info)
    local mb_num=$(echo "$mb_info" | awk '{print $1}')
    local mb_sr=$(echo "$mb_info" | awk '{print $2}')
    local mb_n=$(echo "$mb_info" | awk '{print $3}')
    local mb_t=$(echo "$mb_info" | awk '{print $4}')
    local mb_s=$(echo "$mb_info" | awk '{print $5}')
    # Guard against non-numeric values when sharders are down
    [[ "$mb_num" =~ ^[0-9]+$ ]] || mb_num=0
    [[ "$mb_sr" =~ ^[0-9]+$ ]] || mb_sr=0
    [[ "$mb_n" =~ ^[0-9]+$ ]] || mb_n=0
    [[ "$mb_t" =~ ^[0-9]+$ ]] || mb_t=0
    [[ "$mb_s" =~ ^[0-9]+$ ]] || mb_s=0
    # High-water mark: never report a lower MB than previously seen
    if [ "$mb_num" -gt "$HIGHEST_MB_NUM" ]; then
        HIGHEST_MB_NUM=$mb_num
        HIGHEST_MB_SR=$mb_sr
        HIGHEST_MB_N=$mb_n
        HIGHEST_MB_T=$mb_t
        HIGHEST_MB_S=$mb_s
    elif [ "$mb_num" -lt "$HIGHEST_MB_NUM" ]; then
        mb_num=$HIGHEST_MB_NUM
        mb_sr=$HIGHEST_MB_SR
        mb_n=$HIGHEST_MB_N
        mb_t=$HIGHEST_MB_T
        mb_s=$HIGHEST_MB_S
    fi
    local mb_boundary=$((mb_sr + 90))

    local mb_delta=$((mb_num - PREV_MB_NUM))
    local mb_rate="0"
    if [ "$elapsed" -gt 0 ] && [ "$PREV_ROUND" -gt 0 ]; then
        mb_rate=$(echo "scale=3; $mb_delta / $elapsed" | bc 2>/dev/null || echo "?")
    fi

    echo -e "\n${CYAN}── Chain Stats ──${NC}"
    printf "  %-25s %s\n" "Current Round:" "$round"
    printf "  %-25s %s blocks in %ss = %s blocks/s\n" "Blocks Since Last:" "$blocks_produced" "$elapsed" "$bps"
    printf "  %-25s MB#%s (SR=%s, M=%s, S=%s, T=%s)\n" "Latest MB:" "$mb_num" "$mb_sr" "$mb_n" "$mb_s" "$mb_t"
    printf "  %-25s %s\n" "MB Boundary (SR+90):" "$mb_boundary"
    printf "  %-25s +%s MBs (%s MB/s)\n" "MB Changes:" "$mb_delta" "$mb_rate"

    PREV_ROUND=$round
    PREV_TIME=$now
    PREV_MB_NUM=$mb_num
}

print_dkg_table() {
    echo -e "\n${CYAN}── DKG Status (fresh_from_mem/loaded_from_store/cleared_stale/mismatch/nil/finalized) ──${NC}"
    printf "  %-12s %s\n" "Container" "Stats"
    printf "  %-12s %s\n" "─────────" "─────"
    for m in miner-1 miner-2 miner-3 miner-4; do
        local running=$(docker ps --format '{{.Names}}' 2>/dev/null | grep -c "^${m}$")
        if [ "$running" -eq 1 ]; then
            local stats=$(get_dkg_status "$m")
            printf "  %-12s %s\n" "$m" "$stats"
        else
            printf "  %-12s ${RED}DOWN${NC}\n" "$m"
        fi
    done
}

print_container_status() {
    echo -e "\n${CYAN}── Container Status ──${NC}"
    printf "  %-12s %-8s %s\n" "Container" "Status" "Uptime"
    printf "  %-12s %-8s %s\n" "─────────" "──────" "──────"
    for c in miner-1 miner-2 miner-3 miner-4 sharder-1 sharder-2; do
        local info=$(docker ps --format '{{.Names}} {{.Status}}' 2>/dev/null | grep "^${c} " || echo "")
        if [ -n "$info" ]; then
            local status=$(echo "$info" | awk '{print $2}')
            local uptime=$(echo "$info" | cut -d' ' -f3-)
            printf "  %-12s ${GREEN}%-8s${NC} %s\n" "$c" "$status" "$uptime"
        else
            printf "  %-12s ${RED}%-8s${NC}\n" "$c" "DOWN"
        fi
    done
}

print_recent_events() {
    echo -e "\n${CYAN}── Recent Chaos Events ──${NC}"
    local events=$(get_recent_chaos)
    if [ -n "$events" ]; then
        echo "$events" | while IFS= read -r line; do
            echo "  $line"
        done
    else
        echo "  (none)"
    fi

    echo -e "\n${CYAN}── Recent VC Events ──${NC}"
    local vc_events=$(get_recent_vc)
    if [ -n "$vc_events" ]; then
        echo "$vc_events" | while IFS= read -r line; do
            echo "  $line"
        done
    else
        echo "  (none)"
    fi
}

# ─── Main Loop ──────────────────────────────────────────────────────────────

echo "Starting DKG Chaos Test Monitor (interval: ${INTERVAL}s)"
echo "Logging to $LOG"

# Initial snapshot
PREV_ROUND=$(get_round)
[[ "$PREV_ROUND" =~ ^[0-9]+$ ]] || PREV_ROUND=0
PREV_TIME=$(date +%s)
mb_info=$(get_mb_info)
PREV_MB_NUM=$(echo "$mb_info" | awk '{print $1}')
[[ "$PREV_MB_NUM" =~ ^[0-9]+$ ]] || PREV_MB_NUM=0

while true; do
    REPORT_NUM=$((REPORT_NUM + 1))
    ROUND=$(get_round)

    {
        print_header
        print_chain_stats "$ROUND"
        print_container_status
        print_dkg_table
        print_recent_events
        echo ""
    } 2>&1 | tee -a "$LOG"

    sleep "$INTERVAL"
done
