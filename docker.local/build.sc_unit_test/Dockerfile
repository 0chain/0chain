FROM zchain_build_base
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

COPY ./code/go/0chain.net/core/go.mod            ./code/go/0chain.net/core/go.sum            $SRC_DIR/go/0chain.net/core/
COPY ./code/go/0chain.net/chaincore/go.mod       ./code/go/0chain.net/chaincore/go.sum       $SRC_DIR/go/0chain.net/chaincore/
COPY ./code/go/0chain.net/smartcontract/go.mod   ./code/go/0chain.net/smartcontract/go.sum   $SRC_DIR/go/0chain.net/smartcontract/
COPY ./code/go/0chain.net/conductor/go.mod       ./code/go/0chain.net/conductor/go.sum       $SRC_DIR/go/0chain.net/conductor/

RUN cd $SRC_DIR/go/0chain.net/smartcontract && \
    go mod download

RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

WORKDIR $SRC_DIR/go
