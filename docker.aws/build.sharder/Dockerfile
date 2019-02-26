# Compile the sharder in an un-tagged image so the final, tagged image can be smaller:
ARG image_tag
FROM 0chain/build_base:${image_tag} as sharder_build
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/core/go.mod            ./code/go/0chain.net/core/go.sum            $SRC_DIR/go/0chain.net/core/
COPY ./code/go/0chain.net/chaincore/go.mod       ./code/go/0chain.net/chaincore/go.sum       $SRC_DIR/go/0chain.net/chaincore/
COPY ./code/go/0chain.net/smartcontract/go.mod   ./code/go/0chain.net/smartcontract/go.sum   $SRC_DIR/go/0chain.net/smartcontract/
COPY ./code/go/0chain.net/sharder/go.mod         ./code/go/0chain.net/sharder/go.sum         $SRC_DIR/go/0chain.net/sharder/
COPY ./code/go/0chain.net/sharder/sharder/go.mod ./code/go/0chain.net/sharder/sharder/go.sum $SRC_DIR/go/0chain.net/sharder/sharder/

WORKDIR $SRC_DIR/go/0chain.net/sharder/sharder
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
RUN go build -v -tags bn256

# Copy the build artifact into a minimal runtime image:
FROM 0chain/run_base:${image_tag}
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=sharder_build $APP_DIR/go/0chain.net/sharder/sharder/sharder $APP_DIR/bin/

#Store all files and run environment under 0chain userid.
RUN addgroup -g 1000 -S 0chain && adduser -u 1000 -S 0chain -G 0chain
USER 0chain:0chain
