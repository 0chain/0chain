#FROM golang:1.9.2-alpine3.6 AS build
FROM iron/go:dev

# Set the working directory to /app
WORKDIR /0chain

# Set an env var that matches your github repo name, replace treeder/dockergo here with your repo name
ENV SRC_DIR=/0chain

# Add the source code:
ADD . $SRC_DIR/

# Set the environment variable $GOPATH
ENV GOPATH=/0chain/code/go

# Download the dependencies
RUN go get github.com/golang/snappy
RUN go get github.com/gomodule/redigo/redis  
RUN go get github.com/vmihailenco/msgpack
RUN go get -u golang.org/x/crypto/...

# Remove Existing build files -- hack ?
#RUN cd $SRC_DIR/code/go/; rm -rf miner sharder pkg

# Build it:
RUN cd $SRC_DIR/code/go/src; go build 0chain.net/miner/miner; cp miner ../;

EXPOSE 6379
EXPOSE 7070

# Copy miner from src to root - hack ?
#RUN cp miner ../

# Install Redis
#RUN apk add --no-cache redis

# Start Redis Server
#RUN redis-server --daemonize yes

# Run the code once the build is successful
CMD cd $SRC_DIR/code/go; ./miner --port 7070 -test
