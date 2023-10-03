#!/bin/sh

cmd="build"

# generate mocks
make install-mockery
make build-mocks

docker $cmd -f docker.local/build.benchmarks/Dockerfile . -t zchain_benchmarks
