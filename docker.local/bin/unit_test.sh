#!/bin/sh
set -e

# Allocate interactive TTY to allow Ctrl-C.
INTERACTIVE="-it"
PACKAGE=""
IMAGE=""


GENERAL_IMAGE="zchain_unit_test"
SC_IMAGE="zchain_sc_unit_test"

if [[ "$@" == *"--ci"* ]]
then
    # We need non-interactive mode for CI
    INTERACTIVE=""

    echo "Building both general and SC test images"
    docker build -f docker.local/build.sc_unit_test/Dockerfile . -t zchain_sc_unit_test
    docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test
else
    PACKAGE="$1"

    if [[ "$@" == *"--sc"* ]]
    then
        IMAGE="$SC_IMAGE"
        echo "Building SC test image"
        docker build -f docker.local/build.sc_unit_test/Dockerfile . -t $IMAGE
    else
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
    docker run $INTERACTIVE $IMAGE sh -c "cd /0chain/go/$PACKAGE; go test -tags bn256 -cover ./..."
else
    # Run all tests.

    echo "Running general unit tests:"
    docker run $INTERACTIVE $GENERAL_IMAGE sh -c "cd 0chain.net; go test -tags bn256 -cover ./..."

    echo "Running smart contract unit tests:"
    docker run $INTERACTIVE $SC_IMAGE sh -c "\
        cd 0chain.net/smartcontract/minersc; $GO_TEST; \ cd -;
        cd 0chain.net/smartcontract/storagesc; $GO_TEST \ cd -;
    "
fi
