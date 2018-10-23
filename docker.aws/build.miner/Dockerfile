#Store all files and run environment under 0chain userid.
ARG ZCHAIN_CODEBASE
RUN echo "ZCHAIN_CODEBASE: $ZCHAIN_CODEBASE"

FROM ubuntu:16.04
RUN groupadd -r 0chain && useradd --no-log-init -r -g 0chain 0chain

FROM 0chain/base
ADD MINER_VERSION .

ENV SRC_DIR=/0chain

ENV GOPATH=$GOPATH:$SRC_DIR/go

WORKDIR $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build 0chain.net/miner/miner

FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=1 $APP_DIR/miner $APP_DIR/bin
USER 2000:2000
