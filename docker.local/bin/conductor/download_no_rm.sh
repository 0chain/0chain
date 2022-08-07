#!/bin/sh

set -e

# add to read pools
./zboxcli/zbox --wallet testing.json rp-lock \
 --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

# auth user
# ./zboxcli/zbox --wallet testing-auth.json rp-lock \
#     --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

go run 0chain/code/go/0chain.net/conductor/sdkproxy/main.go -f read_marker \
    -run 0chain/docker.local/bin/conductor/proxied/download_b.sh
