#!/bin/sh
set -e

# generate mocks
make build-mocks

cmd="build"

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        shift
        ;;
    esac
done

GIT_COMMIT=$(git rev-list -1 HEAD)
echo "$GIT_COMMIT"

docker $cmd --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/build.miner/Dockerfile.integration_tests . -t miner

for i in $(seq 1 8);
do
  MINER=$i docker-compose -p miner"$i" -f docker.local/build.miner/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
