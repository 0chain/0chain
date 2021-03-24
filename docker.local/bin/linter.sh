#!/bin/sh
set -e

docker build -f docker.local/build.lint_test/Dockerfile . -t zchain_lint_test

docker run zchain_lint_test sh -c '
    for mod_file in $(find * -name go.mod); do
        mod_dir=$(dirname $mod_file)
        (cd $mod_dir; go mod download; golangci-lint run)
    done
'
