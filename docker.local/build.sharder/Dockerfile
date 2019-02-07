FROM zchain_base

ENV SRC_DIR=/0chain

#Add the source code
ADD ./code/go $SRC_DIR/go

ENV GO111MODULE=on

WORKDIR $SRC_DIR/go/0chain.net/sharder/sharder

RUN go mod download

# Build it:
RUN cd $SRC_DIR/go/0chain.net/sharder/sharder && \
    go get -d && \
    cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    make clean libzstd.a && \
    cd $SRC_DIR/go/0chain.net/sharder/sharder && \
    go build -v -tags bn256

# Build it:
RUN cd $SRC_DIR/go/0chain.net/sharder/sharder && \
    go build -v -tags bn256

FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=0 $APP_DIR/go/0chain.net/sharder/sharder/sharder $APP_DIR/bin
