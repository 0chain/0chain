#!/bin/bash

set -e

key=$1
value=$2

echo "Updating 0chain config; param $key value $value";

sed -i "s/$key: */$key: $value/g" docker.local/config/0chain.yaml