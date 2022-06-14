#!/bin/sh

cmd="build"

docker $cmd -f docker.local/build.benchmarks/Dockerfile . -t zchain_benchmarks

