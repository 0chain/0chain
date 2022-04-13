FROM zchain_build_base
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

RUN cd $SRC_DIR/go/0chain.net && \
    go mod download

RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

WORKDIR $SRC_DIR/go
