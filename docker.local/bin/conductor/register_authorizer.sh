#!/bin/sh
set -e

rm -rf ~/.zcn/testing.json

printf '{"client_id":"1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802","client_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","keys":[{"public_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","private_key":"c06b6f6945ba02d5a3be86b8779deca63bb636ce7e46804a479c50e53c864915"}],"mnemonics":"cactus panther essence ability copper fox wise actual need cousin boat uncover ride diamond group jacket anchor current float rely tragic omit child payment","version":"1.0","date_created":"2021-08-04 18:53:56.949069945 +0100 BST m=+0.018986002"}' > ~/.zcn/testing.json

for run in $(seq 1 2)
do
  ./zwalletcli/zwallet --wallet testing.json faucet \
      --methodName pour --input "{Pay day}"
done

./zwalletcli/zwallet auth-register --url http://198.18.0.131 --client_key c3c3976cacfe05b719aa167a0446260bb71ae4975647febaf257321e612e7812b30535a8fbc615ba65bad73fae5cf538fb7798805a63408d22a89970a289d988 --client_id fb276b545a4c2f1d1771f2f9cdbf106f78ea1b2c7ea4ca763161145ec891aa26 --min_stake 1 --max_stake 10 --num_delegates 5 --service_charge 0.0 --wallet testing.json
./zwalletcli/zwallet auth-register --url http://198.18.0.132 --client_key c08ffe6d3a85353c42ea8ad785978569f32c2fb5283e4be9a3ec23ef89f28a1cd5a5c2affafa903989ae3d71a0ec2dfa6bbce155b8a818f1929b51538b705b11 --client_id 45ad4982464f7e47d1160b608fb9285a1d0a1e1f6242f00acb02118179224f13 --min_stake 1 --max_stake 10 --num_delegates 5 --service_charge 0.0 --wallet testing.json
./zwalletcli/zwallet auth-register --url http://198.18.0.133 --client_key 7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518 --client_id 1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802 --min_stake 1 --max_stake 10 --num_delegates 5 --service_charge 0.0 --wallet testing.json

