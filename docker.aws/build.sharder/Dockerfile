#Store all files and run environment under 0chain userid.
ARG image_tag

FROM ubuntu:16.04
RUN groupadd -r 0chain && useradd --no-log-init -r -g 0chain 0chain

FROM 0chain/base:${image_tag}

ADD ./docker.aws/build.sharder/SHARDER_VERSION .

ENV SRC_DIR=/0chain


#Add the source code
ADD ./code/go $SRC_DIR/go

ENV GO111MODULE=on

WORKDIR $SRC_DIR/go/0chain.net/sharder/sharder

RUN go mod download

# Build it:
RUN cd $SRC_DIR/go/0chain.net/sharder/sharder && \
    go get -d && \
    cd $GOPATH/pkg/mod/github.com/valyala/gozstd* && \
    make clean libzstd.a && \
    cd $SRC_DIR/go/0chain.net/sharder/sharder && \
    go build -v -tags bn256

# Build it:
RUN cd $SRC_DIR/go/0chain.net/sharder/sharder && \
    go build -v -tags bn256

WORKDIR $SRC_DIR/bin
RUN cp $SRC_DIR/go/0chain.net/sharder/sharder/sharder $SRC_DIR/bin

#Run the sharder from the /0chain
WORKDIR $SRC_DIR
USER 2000:2000