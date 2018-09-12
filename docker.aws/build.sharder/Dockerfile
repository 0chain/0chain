FROM zchain_base

ENV SRC_DIR=/0chain
ENV GOPATH=$SRC_DIR/go

WORKDIR $SRC_DIR

#Download the dependencies
RUN go get github.com/golang/snappy
RUN go get github.com/tecbot/gorocksdb
RUN go get github.com/gomodule/redigo/redis
RUN go get github.com/gocql/gocql
RUN go get github.com/vmihailenco/msgpack
RUN go get -u golang.org/x/crypto/...
RUN go get go.uber.org/zap
RUN go get github.com/rcrowley/go-metrics
RUN go get github.com/spf13/viper
RUN go get gopkg.in/natefinch/lumberjack.v2
RUN go get github.com/koding/cache
RUN go get github.com/hashicorp/golang-lru

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build 0chain.net/sharder/sharder

FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=0 $APP_DIR/sharder $APP_DIR/bin

#Store all files and run environment under 0chain userid.
FROM ubuntu:16.04
RUN groupadd -r 0chain && useradd --no-log-init -r -g 0chain 0chain
USER 2000:2000

