#!/bin/sh
set -e

# Allocate interactive TTY to allow Ctrl-C.
INTERACTIVE="-it"
PACKAGE=""

if [[ "$1" == "--ci" ]]
then
    # But we need non-interactive mode for CI
    INTERACTIVE=""
else
    PACKAGE="$1"
fi

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

docker run $INTERACTIVE zchain_unit_test sh -c '
    for mod_file in $(find * -name go.mod); do
        mod_dir=$(dirname $mod_file)
        (cd $mod_dir; go test -tags bn256 $mod_dir/...)
    done

