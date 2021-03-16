FROM zchain_build_base
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/core/go.mod            ./code/go/0chain.net/core/go.sum            $SRC_DIR/go/0chain.net/core/
COPY ./code/go/0chain.net/chaincore/go.mod       ./code/go/0chain.net/chaincore/go.sum       $SRC_DIR/go/0chain.net/chaincore/
COPY ./code/go/0chain.net/smartcontract/go.mod   ./code/go/0chain.net/smartcontract/go.sum   $SRC_DIR/go/0chain.net/smartcontract/
COPY ./code/go/0chain.net/miner/go.mod           ./code/go/0chain.net/miner/go.sum           $SRC_DIR/go/0chain.net/miner/
COPY ./code/go/0chain.net/miner/miner/go.mod     ./code/go/0chain.net/miner/miner/go.sum     $SRC_DIR/go/0chain.net/miner/miner/
COPY ./code/go/0chain.net/sharder/go.mod         ./code/go/0chain.net/sharder/go.sum         $SRC_DIR/go/0chain.net/sharder/
COPY ./code/go/0chain.net/sharder/sharder/go.mod ./code/go/0chain.net/sharder/sharder/go.sum $SRC_DIR/go/0chain.net/sharder/sharder/
COPY ./code/go/0chain.net/conductor/go.mod       ./code/go/0chain.net/conductor/go.sum       $SRC_DIR/go/0chain.net/conductor/
COPY ./code/go/0chain.net/mocks/go.mod           ./code/go/0chain.net/mocks/go.sum           $SRC_DIR/go/0chain.net/mocks/

RUN cd $SRC_DIR/go/0chain.net/core && \
    go mod download && \
    cd $SRC_DIR/go/0chain.net/chaincore && \
    go mod download && \
    cd $SRC_DIR/go/0chain.net/smartcontract && \
    go mod download && \
    cd $SRC_DIR/go/0chain.net/miner && \
    go mod download && \
    cd $SRC_DIR/go/0chain.net/miner/miner && \
    go mod download && \
    cd $SRC_DIR/go/0chain.net/sharder && \
    go mod download && \
    cd $SRC_DIR/go/0chain.net/sharder/sharder && \
    go mod download && \
    cd $SRC_DIR/go/0chain.net/mocks && \
    go mod download

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?
RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

RUN mkdir /tmp/mockery && \
    cd /tmp/mockery && \
    wget -O - https://github.com/vektra/mockery/releases/download/v2.5.1/mockery_2.5.1_Linux_x86_64.tar.gz | tar xz && \
    cp mockery /bin && \
    rm -r /tmp/mockery

RUN cd $SRC_DIR/go/0chain.net/core && \
    go generate ./...

WORKDIR $SRC_DIR/go
