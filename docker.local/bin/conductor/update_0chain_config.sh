#!/bin/bash

set -e

key=$1
value=$2

echo "Updating 0chain config; param $key value $value";

sed -i.bak "s/$key: [0-9]*/$key: $value/g" docker.local/config/0chain.yaml