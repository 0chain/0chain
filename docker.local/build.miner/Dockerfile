FROM zchain_base

ENV SRC_DIR=/0chain
ENV GOPATH=$GOPATH:$SRC_DIR/go

WORKDIR $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build -tags bn256 0chain.net/miner/miner


FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=0 $APP_DIR/miner $APP_DIR/bin
