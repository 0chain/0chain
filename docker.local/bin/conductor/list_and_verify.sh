#!/bin/bash

set -e

rm -rf ~/.zcn/testing.json
rm -rf ~/.zcn/allocation.txt

./zwalletcli/zwallet --wallet testing.json faucet \
    --methodName pour --input "{Pay day}" --tokens 100

./zwalletcli/zwallet --wallet testing.json getbalance

BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
BLOBBER3=2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18
BLOBBER4=2a4d5a5c6c0976873f426128d2ff23a060ee715bccf0fd3ca5e987d57f25b78e

# stake blobbers
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER1 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER2 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER3 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER4 --tokens 2

# for test logs
./zboxcli/zbox --wallet testing.json ls-blobbers

# create allocation
./zboxcli/zbox --wallet testing.json newallocation \
    --read_price 0.001-10 --write_price 0.01-10 --size 104857600 --lock 2 \
    --data 2 --parity 2

# add to read pools
./zboxcli/zbox --wallet testing.json rp-lock \
    --tokens 4.0

# create random file
head -c 5M < /dev/urandom > random.bin

# upload initial file
./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=random.bin \
    --remotepath=/remote/random.bin

rm -f random.bin

sleep 60;

allocation=$(cat ~/.zcn/allocation.txt)
cmd_output=$(./zboxcli/zbox --wallet testing.json list \
    --allocation "$allocation" \
    --remotepath /remote)

if [[ $cmd_output == *random.bin* ]]
then 
    exit 0
else
    exit 1
fi
