# Compile the sharder in an un-tagged image so the final, tagged image can be smaller:
FROM zchain_build_base as sharder_build

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

# Build it:
WORKDIR $SRC_DIR/go/0chain.net/sharder/sharder

ARG GIT_COMMIT
ENV GIT_COMMIT=$GIT_COMMIT
RUN go build -v -tags bn256 -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

# Copy the build artifact into a minimal runtime image:
FROM zchain_run_base

ENV APP_DIR=/0chain
WORKDIR $APP_DIR

COPY --from=sharder_build $APP_DIR/go/0chain.net/sharder/sharder/sharder $APP_DIR/bin/
