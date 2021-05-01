#!/bin/sh

# Runs each unit test in batches corresponding to each subdirectory
# of code/go/0chain.net.
# Returns 0 if all of the tests pass and 1 if any one of the tests fail.

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test
docker run $INTERACTIVE zchain_unit_test sh -c "cd 0chain.net; go test -tags bn256 -cover ./..."

