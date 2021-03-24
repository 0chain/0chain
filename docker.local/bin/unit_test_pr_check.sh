#!/bin/sh
set -e

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

docker run zchain_unit_test sh -c "'
    for mod_file in $(find * -name go.mod); do
        mod_dir=$(dirname $mod_file)
        (cd $mod_dir; go test -tags bn256 $mod_dir/...)
    done
    '"
echo "finished all unit tests"
