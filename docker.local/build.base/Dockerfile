# docker.local
FROM golang:1.11.4-alpine3.8
RUN apk add --update --no-cache build-base linux-headers git cmake bash perl grep
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev jemalloc jemalloc-dev zstd
RUN apk add --update --no-cache gmp-dev openssl-dev
RUN apk add --update --no-cache autoconf automake bison flex libtool

# Install latest gflags
RUN cd /tmp && \
    wget -O gflags.tar.gz https://github.com/gflags/gflags/archive/v2.2.1.tar.gz && \
    tar xf gflags.tar.gz && \
    cd gflags-* && \
    mkdir build && \
    cd build && \
    cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
    make install -j $(nproc) && \
    cd /tmp && \
    rm -R /tmp/gflags*

# Install RocksDB
RUN cd /tmp && \
    git clone https://github.com/0chain/rocksdb.git && \
    cd rocksdb && \
    make  -j $(nproc) shared_lib && \
    cp -P librocksdb.so* /usr/local/lib && \
    cp -r include/* /usr/local/include/ && \
    rm -R /tmp/rocksdb/

ENV CGO_CFLAGS="-I$GOPATH/src/github.com/herumi/bls/include -I$GOPATH/src/github.com/herumi/mcl/include"
ENV CGO_LDFLAGS="-L$GOPATH/src/github.com/herumi/bls/lib -L$GOPATH/src/github.com/herumi/mcl/lib"

RUN go get -d github.com/herumi/mcl/ffi/go/mcl && \
    cd $GOPATH/src/github.com/herumi/mcl && \
    CFLAGS='-DMCLBN_FP_UNIT_SIZE=4' make && \
    cp $GOPATH/src/github.com/herumi/mcl/lib/lib*.so /usr/local/lib

RUN go get -d github.com/herumi/bls/ffi/go/bls && \
    cd $GOPATH/src/github.com/herumi/bls && \
    CFLAGS='-DMCLBN_FP_UNIT_SIZE=4' make && \
    cp $GOPATH/src/github.com/herumi/bls/lib/lib*.so /usr/local/lib