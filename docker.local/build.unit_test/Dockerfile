FROM zchain_build_base
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net
COPY ./tmp $GOPATH/pkg/mod

RUN cd $SRC_DIR/go/0chain.net && \
    go mod download

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?
RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

RUN cd $GOPATH/pkg/mod* && \
    chmod -R +w .

WORKDIR $SRC_DIR/go
