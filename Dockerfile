FROM iron/go:dev

ENV DOCKER=true

ENV SRC_DIR=/0chain

# Add the source code:
ADD ./code/go/src $SRC_DIR/code/go/src

# Add the config dir
ADD ./code/go/config $SRC_DIR/config

Add ./code/go/data $SRC_DIR/data

WORKDIR $SRC_DIR

# Set the environment variable $GOPATH
ENV GOPATH=/0chain/code/go

# Download the dependencies
RUN go get github.com/golang/snappy
RUN go get github.com/gomodule/redigo/redis
RUN go get github.com/vmihailenco/msgpack
RUN go get -u golang.org/x/crypto/...

# Build it:
RUN cd $SRC_DIR; go build 0chain.net/miner/miner

EXPOSE 7070

# Run the code once the build is successful
CMD ./miner --port 7070 -test
