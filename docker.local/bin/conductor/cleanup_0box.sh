#!/bin/bash

set -e

pushd 0box;
./docker.local/bin/clean.sh;
./docker.local/bin/init.sh;
popd;
