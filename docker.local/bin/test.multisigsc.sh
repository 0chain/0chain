#!/bin/sh
set -e

cmd="build"

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        shift
        ;;
    esac
done

docker $cmd -f docker.local/build.test.multisigsc/Dockerfile . -t zchain_test_multisigsc

docker run \
    -it \
    --network testnet0 \
    --mount type=bind,src=$PWD/docker.local/config,dst=/0chain/config,readonly \
    zchain_test_multisigsc \
    ./bin/test_multisigsc
