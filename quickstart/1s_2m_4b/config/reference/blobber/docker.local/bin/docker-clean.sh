#!/bin/sh

set -e
docker-compose                                                \
    -f ./docker.local/docker-clean/docker-clean-compose.yml   \
    up                                                        \
    --build docker-clean
