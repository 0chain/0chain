#!/bin/bash

id=$1

if ! [ "$(command -v jq)" ]; then
    echo 'jq not found, installing'
    apt install -y jq 
fi

if ! [ "$(command -v bc)" ]; then
    echo 'bc not found, installing'
    apt install -y bc
fi

prev_rewards=$(./zboxcli/zbox --wallet testing.json sp-info \
        --blobber_id "$id" \
        --json --silent | jq '.rewards')

if [ "$prev_rewards" == "" ]; then
    echo "blobber doesn't exist"
    exit 1
fi

echo "Rewards now: $prev_rewards"
while true; do
    output=$(./zboxcli/zbox --wallet testing.json sp-info \
        --blobber_id "$id" \
        --json --silent)
    echo "Output now: $output"
    reward=$(echo "$output" | jq '.rewards')
    echo "Rewards now: $reward"
    if (( $(echo "$reward > $prev_rewards" | bc -l) )); then
        echo "Increased"
        exit 0
    fi
    prev_rewards=$reward
    sleep 2s
done