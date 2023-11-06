#!/bin/sh

set -e

for i in {1..5}; do
    ./zwalletcli/zwallet --wallet testing.json faucet \
      --methodName pour --input "{Pay day}"
done

./zboxcli/zbox --wallet testing.json sp-lock \
    --validator_id 41313b795d2c057b6277801e9ed277b444770c2af75f5209afd00bd07c72cc0b \
    --tokens 1

./zboxcli/zbox --wallet testing.json sp-lock \
    --validator_id ab549edb7cea822dab0b460f65dcde85f698c1e97d730e3ffc6b0f8b576b65bd \
    --tokens 1

./zboxcli/zbox --wallet testing.json sp-lock \
    --validator_id 86cf791f03f01e3e4d318b1ca009a51c91dd43f7cf3c87a32f531b609cc5044b \
    --tokens 1

./zboxcli/zbox --wallet testing.json sp-lock \
    --validator_id 823cb45de27dfe739b320dcf6449e5fdea35c60804fd81d6f22c005042cfb337 \
    --tokens 1
