FROM zchain_base

ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/core/go.mod            ./code/go/0chain.net/core/go.sum            $SRC_DIR/go/0chain.net/core/
COPY ./code/go/0chain.net/chaincore/go.mod       ./code/go/0chain.net/chaincore/go.sum       $SRC_DIR/go/0chain.net/chaincore/
COPY ./code/go/0chain.net/smartcontract/go.mod   ./code/go/0chain.net/smartcontract/go.sum   $SRC_DIR/go/0chain.net/smartcontract/
# FIXME: Missing a sharder go.sum file?
COPY ./code/go/0chain.net/sharder/go.mod                                                     $SRC_DIR/go/0chain.net/sharder/
COPY ./code/go/0chain.net/sharder/sharder/go.mod ./code/go/0chain.net/sharder/sharder/go.sum $SRC_DIR/go/0chain.net/sharder/sharder/
WORKDIR $SRC_DIR/go/0chain.net/sharder/sharder
RUN go mod download

# Build libzstd:
# FIXME: Can we move this to zchain_base?
RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    make clean libzstd.a

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

# Build it:
RUN go build -v -tags bn256

# Minimize the image size:
FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=0 $APP_DIR/go/0chain.net/sharder/sharder/sharder $APP_DIR/bin/
