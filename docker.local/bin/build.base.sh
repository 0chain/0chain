#!/bin/sh
set -e

# If --help is passed, print usage and exit.
if [[ "$*" == *"--help"* ]]
then
    echo "Usage: $0 [--help]" 
    echo "Builds base image for building miners and sharders. Need to be run from the root of the repository."
    echo "  --help: print this help message"
    exit 0
fi

cmd="build"
build_dockerfile="docker.local/build.base/Dockerfile.build_base"
run_dockerfile="docker.local/build.base/Dockerfile.run_base"

docker $cmd -f $build_dockerfile . -t zchain_build_base
docker $cmd -f $run_dockerfile docker.local/build.base -t zchain_run_base
