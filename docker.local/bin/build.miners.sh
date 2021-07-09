#!/bin/bash
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo "$GIT_COMMIT"

ROOT="$(git rev-parse --show-toplevel)"
DOCKER_DIR="$ROOT/docker.local/build.miner"
DOCKER_FILE="$DOCKER_DIR/Dockerfile"
DOCKERCOMPOSE="$DOCKER_DIR/docker-compose.yml"

APP_ROOT="$ROOT/code/go/0chain.net"

if [[ "$@" == *"--dev"* ]]
then
    cd $APP_ROOT
    echo "Building: --dev mode: vendoring dependencies"
    rm -rf vendor
    go mod vendor
    # libzstd: start: to rebuild inside container
    dstdir="vendor/github.com/valyala"
    rm -r $dstdir/*
    srcdir="$GOPATH/pkg/mod/github.com/valyala"
    cp -r "$srcdir/$(ls $srcdir | tail -n1)" $dstdir/gozstd
    chmod -R +w vendor
    # libzstd: end
    cd $ROOT
fi

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f $DOCKERFILE -t miner .

if [[ "$@" == *"--dev"* ]]
then
    echo "Build complete: cleaning vendored dependencies"
    cd $APP_DIR
    rm -rf vendor
    cd $ROOT
fi

for i in $(seq 1 5);
do
  MINER=$i docker-compose -p miner$i -f $DOCKERCOMPOSE build --force-rm
done

docker.local/bin/sync_clock.sh
