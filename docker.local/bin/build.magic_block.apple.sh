#!/bin/sh
docker buildx build --platform linux/amd64 -f docker.local/build.magicBlock/Dockerfile . -t magicblock
docker-compose -p magic_block -f docker.local/build.magicBlock/docker-compose.yml build --force-rm