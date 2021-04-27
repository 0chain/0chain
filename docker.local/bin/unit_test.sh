#!/bin/bash
set -e

# Allocate interactive TTY to allow Ctrl-C.
INTERACTIVE="-it"
PACKAGE=""

if [[ "$1" == *"--ci"* ]]
then
    # We need non-interactive mode for CI
    INTERACTIVE=""
    echo "Building both general and SC test images"
    docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test
else
    PACKAGE="$1"

    echo "Building general test image"
    docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test
fi

if [[ -n "$PACKAGE" ]]; then
    # Run tests from a single package.
    # assume that $PACKAGE looks something like: 0chain.net/chaincore/threshold/bls
    echo "Running unit tests from $PACKAGE:"
    docker run $INTERACTIVE zchain_unit_test sh -c "cd /0chain/go/$PACKAGE; go test -tags bn256 -cover ./..."
else
    # Run all tests.
    echo "Running general unit tests:"
    docker run $INTERACTIVE zchain_unit_test sh -c "cd 0chain.net; go test -tags bn256 -cover ./..."
fi
