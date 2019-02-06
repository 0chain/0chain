#Store all files and run environment under 0chain userid.
ARG image_tag

FROM ubuntu:16.04
RUN groupadd -r 0chain && useradd --no-log-init -r -g 0chain 0chain

FROM 0chain/base:${image_tag}

ADD ./docker.aws/build.miner/MINER_VERSION .

ENV SRC_DIR=/0chain
ENV THIRD_PARTY_GOPATH=$GOPATH
ENV GOPATH=$GOPATH:$SRC_DIR/go

WORKDIR $SRC_DIR
COPY ./docker.aws/build.miner/MINER_VERSION $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build -tags bn256 0chain.net/miner/miner


#Copy the miner executable
WORKDIR $SRC_DIR/bin
RUN cp $SRC_DIR/miner $SRC_DIR/bin

#Run the miner from /0chain
WORKDIR $SRC_DIR
USER 2000:2000
