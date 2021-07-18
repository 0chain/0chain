#!/bin/bash

root=$(pwd)

cd ../code/go/0chain.net

code=$(pwd)


# Build libzstd with local repo
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
find . -name "go.mod" -exec sed -i '' "/replace github.com\/valyala\/gozstd/d" {} \;
echo "replace github.com/valyala/gozstd => ../../../../../valyala/gozstd" >> ./go.mod


# Set workdir
cd ./sharder/sharder

# Build bls with CGO_LDFLAGS and CGO_CPPFLAGS to fix `ld: library not found for -lcrypto`
export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib"
export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include"

GIT_COMMIT=$GIT_COMMIT
go build -o $root/data/sharder/sharder -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"



#       - ../config:/0chain/config
#       - ../sharder${SHARDER}/log:/0chain/log
#       - ../sharder${SHARDER}/data:/0chain/data
#     ports:
#       - "717${SHARDER}:717${SHARDER}"
#     networks:
#       default:
#       testnet0:
#         ipv4_address: 198.18.0.8${SHARDER}
#     command: ./bin/sharder --deployment_mode 0 --keys_file config/b0snode${SHARDER}_keys.txt --minio_file config/minio_config.txt
