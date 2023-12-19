#!/bin/sh

set -e

rm -f gs1.bin
rm -f gs2.bin

head -c 1024 < /dev/urandom > s1.bin
head -c 2048 < /dev/urandom > s2.bin

rm -f gb1.bin
rm -f gb2.bin

head -c 32428800 < /dev/urandom > b1.bin
head -c 32430848 < /dev/urandom > b2.bin

./zboxcli/zbox --wallet testing.json delete \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --remotepath=/remote/b1.bin

# # upload small file
# ./zboxcli/zbox --wallet testing.json upload \
#     --allocation "$(cat ~/.zcn/allocation.txt)" \
#     --localpath=s1.bin \
#     --remotepath=/remote/s1.bin

# upload large file
./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=b1.bin \
    --remotepath=/remote/b1.bin

# then update both files, download and delete then
go run 0chain/code/go/0chain.net/conductor/sdkproxy/main.go -m 'dm-rm-wm' -l "0chain/conductor/logs" \
    -run 0chain/docker.local/bin/conductor/proxied/update_b.sh            \
    -run 0chain/docker.local/bin/conductor/proxied/download_b.sh          \
    -run 0chain/docker.local/bin/conductor/proxied/delete_b.sh

    #-run 0chain/docker.local/bin/conductor/proxied/update_s.sh            \
    #-run 0chain/docker.local/bin/conductor/proxied/download_s.sh          \
    #-run 0chain/docker.local/bin/conductor/proxied/delete_s.sh
