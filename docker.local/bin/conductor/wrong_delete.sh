#!/bin/sh

# create random file
head -c 5M < /dev/urandom > delete.bin

# upload it
./zboxcli/zbox --wallet testing.json upload \
    --allocation `cat ~/.zcn/allocation.txt` \
    --commit \
    --localpath=delete.bin \
    --remotepath=/remote/delete.bin || true

./zboxcli/zbox --wallet testing.json delete \
    --allocation `cat ~/.zcn/allocation.txt` \
    --remotepath /remote/delete.bin
