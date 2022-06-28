FROM zchain_build_base as magic_block_build
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

COPY ./code/go/0chain.net/go.mod $SRC_DIR/go/0chain.net/go.mod
COPY ./code/go/0chain.net/go.sum $SRC_DIR/go/0chain.net/go.sum

RUN cd $SRC_DIR/go/0chain.net && \
    go mod download

RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

WORKDIR $SRC_DIR/go/0chain.net/chaincore/block/magicBlock

RUN go build -tags bn256 main.go yaml.go

FROM zchain_run_base
RUN apk add zip
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=magic_block_build $APP_DIR/go/0chain.net/chaincore/block/magicBlock/main $APP_DIR/bin/magicBlock
