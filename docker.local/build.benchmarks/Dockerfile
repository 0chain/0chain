FROM zchain_build_base as base

ENV ROOT=/0chain
ENV SRC_DIR=$ROOT/code/go/0chain.net
ENV GO111MODULE=on

COPY ./code/go/0chain.net/go.mod $SRC_DIR/
COPY ./code/go/0chain.net/go.sum $SRC_DIR/

RUN cd $SRC_DIR && \
    go mod download

COPY ./code/go/0chain.net $SRC_DIR

WORKDIR $SRC_DIR/smartcontract/benchmark/main

RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

RUN cd $SRC_DIR/smartcontract/benchmark/main && go build -tags bn256 main.go

FROM zchain_run_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=base $APP_DIR/go/0chain.net/smartcontract/benchmark/main $APP_DIR/bin/benchmark
