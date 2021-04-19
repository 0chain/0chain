#!/bin/bash
set -e

# Allocate interactive TTY to allow Ctrl-C.
INTERACTIVE="-it"
PACKAGE=""
IMAGE=""

GENERAL_IMAGE="zchain_unit_test"
SC_IMAGE="zchain_sc_unit_test"

if [[ "$1" == *"--ci"* ]]
then
    # We need non-interactive mode for CI
    INTERACTIVE=""
    echo "Building both general and SC test images"
    docker build -f docker.local/build.sc_unit_test/Dockerfile . -t zchain_sc_unit_test
    docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test
else
    if [[ "$1" == *"--sc"* ]]
    then
        PACKAGE="$2"
        IMAGE="$SC_IMAGE"
        echo "Building SC test image"
        docker build -f docker.local/build.sc_unit_test/Dockerfile . -t $IMAGE
    else
        PACKAGE="$1"
        IMAGE="$GENERAL_IMAGE"
        echo "Building general test image"
        docker build -f docker.local/build.unit_test/Dockerfile . -t $IMAGE
    fi
fi

GO_TEST="go test -v -cover -tags bn256"

if [[ -n "$PACKAGE" ]]; then
    # Run tests from a single package.
    # assume that $PACKAGE looks something like: 0chain.net/chaincore/threshold/bls
    echo "Running unit tests from $PACKAGE:"
    docker run $INTERACTIVE $IMAGE sh -c "cd $PACKAGE; $GO_TEST"
else
    # Run all tests.
    echo no package run all tests
    # avoid expanding outside of Docker container
    cd_mod='cd 0chain.net/$mod'
    all_in_mod='0chain.net/$mod/...'

    GENERAL_MODS="core \
        chaincore chaincore/block/magicBlock \
        conductor conductor/conductor \
        smartcontract smartcontract/multisigsc/test \
        miner miner/miner \
        sharder sharder/sharder \
    "

    echo "Running general unit tests:"
    docker run $INTERACTIVE $GENERAL_IMAGE sh -c "\
        for mod in $GENERAL_MODS; do \
            $cd_mod; \
            $GO_TEST $all_in_mod; \
            cd -; \
        done \
    "

    echo "Running smart contract unit tests:"
    docker run $INTERACTIVE $SC_IMAGE sh -c "cd 0chain.net/smartcontract/storagesc; $GO_TEST;"
    docker run $INTERACTIVE $SC_IMAGE sh -c "cd 0chain.net/smartcontract/minersc; $GO_TEST;"
fi
