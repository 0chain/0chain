FROM iron/go:dev
RUN apk add --update --no-cache build-base linux-headers git cmake bash perl grep
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev jemalloc jemalloc-dev zstd

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
    make  shared_lib && \
    mkdir -p /usr/local/rocksdb/lib && \
    mkdir /usr/local/rocksdb/include && \
    cp librocksdb.so* /usr/local/rocksdb/lib && \
    cp /usr/local/rocksdb/lib/librocksdb.so* /usr/lib/ && \
    cp -r include /usr/local/rocksdb/ && \
    cp -r include/* /usr/include/ && \
    rm -R /tmp/rocksdb/


# Install Mitsunari Shigeo's BLS threshold crypto lib
# It has a homebrew build system that requires all four codebases be in the same dir.
RUN apk --update --no-cache add gmp-dev openssl-dev
RUN printf "#!/bin/sh\necho x86_64\n" > /usr/local/bin/arch && \
    chmod +x /usr/local/bin/arch && \
    cd /tmp && \
    curl -L -o mcl.tar.gz https://github.com/herumi/mcl/archive/new_deserialize_api.tar.gz && \
    curl -L -o cybozulib.tar.gz https://github.com/herumi/cybozulib/archive/release20180830.tar.gz && \
    curl -L -o xbyak.tar.gz https://github.com/herumi/xbyak/archive/v5.73.tar.gz && \
    tar xf mcl.tar.gz && mv mcl-* mcl && \
    tar xf cybozulib.tar.gz && mv cybozulib-* cybozulib && \
    tar xf xbyak.tar.gz && mv xbyak-* xbyak && \
    cd /tmp/mcl && \
      CFLAGS='-DMCLBN_FP_UNIT_SIZE=4' make lib/libmcl.a && \
      cp lib/*.a /usr/local/lib && \
      cp -r include/mcl /usr/local/include/mcl && \
    cd /tmp && \
    curl -L -o bls.tar.gz https://github.com/herumi/bls/archive/new_deserialize_api.tar.gz && \
    tar xf bls.tar.gz && mv bls-* bls && \
    cd /tmp/bls && \
      make UNIT=4 obj/bls.o obj/bls_c.o && \
      ar -r lib/libbls.a obj/bls.o obj/bls_c.o && \
      cp lib/libbls.a /usr/local/lib && \
      mkdir /usr/local/include/bls && \
      cp include/bls/*.h /usr/local/include/bls && \
    rm -R /tmp/bls* && \
    rm -R /tmp/cybozulib* && \
    rm -R /tmp/mcl* && \
    rm -R /tmp/xbyak* && \
    rm /usr/local/bin/arch

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


ARG REBUILD_VER=unknown

RUN go get -d github.com/herumi/mcl/ffi/go/mcl && \
    cd $GOPATH/src/github.com/herumi/mcl && \
    CGO_CFLAGS='-DMCLBN_FP_UNIT_SIZE=4' make && \
    cp $GOPATH/src/github.com/herumi/mcl/lib/lib*.so /usr/lib && \
    go get -tags bn256 github.com/herumi/mcl/ffi/go/mcl

RUN go get -d github.com/herumi/bls/ffi/go/bls && \
    cd $GOPATH/src/github.com/herumi/bls && \
    CFLAGS='-DMCLBN_FP_UNIT_SIZE=4' make && \
    cp $GOPATH/src/github.com/herumi/bls/lib/lib*.so /usr/lib && \
    sed -i 's/MCLBN_FP_UNIT_SIZE=6/MCLBN_FP_UNIT_SIZE=4/' ffi/go/bls/mcl.go && \
    sed -i 's/MCLBN_FP_UNIT_SIZE=6/MCLBN_FP_UNIT_SIZE=4/' ffi/go/bls/bls.go && \
    sed -i 's/-lbls384_dy/-lbls256_dy/' ffi/go/bls/bls.go && \
    go get -tags bn256 github.com/herumi/bls/ffi/go/bls