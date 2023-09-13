#!/bin/sh

set -e

remotepath=$1
destname=$2

# upload it
./zboxcli/zbox --wallet testing.json rename \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --remotepath=$remotepath \
    --destname=$destname
