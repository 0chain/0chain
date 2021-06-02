FROM zchain_build_base
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net $SRC_DIR/code/go/0chain.net
COPY ./generate_mocks.sh $SRC_DIR/generate_mocks.sh

RUN go get github.com/vektra/mockery/v2/.../
RUN cd $SRC_DIR && ls && ./generate_mocks.sh

RUN cd $SRC_DIR/code/go/0chain.net && \
    go mod download

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?

RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd@v1.5.* && \
    chmod -R +w . && \
    make clean libzstd.a

WORKDIR $SRC_DIR/code/go
