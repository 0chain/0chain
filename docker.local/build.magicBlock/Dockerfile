FROM zchain_build_base as magic_block_build
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

COPY ./code/go/0chain.net/core/go.mod          ./code/go/0chain.net/core/go.sum          $SRC_DIR/go/0chain.net/core/
COPY ./code/go/0chain.net/chaincore/go.mod     ./code/go/0chain.net/chaincore/go.sum     $SRC_DIR/go/0chain.net/chaincore/
COPY ./code/go/0chain.net/smartcontract/go.mod ./code/go/0chain.net/smartcontract/go.sum $SRC_DIR/go/0chain.net/smartcontract/
COPY ./code/go/0chain.net/conductor/go.mod     ./code/go/0chain.net/conductor/go.sum     $SRC_DIR/go/0chain.net/conductor/
WORKDIR $SRC_DIR/go/0chain.net/chaincore/block/magicBlock
RUN go mod download

RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

RUN go build -tags bn256 main.go yaml.go

FROM zchain_run_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=magic_block_build $APP_DIR/go/0chain.net/chaincore/block/magicBlock/main $APP_DIR/bin/magicBlock
