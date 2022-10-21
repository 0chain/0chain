#!/bin/sh

set -e

for running in $(docker ps -q)
do
    docker stop "$running"
done

./docker.local/bin/clean.sh

if [ $# -eq 0 ]; then
    echo "No test files names provided"
    exit 1
fi

for t in $@; do 
    tests="${tests} ./docker.local/config/conductor.${t}.yaml"; 
done

# go caches all build by default
(cd ./code/go/0chain.net/conductor/conductor/ && go build)
# start the conductor
./code/go/0chain.net/conductor/conductor/conductor                     \
    -config "./docker.local/config/conductor.config.yaml"              \
    -tests "${tests}"
