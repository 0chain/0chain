# Begin minimal miner image:
FROM alpine:3.8 as run_env
RUN apk add --update --no-cache bash
RUN apk add --update --no-cache zlib bzip2-dev snappy lz4-dev jemalloc zstd
RUN apk add --update --no-cache gmp openssl
WORKDIR /usr/local/lib
COPY --from=zchain_base /usr/local/lib/libgflags.so.*.*.* /usr/local/lib/librocksdb.so.*.*.* ./
COPY --from=zchain_base /usr/local/lib/libmcl.so /usr/local/lib/libbls*.so ./
RUN ln -s libgflags.so* $(ls libgflags.so* | cut -f1-4 -d.)
RUN ln -s librocksdb.so* $(ls librocksdb.so* | cut -f1-4 -d.)

# Compile the miner in a different image so as not to bloat the minimal:
FROM zchain_base as build_env
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/core/go.mod          ./code/go/0chain.net/core/go.sum          $SRC_DIR/go/0chain.net/core/
COPY ./code/go/0chain.net/chaincore/go.mod     ./code/go/0chain.net/chaincore/go.sum     $SRC_DIR/go/0chain.net/chaincore/
COPY ./code/go/0chain.net/smartcontract/go.mod ./code/go/0chain.net/smartcontract/go.sum $SRC_DIR/go/0chain.net/smartcontract/
COPY ./code/go/0chain.net/miner/go.mod         ./code/go/0chain.net/miner/go.sum         $SRC_DIR/go/0chain.net/miner/
COPY ./code/go/0chain.net/miner/miner/go.mod   ./code/go/0chain.net/miner/miner/go.sum   $SRC_DIR/go/0chain.net/miner/miner/
WORKDIR $SRC_DIR/go/0chain.net/miner/miner
RUN go mod download

# Build libzstd:
# FIXME: Can we move this to zchain_base?
RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    make clean libzstd.a

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

# Build it:
RUN go build -v -tags bn256

# End with minimal image size:
FROM run_env
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=build_env $APP_DIR/go/0chain.net/miner/miner/miner $APP_DIR/bin/
