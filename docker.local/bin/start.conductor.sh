#!/bin/sh

set -e

# go caches all build by default
(cd ./code/go/0chain.net/conductor/conductor/ && go build)
# start
./code/go/0chain.net/conductor/conductor/conductor \
    -config ./docker.local/config/conductor.yaml
