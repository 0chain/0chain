FROM iron/go:dev
RUN apk add --update --no-cache build-base linux-headers git cmake bash perl
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev jemalloc jemalloc-dev zstd

# installing latest gflags 
RUN cd /tmp && \
    git clone https://github.com/gflags/gflags.git && \
    cd gflags && \
    mkdir build && \
    cd build && \
    cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
    make install && \
    cd /tmp && \
    rm -R /tmp/gflags/

# Install Rocksdb 
RUN cd /tmp && \
    git clone https://github.com/0chain/rocksdb.git && \
    cd rocksdb && \
    make -j8 shared_lib && \
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

ADD VERSION .
