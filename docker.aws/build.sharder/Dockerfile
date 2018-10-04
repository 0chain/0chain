#Store all files and run environment under 0chain userid.
FROM ubuntu:16.04
RUN groupadd -r 0chain && useradd --no-log-init -r -g 0chain 0chain

FROM zchain_base

ENV SRC_DIR=/0chain
ENV GOPATH=$GOPATH:$SRC_DIR/go

RUN go get github.com/gocql/gocql

WORKDIR $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build 0chain.net/sharder/sharder

FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=1 $APP_DIR/sharder $APP_DIR/bin
USER 2000:2000
