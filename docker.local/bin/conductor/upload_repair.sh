#!/bin/sh

set -e

if [ ! -d "$(pwd)/testrepair" ]
then
    mkdir testrepair
fi

# create random files
head -c 10M < /dev/urandom > testrepair/repair1.bin
head -c 15M < /dev/urandom > testrepair/repair2.bin
head -c 20M < /dev/urandom > testrepair/repair3.bin

# upload
./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath="$(pwd)/testrepair/repair1.bin" \
    --remotepath=/remote/repair/repair1.bin

sleep 60

./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath="$(pwd)/testrepair/repair2.bin" \
    --remotepath=/remote/repair/repair2.bin

sleep 60

./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath="$(pwd)/testrepair/repair3.bin" \
    --remotepath=/remote/repair/repair3.bin
