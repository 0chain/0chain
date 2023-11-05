#!/bin/sh

sed -i "s/ipv4_address: 198.18.0.98/ipv4_address: 198.18.0.95/" docker.local/docker-compose-ct.yml
sed -i "s/AWS_ACCESS_KEY_ID=key_id/AWS_ACCESS_KEY_ID=$1/" docker.local/docker-compose-ct.yml
sed -i "s/AWS_SECRET_ACCESS_KEY=secret_key/AWS_SECRET_ACCESS_KEY=$2/" docker.local/docker-compose-ct.yml
sed -i "s,block_worker: .*$,block_worker: $3," docker.local/config/0box.yaml