#!/bin/sh

docker stop $(docker ps -q)
set -e


./docker.local/bin/clean.sh

if [ $# -eq 0 ]; then
    echo "No test files names provided"
    exit 1
fi

for t in $@; do 
    tests="${tests} ./docker.local/config/conductor.${t}.yaml"; 
done

# go caches all build by default
(cd ./code/go/0chain.net/conductor/conductor/ && go build -tags "bn256")
# start the conductor
./code/go/0chain.net/conductor/conductor/conductor                     \
    -config "./docker.local/config/conductor.config.yaml"              \
    -tests "${tests}"
