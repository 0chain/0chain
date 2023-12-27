#!/bin/sh
set -e

cmd="build"
build_dockerfile="docker.local/build.base/Dockerfile.build_base_go119"
run_dockerfile="docker.local/build.base/Dockerfile.run_base_go119"

docker $cmd -f $build_dockerfile . -t zchain_build_base_go119
docker $cmd -f $run_dockerfile docker.local/build.base -t zchain_run_base_go119
