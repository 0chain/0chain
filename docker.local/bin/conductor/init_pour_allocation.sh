#!/bin/sh

set -e

rm -rf ~/.zcn/testing.json
# rm -rf ~/.zcn/testing-auth.json
rm -rf ~/.zcn/allocation.txt

for run in $(seq 1 10)
do
  ./zwalletcli/zwallet --wallet testing.json faucet \
      --methodName pour --input "{Pay day}"
done

# for run in $(seq 1 4)
# do
#   ./zwalletcli/zwallet \
#       --wallet testing-auth.json faucet \
#       --methodName pour --input "{Pay day}"
# done

./zwalletcli/zwallet --wallet testing.json getbalance
# ./zwalletcli/zwallet --wallet testing-auth.json getbalance

BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d

# stake blobbers
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER1 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER2 --tokens 2

# for test logs
./zboxcli/zbox --wallet testing.json ls-blobbers

# create allocation
./zboxcli/zbox --wallet testing.json newallocation \
    --read_price 0.001-10 --write_price 0.01-10 --size 104857600 \
    --lock 0.01953125 --data 1 --parity 1 --expire 48h

# create random file
head -c 52428800 < /dev/urandom > random.bin

# upload initial file
./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=random.bin \
    --remotepath=/remote/random.bin

# and delete it then
./zboxcli/zbox --wallet testing.json delete \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --remotepath=/remote/random.bin

# client id (doesn't work)
#
# "$(grep -Po '"client_id":.*?[^\\]"' ~/.zcn/testing-auth.json | awk -F':' '{print $2}')"

# get auth ticket
#
# "$(./zboxcli/zbox --wallet testing.json share --allocation "$(cat ~/.zcn/allocation.txt)" --remotepath=/remote/remote.bin --clientid "$(grep -Po '"client_id":.*?[^\\]"' ~/.zcn/testing-auth.json | awk -F':' '{print $2}')" | cut -c13-)"

# 10% of 104857600 is
#
#             1G             104857600
#    -------------------  = -----------
#     0.1 (write price)         x
#
