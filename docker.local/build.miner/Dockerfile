# Compile the miner in an un-tagged image so the final, tagged image can be smaller:
FROM zchain_build_base as miner_build
ENV GO111MODULE=on

# Add the source code:
COPY code/go/0chain.net /0chain.net

# Set workdir
WORKDIR /0chain.net/miner/miner

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?

RUN [[ -d vendor ]] || ( \
    go mod download && \
    go mod vendor -v && \
    dir=github.com/valyala/gozstd && \
    rm -r vendor/$dir && \
    mv $GOPATH/pkg/mod/$dir* vendor/$dir )

RUN cd vendor/github.com/valyala/gozstd* && \
    chmod -R +w . && \
    make clean libzstd.a

# Build it:
ARG GIT_COMMIT
ENV GIT_COMMIT=$GIT_COMMIT
RUN go build -mod vendor -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

# Copy the build artifact into a minimal runtime image:
FROM zchain_run_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=miner_build /0chain.net/miner/miner/miner $APP_DIR/bin/