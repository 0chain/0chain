#!/bin/bash

ptype=$1
ids=$2


# TODO: Extend for other providers
command=""
id_field=""
case "${ptype}" in
    "blobber")
        command="sp-lock"
        id_field="blobber_id"
    ;;
    "validator")
        command="sp-lock"
        id_field="validator_id"
    ;;
    *)
        echo "unknown provider type $ptype"
        exit 1
    ;;
esac

./zwalletcli/zwallet --wallet testing.json faucet \
    --methodName pour --input "{Pay day}" --tokens 99


for id in ${ids//,/ }
do
    echo "Staking $id"
    ./zboxcli/zbox --wallet testing.json $command \
        --${id_field} $id \
        --tokens 5
done
