#!/bin/sh

set -e

# go caches all build by default
go build code/go/0chain.net/conductor/conductor
# start
./code/go/0chain.net/conductor/conductor/conductor \
    -config ./docker.local/config/conductor.yaml
