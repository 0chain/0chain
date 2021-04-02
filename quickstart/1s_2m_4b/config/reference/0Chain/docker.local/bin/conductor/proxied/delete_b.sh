#!/bin/sh

set -e

./zboxcli/zbox --wallet testing.json delete \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --remotepath=/remote/b1.bin
