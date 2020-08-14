#!/bin/sh

set -e

# stop all running container may be still alive after previous tests
docker stop $(docker ps -a -q)

# go caches all build by default
(cd ./code/go/0chain.net/conductor/conductor/ && go build)
# start the conductor
./code/go/0chain.net/conductor/conductor/conductor                     \
    -config "./docker.local/config/conductor.config.yaml"              \
    -tests "./docker.local/config/conductor.${1:-view-change-1}.yaml"
