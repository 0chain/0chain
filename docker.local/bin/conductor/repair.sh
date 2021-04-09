#!/bin/sh

set -e

./zboxcli/zbox --wallet testing.json start-repair \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --rootpath "$(pwd)/testrepair/" --repairpath /remote/repair/ --verbose

rm -rf testrepair