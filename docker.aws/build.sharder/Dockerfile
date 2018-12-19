#Store all files and run environment under 0chain userid.
ARG image_tag

FROM ubuntu:16.04
RUN groupadd -r 0chain && useradd --no-log-init -r -g 0chain 0chain

FROM 0chain/base:${image_tag}

ADD ./docker.aws/build.sharder/SHARDER_VERSION .

ENV SRC_DIR=/0chain
ENV GOPATH=$GOPATH:$SRC_DIR/go

RUN go get github.com/gocql/gocql

WORKDIR $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build 0chain.net/sharder/sharder

WORKDIR $SRC_DIR/bin
RUN cp $SRC_DIR/sharder $SRC_DIR/bin

#Run the sharder from the /0chain
WORKDIR $SRC_DIR
USER 2000:2000
