#!/bin/bash

./zwalletcli/zwallet --wallet testing.json faucet \
  --methodName pour --input "{Pay day}" --tokens 99

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25 \
    --tokens 5 --desc "Send"

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d \
    --tokens 5 --desc "Send"

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id 2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18 \
    --tokens 5 --desc "Send"

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id 2a4d5a5c6c0976873f426128d2ff23a060ee715bccf0fd3ca5e987d57f25b78e \
    --tokens 5 --desc "Send"

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id 41313b795d2c057b6277801e9ed277b444770c2af75f5209afd00bd07c72cc0b \
    --tokens 5 --desc "Send"

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id ab549edb7cea822dab0b460f65dcde85f698c1e97d730e3ffc6b0f8b576b65bd \
    --tokens 5 --desc "Send"

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id 86cf791f03f01e3e4d318b1ca009a51c91dd43f7cf3c87a32f531b609cc5044b \
    --tokens 5 --desc "Send"

./zwalletcli/zwallet --wallet testing.json send \
    --to_client_id 823cb45de27dfe739b320dcf6449e5fdea35c60804fd81d6f22c005042cfb337 \
    --tokens 5 --desc "Send"
