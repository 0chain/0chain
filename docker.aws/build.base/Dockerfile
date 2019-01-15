FROM iron/go:dev
RUN apk add --update --no-cache build-base linux-headers git cmake bash perl grep
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev jemalloc jemalloc-dev zstd
RUN apk add --update --no-cache gmp-dev openssl-dev
RUN apk add --update --no-cache autoconf automake bison flex libtool

# Install golang dep
RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && \
    chmod +x /usr/local/bin/dep

# Install latest gflags
RUN cd /tmp && \
    curl -L -o gflags.tar.gz https://github.com/gflags/gflags/archive/v2.2.1.tar.gz && \
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
    #git checkout 5.13.fb && \
    make  -j $(nproc) shared_lib && \
    mkdir -p /usr/local/rocksdb/lib && \
    mkdir /usr/local/rocksdb/include && \
    cp librocksdb.so* /usr/local/rocksdb/lib && \
    cp /usr/local/rocksdb/lib/librocksdb.so* /usr/lib/ && \
    cp -r include /usr/local/rocksdb/ && \
    cp -r include/* /usr/include/ && \
    rm -R /tmp/rocksdb/

#Download the go dependencies
RUN go get github.com/golang/snappy
RUN go get -d github.com/valyala/gozstd && \
    cd $GOPATH/src/github.com/valyala/gozstd && \
    make clean libzstd.a
RUN go get github.com/gomodule/redigo/redis
RUN go get github.com/0chain/gorocksdb
RUN go get github.com/vmihailenco/msgpack
RUN go get -u golang.org/x/crypto/...
RUN go get go.uber.org/zap
RUN go get github.com/rcrowley/go-metrics
RUN go get github.com/spf13/viper
RUN go get gopkg.in/natefinch/lumberjack.v2
RUN go get github.com/koding/cache
RUN go get github.com/hashicorp/golang-lru

RUN go get -d github.com/herumi/mcl/ffi/go/mcl && \
    cd $GOPATH/src/github.com/herumi/mcl && \
    CFLAGS='-DMCLBN_FP_UNIT_SIZE=4' make && \
    cp $GOPATH/src/github.com/herumi/mcl/lib/lib*.so /usr/lib && \
    CGO_CFLAGS="-I$GOPATH/src/github.com/herumi/mcl/include" CGO_LDFLAGS="-L$GOPATH/src/github.com/herumi/mcl/lib" go get -tags bn256 github.com/herumi/mcl/ffi/go/mcl
    
RUN go get -d github.com/herumi/bls/ffi/go/bls && \
    cd $GOPATH/src/github.com/herumi/bls && \
    CFLAGS='-DMCLBN_FP_UNIT_SIZE=4' make && \
    cp $GOPATH/src/github.com/herumi/bls/lib/lib*.so /usr/lib && \
    CGO_CFLAGS="-I$GOPATH/src/github.com/herumi/bls/include -I$GOPATH/src/github.com/herumi/mcl/include" CGO_LDFLAGS="-L$GOPATH/src/github.com/herumi/bls/lib -L$GOPATH/src/github.com/herumi/mcl/lib" go get -tags bn256 github.com/herumi/bls/ffi/go/bls


RUN cd /tmp && \
    git clone https://github.com/blynn/pbc.git && \
    cd pbc && \
    mkdir m4 && \
    autoreconf -i && \
    ./configure && \
    make && \
    make install

RUN go get github.com/Nik-U/pbc
