#!/bin/sh
set -e

# Allocate interactive TTY to allow Ctrl-C.
INTERACTIVE="-it"
PACKAGE=""

if [ "$1" = "--ci" ]
then
    # But we need non-interactive mode for CI
    INTERACTIVE=""
else
    PACKAGE="$1"
fi

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

if [ -n "$PACKAGE" ]; then
    # Run tests from a single package.
    # Assume that $PACKAGE looks something like: 0chain.net/chaincore/threshold/bls
    docker run $INTERACTIVE zchain_unit_test sh -c "cd $PACKAGE; go test -tags bn256"
else
    # Run all tests.
    docker run $INTERACTIVE zchain_unit_test sh -c '
        for mod_file in $(find * -name go.mod); do
            mod_dir=$(dirname $mod_file)
            (cd $mod_dir; go test -tags bn256 $mod_dir/...)
        done
    '
fi
