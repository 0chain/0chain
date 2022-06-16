#!/bin/bash

snappy=$(brew --prefix snappy)
lz4=$(brew --prefix lz4)
gmp=$(brew --prefix gmp)
openssl=$(brew --prefix openssl@1.1)

export LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
export LD_LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
export DYLD_LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
export CGO_LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4"
export CGO_CFLAGS="-I/usr/local/include"
export CGO_CPPFLAGS="-I/usr/local/include"
export LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4"
export CFLAGS="-I/usr/local/include"
export CPPFLAGS="-I/usr/local/include"


echo "Start testing..."
cd ../code/go/0chain.net
go test -mod mod -tags "bn256 development dev" ./...
echo "Tests completed."