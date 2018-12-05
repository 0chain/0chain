#!/bin/sh
docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

# Allocate interactive TTY to allow Ctrl-C.
if [ -n "$1" ]; then
    docker run -it zchain_unit_test go test "$1/..."
else
    docker run -it zchain_unit_test go test "0chain.net/..."
fi