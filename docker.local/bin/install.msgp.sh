#!/bin/bash
set -e

# Install msgp
GO111MODULE=auto go get -u github.com/0chain/msgp
cd $(go env GOPATH)/src/github.com/0chain/msgp
make install
