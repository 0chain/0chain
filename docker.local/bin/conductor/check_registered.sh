#!/bin/bash

set -e

provider_type=$1
blobbers=$2
found=""

command=""
case "${provider_type}" in
    "blobber")
        command="ls-blobbers"
    ;;
    "validator")
        command="ls-validators"
    ;;
    *)
        echo "unknown provider type $provider_type"
        exit 1
    ;;
esac

for blobber in ${blobbers//,/ }; do
    while [[ -z "$found" ]]; do
        echo "Checking blobber $blobber registered"
        # ./zboxcli/zbox --wallet testing.json ls-blobbers | grep $blobber
        found=$(../zboxcli/zbox --wallet testing.json $command --silent | grep "$blobber" || true)
        echo "result: $found"
        if [ -z "$found" ]; then
            echo "Blobber $blobber not registered yet"
            sleep 10
        fi
    done
    found=""
done