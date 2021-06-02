#!/bin/sh
set -e

docker build -f docker.local/build.sc_unit_test/Dockerfile . -t zchain_sc_unit_test

# Use with
# - 0chain.net/smartcontract/storagesc
# or
# - 0chain/code/go/0chain.net/smartcontract/minersc
# arguments
if [ -n "$1" ]; then
    # Assume that $1 looks something like: 0chain.net/chaincore/threshold/bls
    docker run -it zchain_sc_unit_test sh -c "cd $1; go test -v -cover -tags bn256"
fi
