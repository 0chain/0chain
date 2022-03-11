#!/bin/bash
set -e

# Install msgp
GO111MODULE=off go get -u github.com/peterlimg/msgp
cd $(go env GOPATH)/src/github.com/peterlimg/msgp
make install
