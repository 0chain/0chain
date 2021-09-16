#!/bin/sh
set -e

cmd="build"
build_dockerfile="docker.local/build.base/Dockerfile.build_base"
run_dockerfile="docker.local/build.base/Dockerfile.run_base"

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        build_dockerfile="docker.local/build.base/Dockerfile.build_base.m1"
        run_dockerfile="docker.local/build.base/Dockerfile.run_base.m1"
        shift
        ;;
    esac
done

docker $cmd -f $build_dockerfile . -t zchain_build_base
docker $cmd -f $run_dockerfile docker.local/build.base -t zchain_run_base
