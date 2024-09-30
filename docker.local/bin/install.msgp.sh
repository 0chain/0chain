#!/bin/bash
set -e

# Install msgp
go install github.com/tinylib/msgp@latest
cd $(go env GOPATH)/src/github.com/0chain/msgp
make install
