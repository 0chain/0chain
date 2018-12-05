#!/bin/sh
docker build -f docker.local/build.base/Dockerfile docker.local/build.base -t zchain_base $*
