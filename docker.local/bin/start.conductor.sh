#!/bin/sh

set -e

export GOOS=linux
export GARCH=amd64
export CGO_ENABLED=0

# go caches all build by default
# build conductor executable to run in container
(cd ./code/go/0chain.net/conductor/conductor/ && go build)
# build runner to run on host machine
(cd ./code/go/0chain.net/conductor/runner/ && go build)
# start the conductor in container as daemon
docker-compose -f ./docker.local/conductor/conductor-docker-compose.yml up --build -d
# start the runner
./code/go/0chain.net/conductor/runner/runner \
    -config ./docker.local/config/conductor.yaml
# dirty hack: todo use service tag
docker stop conductor_conductor_1
