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

RUN RUN apk add curl

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.38.0

RUN golangci-lint --version

WORKDIR $SRC_DIR/go
