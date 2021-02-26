#!/bin/sh
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

# Build it locally
TOP="$(git rev-parse --show-toplevel)"
cd $TOP/code/go/0chain.net/miner/miner && go build -v -tags "bn256 development" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"
cd $TOP

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f docker.local/build.miner/Dockerfile.dev . -t miner --build-arg MODE=quick

for i in $(seq 1 5);
do
  MINER=$i docker-compose -p miner$i -f docker.local/build.miner/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
