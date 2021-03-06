#!/bin/bash
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

ROOT="$(git rev-parse --show-toplevel)"
DOCKERDIR="$ROOT/docker.local/build.miner"
DOCKERFILE="$DOCKERDIR/Dockerfile"
DOCKERCOMPOSE="$DOCKERDIR/docker-compose.yml"

if [[ "$@" == *"--dev"* ]]
then
    echo -e "\nDevelopment mode: building miner locally\n"

    cd $ROOT/code/go/0chain.net/miner/miner
    go build -v -tags "bn256 development" \
        -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

    sed 's,%COPY%,COPY ./code,g' $DOCKERFILE.template > $DOCKERFILE

    cd $ROOT
    docker build --build-arg GIT_COMMIT=$GIT_COMMIT \
        -f $DOCKERFILE . -t miner --build-arg DEV=yes
else
    echo -e "\nProduction mode: building miner in Docker\n"

    sed 's,%COPY%,COPY --from=miner_build $APP_DIR,g' $DOCKERFILE.template > $DOCKERFILE

    cd $ROOT
    docker build --build-arg GIT_COMMIT=$GIT_COMMIT \
        -f $DOCKERFILE . -t miner --build-arg DEV=no
fi

for i in $(seq 1 5);
do
    MINER=$i docker-compose -p miner$i -f $DOCKERCOMPOSE build --force-rm
done

$ROOT/docker.local/bin/sync_clock.sh
