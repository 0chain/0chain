#!/bin/sh

cmd="build"

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        shift
        ;;
    esac
done

docker $cmd -f docker.local/build.magicBlock/Dockerfile . -t magicblock
docker-compose -p magic_block -f docker.local/build.magicBlock/docker-compose.yml build --force-rm