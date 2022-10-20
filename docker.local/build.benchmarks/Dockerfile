FROM zchain_build_base as benchmark_build

ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files

# RUN cd $SRC_DIR/go/0chain.net && \
#     go mod download -x

COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

WORKDIR $SRC_DIR/go/0chain.net
RUN go mod vendor -v

RUN rm -r ./vendor/github.com/valyala/gozstd

RUN cp -r /gozstd ./vendor/github.com/valyala/gozstd

WORKDIR $SRC_DIR/go/0chain.net/core

# Build libzstd:
# FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
# FIXME: Also, is there a way we can move this to zchain_build_base?
# RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
#     chmod -R +w . && \
#     make clean libzstd.a
WORKDIR $SRC_DIR/go/0chain.net/smartcontract/benchmark/main

    # Build it:
ARG GIT_COMMIT
ENV GIT_COMMIT=$GIT_COMMIT
RUN go build -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

# Copy the build artifact into a minimal runtime image:
FROM zchain_run_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
COPY --from=benchmark_build $APP_DIR/go/0chain.net/smartcontract/benchmark/main/main $APP_DIR/bin/
