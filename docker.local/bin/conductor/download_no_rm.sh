#!/bin/sh

set -e

# add to read pools
./zboxcli/zbox --wallet testing.json rp-lock \
    --duration=1h --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

# # auth user
# ./zboxcli/zbox --wallet testing-auth.json rp-lock \
#     --duration=1h --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0


# create random file
head -c 52428800 < /dev/urandom > random.bin

./zboxcli/zbox \
    --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=random.bin \
    --remotepath=/remote/random.bin

trap "kill 0" EXIT

go run 0chain/code/go/0chain.net/conductor/sdkproxy/main.go -f read_marker &
sleep 3

# cleanup
rf -f got.bin

# download without read_marker
HTTP_PROXY="http://0.0.0.0:15211" ./zboxcli/zbox \
    --wallet testing.json download \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=got.bin \
    --remotepath=/remote/random.bin
