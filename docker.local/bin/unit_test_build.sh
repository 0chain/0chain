#!/bin/bash
set -e

# generate mocks
make install-mockery
make build-mocks

cmd="build"
dockerfile="docker.local/build.unit_test/Dockerfile"
platform=""

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        dockerfile="docker.local/build.unit_test/Dockerfile.m1"
        platform="--platform=linux/amd64"
        shift
        ;;
    esac
done


# Runs each unit test in batches corresponding to each subdirectory
# of code/go/0chain.net.
# Returns 0 if all of the tests pass and 1 if any one of the tests fail.
docker $cmd -f $dockerfile . -t zchain_unit_test
docker run --add-host=host.docker.internal:host-gateway -v /var/run/docker.sock:/var/run/docker.sock \
	-e DOCKER_HOST_ENV=host.docker.internal:host-gateway \
	$platform $INTERACTIVE -v $(pwd)/code:/codecov  \
	zchain_unit_test sh -c "cd 0chain.net; go test -tags bn256 -coverprofile=/codecov/coverage.txt -covermode=atomic ./..."
