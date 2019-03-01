#!/bin/sh
set -e

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

# Allocate interactive TTY to allow Ctrl-C.
if [[ -n "$1" ]]; then
    # Run tests from a single package.
    # Assume that $1 looks something like: 0chain.net/chaincore/threshold/bls
    docker run -it zchain_unit_test sh -c "cd $1; go test -tags bn256"
else
    # Run all tests.
    docker run -it zchain_unit_test sh -c '
        for mod_file in $(find * -name go.mod); do
            mod_dir=$(dirname $mod_file)
            (cd $mod_dir; go test -tags bn256 $mod_dir/...)
        done
    '
fi
