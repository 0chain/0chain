#!/bin/sh
set -e

docker build -f docker.local/build.base/Dockerfile.build_base . -t zchain_build_base
docker build -f docker.local/build.base/Dockerfile.run_base   docker.local/build.base -t zchain_run_base
