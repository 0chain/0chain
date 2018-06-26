FROM iron/go:dev

ENV SRC_DIR=/0chain
ENV GOPATH=$SRC_DIR/go

WORKDIR $SRC_DIR

#Download the dependencies
RUN go get github.com/golang/snappy
RUN go get github.com/gomodule/redigo/redis
RUN go get github.com/vmihailenco/msgpack
RUN go get -u golang.org/x/crypto/...
RUN go get go.uber.org/zap
RUN go get github.com/rcrowley/go-metrics
RUN go get github.com/spf13/viper
RUN go get gopkg.in/natefinch/lumberjack.v2

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build 0chain.net/miner/miner

FROM alpine
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=0 $APP_DIR/miner $APP_DIR/bin