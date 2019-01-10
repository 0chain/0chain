FROM zchain_base

ENV SRC_DIR=/0chain
ENV THIRD_PARTY_GOPATH=$GOPATH
ENV GOPATH=$GOPATH:$SRC_DIR/go

WORKDIR $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN CGO_CFLAGS="-I$THIRD_PARTY_GOPATH/src/github.com/herumi/bls/include -I$THIRD_PARTY_GOPATH/src/github.com/herumi/mcl/include" CGO_LDFLAGS="-L$THIRD_PARTY_GOPATH/src/github.com/herumi/bls/lib -L$THIRD_PARTY_GOPATH/src/github.com/herumi/mcl/lib" go build -tags bn256 0chain.net/miner/miner




FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=0 $APP_DIR/miner $APP_DIR/bin
