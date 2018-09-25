#!/bin/sh
docker build -f docker.local/build.go_test/Dockerfile code/go/src -t zchain_go_test

# Allocate interactive TTY to allow Ctrl-C.
# Use "-count 1" to disable caching since we're just going to throw the container away after.
docker run -it zchain_go_test go test -count 1 "${@:-./...}"
