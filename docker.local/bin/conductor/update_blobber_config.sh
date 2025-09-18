#!/bin/bash

set -e

blobber_id=$1
key=$2
value=$3

printf '{"client_id":"c4a3573c7f7c2e31210c1988d49ee2b6c2009fe67613457904ea7109cae1f4c9","client_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","keys":[{"public_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","private_key":"c06b6f6945ba02d5a3be86b8779deca63bb636ce7e46804a479c50e53c864915"}],"mnemonics":"cactus panther essence ability copper fox wise actual need cousin boat uncover ride diamond group jacket anchor current float rely tragic omit child payment","version":"1.0","date_created":"2021-08-04 18:53:56.949069945 +0100 BST m=+0.018986002"}' > ~/.zcn/testing.json;

./zwalletcli/zwallet --wallet testing.json faucet \
  --methodName pour --input "{Pay day}" --tokens 1;

echo "Updating Blobber $blobber_id, key $key value $value";

./zboxcli/zbox --wallet testing.json bl-update \
    --blobber_id "$blobber_id" --"$key" "$value";