#!/bin/sh

set -e
docker-compose                                                \
    -f ./docker.local/docker-clean/docker-clean-compose.yml   \
    up                                                        \
    --build --force-recreate docker-clean
