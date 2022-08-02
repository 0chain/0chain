#!/bin/sh

sed -i 's/rate_limit: 0 # 10/rate_limit: 1000 # 1000/' config/0chain_blobber.yaml

sed -i 's/POSTGRES_USER: postgres/&\n\t\t\tPOSTGRES_PASSWORD: secret/' docker.local/b0docker-compose.yml