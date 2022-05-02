#!/bin/sh
set -e

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

docker $cmd -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

#Set this to a value higher than the current number of linter errors. We should lower this number over time
docker run zchain_unit_test bash -c '
    apk add curl
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.44.2
    (cd 0chain.net; golangci-lint run --build-tags bn256 --timeout 10m0s --skip-files '_test.go' --out-format tab)
'
