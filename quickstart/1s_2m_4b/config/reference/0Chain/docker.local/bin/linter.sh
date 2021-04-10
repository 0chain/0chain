#!/bin/sh
set -e

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

docker run zchain_unit_test sh -c '
    apk add curl
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.38.0
    golangci-lint --version
    for mod_file in $(find * -name go.mod); do
        mod_dir=$(dirname $mod_file)
        (cd $mod_dir; go mod download; golangci-lint run)
    done
'
