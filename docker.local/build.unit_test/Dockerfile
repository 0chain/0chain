FROM zchain_build_base
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/go.mod $SRC_DIR/go/0chain.net/
COPY ./code/go/0chain.net/go.sum $SRC_DIR/go/0chain.net/

RUN cd $SRC_DIR/go/0chain.net && go mod download -x

RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

COPY ./code/go/0chain.net $SRC_DIR/code/go/0chain.net

WORKDIR $SRC_DIR/code/go