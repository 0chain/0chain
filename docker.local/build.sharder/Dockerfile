# Compile the sharder in an un-tagged image so the final, tagged image can be smaller:
FROM zchain_build_base as sharder_build
ENV ROOT=/0chain
ENV SRC_DIR=$ROOT/code/go/0chain.net
ENV GO111MODULE=on

# Add the source code:
COPY ./code/go/0chain.net/go.mod $SRC_DIR/
COPY ./code/go/0chain.net/go.sum $SRC_DIR/

RUN cd $SRC_DIR && go mod download -x

COPY ./code/go/0chain.net $SRC_DIR

# Set workdir
WORKDIR $SRC_DIR

RUN go mod vendor -v

RUN rm -r ./vendor/github.com/valyala/gozstd

RUN cp -r /gozstd ./vendor/github.com/valyala/gozstd

# Set workdir
WORKDIR $SRC_DIR/sharder/sharder

# Build it:
ARG GIT_COMMIT
ENV GIT_COMMIT=$GIT_COMMIT
RUN go build -mod vendor -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

# Copy the build artifact into a minimal runtime image:
FROM zchain_run_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=sharder_build $APP_DIR/code/go/0chain.net/sharder/sharder/sharder $APP_DIR/bin/
