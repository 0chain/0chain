#!/bin/sh
set -e

docker buildx build --platform linux/amd64 -f docker.local/build.base/Dockerfile.build_base.apple . -t zchain_build_base
docker buildx build --platform linux/amd64 -f docker.local/build.base/Dockerfile.run_base.apple docker.local/build.base -t zchain_run_base
