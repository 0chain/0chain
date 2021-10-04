#!/bin/bash
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo "$GIT_COMMIT"

ROOT="$(git rev-parse --show-toplevel)"
DOCKER_DIR="$ROOT/docker.local/build.miner"
DOCKER_FILE="$DOCKER_DIR/Dockerfile"
DOCKERCOMPOSE="$DOCKER_DIR/docker-compose.yml"

cmd="build"

if [[ "$*" == *"--dev"* ]]
then
    echo -e "\nDevelopment mode: building miner locally\n"

    cd "$ROOT/code/go/0chain.net/miner/miner"
    go build -v -tags "bn256 development" \
        -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

    sed 's,%COPY%,COPY ./code,g' "$DOCKER_FILE.template" > "$DOCKER_FILE"

    cd "$ROOT"
    docker $cmd --build-arg GIT_COMMIT=$GIT_COMMIT \
        -f "$DOCKER_FILE" . -t miner --build-arg DEV=yes
else
    echo -e "\nProduction mode: building miner in Docker\n"

    sed 's,%COPY%,COPY --from=miner_build $APP_DIR,g' "$DOCKER_FILE.template" > "$DOCKER_FILE"

    cd "$ROOT"

    docker $cmd --build-arg GIT_COMMIT="$GIT_COMMIT" \
        -f "$DOCKER_FILE" . -t miner --build-arg DEV=no
fi

for i in $(seq 1 5);
do
  MINER=$i docker-compose -p miner$i -f $DOCKERCOMPOSE build --force-rm
done

docker.local/bin/sync_clock.sh
