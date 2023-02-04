#!/bin/bash
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo "$GIT_COMMIT"

generate_mocks=1
for arg in "$@"
do
    case $arg in
        --no-mocks)
            generate_mocks=0
        shift
        ;;
    esac
done

# generate mocks
if (( generate_mocks == 1 )); then
    make build-mocks
fi

cmd="build"

# generate swagger
echo "generating swagger.yaml file"
docker.local/bin/test.swagger.sh

docker $cmd --build-arg GIT_COMMIT="$GIT_COMMIT" -f docker.local/build.sharder/Dockerfile . -t sharder

for i in $(seq 1 3);
do
  SHARDER=$i docker-compose -p sharder$i -f docker.local/build.sharder/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
