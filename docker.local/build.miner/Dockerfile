# Compile the miner in an un-tagged image so the final, tagged image can be smaller:
FROM zchain_build_base as miner_build
ENV ROOT=/0chain
ENV SRC_DIR=$ROOT/code/go/0chain.net
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/go.mod $SRC_DIR/
COPY ./code/go/0chain.net/go.sum $SRC_DIR/
COPY ./generate_mocks.sh $ROOT/generate_mocks.sh

RUN cd $SRC_DIR && go mod download -x

COPY ./code/go/0chain.net $SRC_DIR

RUN go get github.com/vektra/mockery/v2/.../
RUN cd $ROOT && ls && ./generate_mocks.sh


# Set workdir
WORKDIR $SRC_DIR

RUN go mod vendor -v

RUN rm -r ./vendor/github.com/valyala/gozstd

RUN cp -r /gozstd ./vendor/github.com/valyala/gozstd

WORKDIR $SRC_DIR/miner/miner


# Build it:
ARG GIT_COMMIT
ARG DEV
ENV GIT_COMMIT=$GIT_COMMIT
RUN if [ "$DEV" = "yes" ] ;\
    then echo "Artifacts will be copied from local machine";\
    else go build -v -tags "bn256 development" -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT" ;\
fi

# Copy the build artifact into a minimal runtime image:
FROM zchain_run_base
ENV APP_DIR=/0chain
ENV SRC_DIR=/code/go/0chain.net
WORKDIR $APP_DIR

COPY --from=miner_build $APP_DIR/code/go/0chain.net/miner/miner/miner $APP_DIR/bin/miner
