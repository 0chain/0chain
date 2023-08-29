#!/bin/bash

provider_type=$1
provider_id=$2
key=$3
monotonicity=$4

case "${provider_type}" in
    miner)
        aggregate_path="miner-aggregates"
        id_field="miner_id"
    ;;
    sharder)
        aggregate_path="sharder-aggregates"
        id_field="sharder_id"
    ;;
    blobber)
        aggregate_path="blobber-aggregates"
        id_field="blobber_id"
    ;;
    validator)
        aggregate_path="validator-aggregates"
        id_field="validator_id"
    ;;
    authorizer)
        aggregate_path="authorizer-aggregates"
        id_field="authorizer_id"
    ;;
    *)
        echo "invalid provider_type"; exit -1;
    ;;
esac

rm -r /tmp/agg-response;

url="http://localhost:9081/v2/$aggregate_path";

echo URL = $url, PID = $provider_id, Key = $key, Mono = $monotonicity;

curl -s $url > /tmp/agg-response;
cat /tmp/agg-response;
cur_val=$(jq --arg PID $provider_id --arg IDF $id_field --arg KEY $key \
    '.[] | select(.[$IDF] == $PID) | .[$KEY]' /tmp/agg-response);
prev_val=$cur_val;

while [ true ]; do
    echo cur = $cur_val, prev = $prev_val
    case "${monotonicity}" in
        "INC")
            [ $cur_val -gt $prev_val ] && exit 0;
        ;;
        "DEC")
            [ $cur_val -lt $prev_val ] && exit 0;
        ;;
        *)
            echo "unknown monotonicity"; exit -1;
        ;;
    esac
    
    prev_val=$cur_val;
    curl -s $url > /tmp/agg-response;
    cur_val=$(jq --arg PID $provider_id --arg IDF $id_field --arg KEY $key \
        '.[] | select(.[$IDF] == $PID) | .[$KEY]' /tmp/agg-response);

    sleep 1s;
done
