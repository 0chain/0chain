FROM iron/go:dev

ENV SRC_DIR=/0chain
ENV GOPATH=$SRC_DIR/go

WORKDIR $SRC_DIR

#Download the dependencies
RUN go get github.com/golang/snappy
RUN go get github.com/gomodule/redigo/redis
RUN go get github.com/vmihailenco/msgpack
RUN go get -u golang.org/x/crypto/...

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build 0chain.net/miner/miner
