#!/bin/sh

set -e

trap "kill 0" EXIT

go run 0chain/code/go/0chain.net/conductor/sdkproxy/main.go -f uploadFile &
sleep 3

# create random file
head -c 52428800 < /dev/urandom > random.bin

# upload initial file
HTTP_PROXY="http://0.0.0.0:15211" ./zboxcli/zbox \
    --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=random.bin \
    --remotepath=/remote/upload.bin
