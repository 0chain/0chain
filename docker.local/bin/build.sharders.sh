#!/bin/bash
set -e

GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT

ROOT="$(git rev-parse --show-toplevel)"
DOCKERDIR="$ROOT/docker.local/build.sharder"
DOCKERFILE="$DOCKERDIR/Dockerfile"
DOCKERCOMPOSE="$DOCKERDIR/docker-compose.yml"

APP_DIR="$ROOT/code/go/0chain.net/sharder/sharder"

if [[ "$@" == *"--dev"* ]]
then
    cd $APP_DIR
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

docker build --build-arg GIT_COMMIT=$GIT_COMMIT -f $DOCKERFILE -t sharder .

if [[ "$@" == *"--dev"* ]]
then
    echo "Build complete: cleaning vendored dependencies"
    cd $APP_DIR
    rm -rf vendor
    cd $ROOT
fi

for i in $(seq 1 3);
do
  SHARDER=$i docker-compose -p sharder$i -f $DOCKERCOMPOSE build --force-rm
done

docker.local/bin/sync_clock.sh