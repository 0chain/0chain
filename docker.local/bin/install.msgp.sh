#!/bin/bash
set -e

# Install msgp
go install github.com/0chain/msgp@latest
cd $(go env GOPATH)/src/github.com/0chain/msgp
make install
