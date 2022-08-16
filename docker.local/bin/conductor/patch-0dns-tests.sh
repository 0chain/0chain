#!/bin/sh

sed -i 's/use_https: true/use_https: false/' docker.local/config/0dns.yaml
sed -i 's/use_path: true/use_path: false/' docker.local/config/0dns.yaml
sed -i 's/rate_limit: 5 # 5/rate_limit: 1000 # 1000/' docker.local/config/0dns.yaml