#!/bin/sh

# create random file
head -c 5M < /dev/urandom > upload.bin

# upload it
./zboxcli/zbox --wallet testing.json upload \
    --allocation `cat ~/.zcn/allocation.txt` \
    --commit \
    --localpath=upload.bin \
    --remotepath=/remote/upload.bin
