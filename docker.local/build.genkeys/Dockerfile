FROM zchain_build_base

ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

RUN cd $SRC_DIR/go/0chain.net && \
    go mod download
WORKDIR $SRC_DIR/go/0chain.net/core

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?
RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a
