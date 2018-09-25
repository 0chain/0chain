#!/bin/sh
docker build -f docker.local/build.go_test/Dockerfile code/go/src -t zchain_go_test

# Allocate interactive TTY to allow Ctrl-C.
docker run -it zchain_go_test go test "${@:-./...}"
