#!/bin/sh

set -e

# add tokens to write pools
./zboxcli/zbox --wallet testing.json wp-lock \
    --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

# create random file
head -c 52428800 < /dev/urandom > random.bin

go run 0chain/code/go/0chain.net/conductor/sdkproxy/main.go -f uploadMeta \
    -run 0chain/docker.local/bin/conductor/proxied/upload_b.sh
