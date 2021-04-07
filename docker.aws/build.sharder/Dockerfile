# Compile the sharder in an un-tagged image so the final, tagged image can be smaller:
ARG image_tag
FROM 0chain_build_base:${image_tag} as sharder_build
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

RUN cd $SRC_DIR/go/0chain.net && \
    go mod download
WORKDIR $SRC_DIR/go/0chain.net/sharder/sharder

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?
RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

# Build it:
# The argument should be repeated because we are in a new build
# context.
ARG image_tag
RUN go build -v -tags bn256 -ldflags "-X 0chain.net/core/build.BuildTag=${image_tag}"

# Copy the build artifact into a minimal runtime image:
FROM 0chain_run_base:${image_tag}
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=sharder_build $APP_DIR/go/0chain.net/sharder/sharder/sharder $APP_DIR/bin/

#Store all files and run environment under 0chain userid.
RUN addgroup -g 2000 -S 0chain && adduser -u 2000 -S 0chain -G 0chain
USER 0chain:0chain
