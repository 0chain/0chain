#!/bin/sh
set -e

docker build -f docker.local/build.test.multisigsc/Dockerfile . -t zchain_test_multisigsc

docker run \
    -it \
    --network testnet0 \
    --mount type=bind,src=$PWD/docker.local/config,dst=/0chain/config,readonly \
    zchain_test_multisigsc \
    ./bin/test_multisigsc
