#!/bin/bash
set -e

# generate mocks
make build-mocks

cmd="build"
dockerfile="docker.local/build.unit_test/Dockerfile"
platform=""

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        dockerfile="docker.local/build.unit_test/Dockerfile.m1"
        platform="--platform=linux/amd64"
        shift
        ;;
    esac
done

# Allocate interactive TTY to allow Ctrl-C.
INTERACTIVE="-it"
PACKAGE=""

if [[ "$1" == *"--ci"* ]]
then
    # We need non-interactive mode for CI
    INTERACTIVE=""
    echo "Building both general and SC test images"
    docker $cmd -f $dockerfile . -t zchain_unit_test
else
    PACKAGE="$1"

    echo "Building general test image"
    docker $cmd -f $dockerfile . -t zchain_unit_test
fi


if [[ -n "$PACKAGE" ]]; then
    # Run tests from a single package.
    # assume that $PACKAGE looks something like: 0chain.net/chaincore/threshold/bls
    echo "Running unit tests from $PACKAGE:"
    docker run "$INTERACTIVE" zchain_unit_test sh -c "cd /0chain/code/go/$PACKAGE; go test -tags bn256 ./..."
else
    # Run all tests.
    echo "Running general unit tests:"
    docker run "$INTERACTIVE" -v $(pwd)/code:/codecov zchain_unit_test sh -c "cd 0chain.net; go test -tags bn256 -coverprofile=/codecov/coverage.txt -covermode=atomic ./..." 
fi
