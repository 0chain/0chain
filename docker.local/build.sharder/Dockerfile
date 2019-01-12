FROM zchain_base

ENV SRC_DIR=/0chain
ENV THIRD_PARTY_GOPATH=$GOPATH
ENV GOPATH=$GOPATH:$SRC_DIR/go

RUN go get github.com/gocql/gocql

WORKDIR $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN CGO_CFLAGS="-I$THIRD_PARTY_GOPATH/src/github.com/herumi/bls/include -I$THIRD_PARTY_GOPATH/src/github.com/herumi/mcl/include" CGO_LDFLAGS="-L$THIRD_PARTY_GOPATH/src/github.com/herumi/bls/lib -L$THIRD_PARTY_GOPATH/src/github.com/herumi/mcl/lib" go build 0chain.net/sharder/sharder

FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=0 $APP_DIR/sharder $APP_DIR/bin
