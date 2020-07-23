#!/bin/sh

set -e

rm -rf ~/.zcn/testing.json
rm -rf ~/.zcn/allocation.txt

for run in {1..20}
do
   ./zwalletcli/zwallet --wallet testing.json faucet --methodName pour --input “{Pay day}”
done

# sometimes the getbalance fails
n=0
until [ "$n" -ge 5 ]
do
   ./zwalletcli/zwallet --wallet testing.json getbalance && break
   n=$((n+1)) 
done

BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
BLOBBER3=2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18

# stake blobbers
./zboxcli/zbox --wallet testing.json sp-lock --blobber_id $BLOBBER1 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock --blobber_id $BLOBBER2 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock --blobber_id $BLOBBER3 --tokens 2

# for test logs
./zboxcli/zbox --wallet testing.json ls-blobbers

# create allocation
./zboxcli/zbox --wallet testing.json newallocation --read_price 0.001-10 \
    --write_price 0.01-10 --size 104857600 --lock 4 --data 2 --parity 1  \
    --expire 48h

# add to read pools
./zboxcli/zbox --wallet testing.json rp-lock --blobber $BLOBBER1 \
    --duration=1h --allocation `cat ~/.zcn/allocation.txt` --tokens 2.0
./zboxcli/zbox --wallet testing.json rp-lock --blobber $BLOBBER2 \
    --duration=1h --allocation `cat ~/.zcn/allocation.txt` --tokens 2.0
./zboxcli/zbox --wallet testing.json rp-lock --blobber $BLOBBER3 \
    --duration=1h --allocation `cat ~/.zcn/allocation.txt` --tokens 2.0

# create random file
head -c 5M < /dev/urandom > random.bin

# upload initial file
./zboxcli/zbox --wallet testing.json upload \
    --allocation `cat ~/.zcn/allocation.txt` \
    --commit \
    --localpath=random.bin \
    --remotepath=/remote/random.bin
