#!/bin/bash
cd ../../
root=$(pwd)

export LIBRARY_PATH="/usr/local/lib"
export LD_LIBRARY_PATH="/usr/local/lib:/usr/local/opt/openssl@1.1/lib"
export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4"
export CGO_CFLAGS="-I/usr/local/opt/openssl@1.1/include -I/usr/local/include"
export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include -I/usr/local/include"


cd "$root/code/go/0chain.net/sharder/sharder"

CGO_ENABLED=1 go build -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"