#!/bin/sh

set -e

rm -rf ~/.zcn/testing.json
rm -rf ~/.zcn/allocation.txt

printf '{"client_id":"1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802","client_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","keys":[{"public_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","private_key":"c06b6f6945ba02d5a3be86b8779deca63bb636ce7e46804a479c50e53c864915"}],"mnemonics":"cactus panther essence ability copper fox wise actual need cousin boat uncover ride diamond group jacket anchor current float rely tragic omit child payment","version":"1.0","date_created":"2021-08-04 18:53:56.949069945 +0100 BST m=+0.018986002"}' > ~/.zcn/owner.json


./zwalletcli/zwallet mn-update-config --keys cost.add_hardfork --values 200 --wallet wallet.json --config ./config.yaml --configDir . --silent
echo "adding harfork APOLLO"
./zwalletcli/zwallet add-hardfork -n apollo -r 0 --wallet wallet.json --config ./config.yaml --configDir . --silent
echo "adding harfork ARES"
./zwalletcli/zwallet add-hardfork -n ares -r 0 --wallet wallet.json --config ./config.yaml --configDir . --silent
echo "adding harfork ARTEMIS"
./zwalletcli/zwallet add-hardfork -n artemis -r 0 --wallet wallet.json --config ./config.yaml --configDir . --silent
echo "adding harfork ATHENA"
./zwalletcli/zwallet add-hardfork -n athena -r 0  --wallet wallet.json --config ./config.yaml --configDir . --silent
echo "adding harfork DEMETER"
./zwalletcli/zwallet add-hardfork -n demeter -r 0  --wallet wallet.json --config ./config.yaml --configDir . --silent
echo "adding harfork ELECTRA"
./zwalletcli/zwallet add-hardfork -n electra -r 0  --wallet wallet.json --config ./config.yaml --configDir . --silent
echo "adding harfork HERCULES"
./zwalletcli/zwallet add-hardfork -n hercules -r 0  --wallet wallet.json --config ./config.yaml --configDir . --silent
