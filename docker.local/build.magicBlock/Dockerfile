FROM zchain_build_base
ENV GO111MODULE=on

ENV BUILD_DIR=/0chain/go/src/magicBlock
ENV CONFIG_DIR=/0chain/config
ENV GOPATH=$GOPATH:$SRC_DIR/go

COPY ./code/go/0chain.net/chaincore/block/magicBlock $BUILD_DIR
COPY ./docker.local/config $CONFIG_DIR

WORKDIR $BUILD_DIR
RUN go mod download
RUN go build -tags bn256 main.go

