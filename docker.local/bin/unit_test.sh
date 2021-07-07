#!/bin/bash
set -e

# Allocate interactive TTY to allow Ctrl-C.
INTERACTIVE="-it"
PACKAGE="$1"

if [[ "$1" == *"--ci"* ]]
then
    # We need non-interactive mode for CI
    INTERACTIVE=""
    echo "Building both general and SC test images"
    docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test
else
    echo "Building general test image"
    docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test
fi

docker run -v `pwd`/code/go/0chain.net:/0chain/code/go/0chain.net zchain_unit_test sh -c "sh /0chain/generate_mocks.sh"


if [[ -n "$PACKAGE" ]]; then
#     Run tests from a single package.
#     assume that $PACKAGE looks something like: 0chain.net/chaincore/threshold/bls
    echo "Running unit tests from $PACKAGE:"
    docker run -v `pwd`/code/go/0chain.net:/0chain/code/go/0chain.net zchain_unit_test sh -c "cd $PACKAGE; go test -tags bn256 -cover ./..."
else
    # Run all tests.
    echo "Running general unit tests:"
    docker run -v `pwd`/code/go/0chain.net:/0chain/code/go/0chain.net zchain_unit_test sh -c "cd 0chain.net; go test -tags bn256 -cover ./..."
fi
