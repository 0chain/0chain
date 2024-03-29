#!/bin/sh

set -e

# create random file
head -c 52428800 < /dev/urandom > random.bin

go run 0chain/code/go/0chain.net/conductor/sdkproxy/main.go -f uploadFile -l "0chain/conductor/logs" \
    -run 0chain/docker.local/bin/conductor/proxied/upload_b.sh
