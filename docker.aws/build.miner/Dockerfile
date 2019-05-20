ARG image_tag

# Compile the miner in an un-tagged image so the final, tagged image can be smaller:
FROM 0chain_build_base:${image_tag} as miner_build
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/core/go.mod          ./code/go/0chain.net/core/go.sum          $SRC_DIR/go/0chain.net/core/
COPY ./code/go/0chain.net/chaincore/go.mod     ./code/go/0chain.net/chaincore/go.sum     $SRC_DIR/go/0chain.net/chaincore/
COPY ./code/go/0chain.net/smartcontract/go.mod ./code/go/0chain.net/smartcontract/go.sum $SRC_DIR/go/0chain.net/smartcontract/
COPY ./code/go/0chain.net/miner/go.mod         ./code/go/0chain.net/miner/go.sum         $SRC_DIR/go/0chain.net/miner/
COPY ./code/go/0chain.net/miner/miner/go.mod   ./code/go/0chain.net/miner/miner/go.sum   $SRC_DIR/go/0chain.net/miner/miner/
WORKDIR $SRC_DIR/go/0chain.net/miner/miner
RUN go mod download

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?
RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

# Build it:
# The argument should be repeated because we are in a new build
# context.
ARG image_tag
RUN go build -v -tags bn256 -ldflags "-X 0chain.net/core/build.BuildTag=${image_tag}"

# Copy the build artifact into a minimal runtime image:
FROM 0chain_run_base:${image_tag}
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=miner_build $APP_DIR/go/0chain.net/miner/miner/miner $APP_DIR/bin/miner

RUN addgroup -g 2000 -S 0chain && adduser -u 2000 -S 0chain -G 0chain
USER 0chain:0chain

