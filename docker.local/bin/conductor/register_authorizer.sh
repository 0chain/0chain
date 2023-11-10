#!/bin/sh
set -e

rm -rf ~/.zcn/wallet.json

printf '{"client_id":"1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802","client_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","keys":[{"public_key":"7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518","private_key":"c06b6f6945ba02d5a3be86b8779deca63bb636ce7e46804a479c50e53c864915"}],"mnemonics":"cactus panther essence ability copper fox wise actual need cousin boat uncover ride diamond group jacket anchor current float rely tragic omit child payment","version":"1.0","date_created":"2021-08-04 18:53:56.949069945 +0100 BST m=+0.018986002"}' > ~/.zcn/wallet.json

./zwalletcli/zwallet --wallet wallet.json faucet \
      --methodName pour --input "{Pay day}" --tokens 100

./zwalletcli/zwallet getbalance --wallet wallet.json

./zwalletcli/zwallet send --tokens 10 --desc 'to auth 2' --to_client_id 47c534abb2bcb33e9944aee9a0df0e0adc4c0b659b9499aa656920975c38a80a
./zwalletcli/zwallet send --tokens 10 --desc 'to auth 3' --to_client_id 7f2097074f678d08146e5585d6965b04307939fee0457ea18c4242bff197c65a
./zwalletcli/zwallet send --tokens 10 --desc 'to auth 3' --to_client_id d5b9204835a6ea8ba93a0b4f14ce38bc72dec5165465416f0a3b9f66f988e1ab

./zwalletcli/zwallet auth-register --url http://198.18.0.131:3031 --client_key 7b630ba670dac2f22d43c2399b70eff378689a53ee03ea20957bb7e73df016200fea410ba5102558b0c39617e5afd2c1843b161a1dedec15e1ab40543a78a518 --client_id 1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802 --min_stake 2 --max_stake 10 --num_delegates 5 --service_charge 0.1 --wallet wallet.json
./zwalletcli/zwallet auth-register --url http://198.18.0.132:3032 --client_key 326759d10f6f6534e28852eed3347c3b27ec6fb4e549b689cf033d9cbee463223f4bd2e17405e738f8c42f58232e1f37b6f8cbb75b242566aab486efcd19700d --client_id 47c534abb2bcb33e9944aee9a0df0e0adc4c0b659b9499aa656920975c38a80a --min_stake 2 --max_stake 10 --num_delegates 5 --service_charge 0.1 --wallet wallet.json
./zwalletcli/zwallet auth-register --url http://198.18.0.133:3033 --client_key 5cd52e8da7d6814edfd9e3ede49eee4b3e45292daed3341bd551c477f0cbe41f12dafd37f381777609775429e796e1640ceddeeb30fff23caca84d76672a96a0 --client_id 7f2097074f678d08146e5585d6965b04307939fee0457ea18c4242bff197c65a --min_stake 2 --max_stake 10 --num_delegates 5 --service_charge 0.1 --wallet wallet.json
./zwalletcli/zwallet auth-register --url http://198.18.0.134:3034 --client_key 0db96df65ad705ca3b3139b02071bfb611523c53c5ab4693cff08588a3ba5e067e4890144650ea0b0428dbd4de3ce5da6874a8822cf2838549ea5cc26f89ea1b --client_id d5b9204835a6ea8ba93a0b4f14ce38bc72dec5165465416f0a3b9f66f988e1ab --min_stake 2 --max_stake 10 --num_delegates 5 --service_charge 0.1 --wallet wallet.json

./zboxcli/zbox sp-lock --authorizer_id 1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802 --tokens 5 --wallet wallet.json
./zboxcli/zbox sp-lock --authorizer_id 47c534abb2bcb33e9944aee9a0df0e0adc4c0b659b9499aa656920975c38a80a --tokens 5 --wallet wallet.json
./zboxcli/zbox sp-lock --authorizer_id 7f2097074f678d08146e5585d6965b04307939fee0457ea18c4242bff197c65a --tokens 5 --wallet wallet.json
./zboxcli/zbox sp-lock --authorizer_id d5b9204835a6ea8ba93a0b4f14ce38bc72dec5165465416f0a3b9f66f988e1ab --tokens 5 --wallet wallet.json
