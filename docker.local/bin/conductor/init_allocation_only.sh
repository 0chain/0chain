#!/bin/sh

set -e

rm -rf ~/.zcn/testing.json
rm -rf ~/.zcn/allocation.txt

printf '{"client_id":"1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802","client_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","keys":[{"public_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","private_key":"c06b6f6945ba02d5a3be86b8779deca63bb636ce7e46804a479c50e53c864915"}],"mnemonics":"cactus panther essence ability copper fox wise actual need cousin boat uncover ride diamond group jacket anchor current float rely tragic omit child payment","version":"1.0","date_created":"2021-08-04 18:53:56.949069945 +0100 BST m=+0.018986002"}' > ~/.zcn/testing.json

for run in $(seq 1 16)
do
  ./zwalletcli/zwallet --wallet testing.json faucet \
      --methodName pour --input "{Pay day}"
done

./zwalletcli/zwallet --wallet testing.json getbalance

BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
BLOBBER3=2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18

# stake blobbers
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER1 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER2 --tokens 2
./zboxcli/zbox --wallet testing.json sp-lock \
    --blobber_id $BLOBBER3 --tokens 2

# for test logs
./zboxcli/zbox --wallet testing.json ls-blobbers

# create allocation
./zboxcli/zbox --wallet testing.json newallocation \
    --read_price 0.001-10 --write_price 0.01-10 --size 104857600 --lock 2 \
    --data 1 --parity 2 --expire 721h

# add to read pools
./zboxcli/zbox --wallet testing.json rp-lock --tokens 4.0
