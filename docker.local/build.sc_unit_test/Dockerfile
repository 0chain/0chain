FROM zchain_build_base
ENV SRC_DIR=/0chain
ENV GO111MODULE=on

COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

RUN cd $SRC_DIR/go/0chain.net && \
    go mod download


WORKDIR $SRC_DIR/go
