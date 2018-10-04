FROM zchain_base

ENV SRC_DIR=/0chain
ENV GOPATH=$GOPATH:$SRC_DIR/go

RUN go get github.com/gocql/gocql

WORKDIR $SRC_DIR

#Add the source code
ADD ./code/go/src $SRC_DIR/go/src

# Build it:
RUN go build 0chain.net/sharder/sharder

FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY ./bin $APP_DIR/bin
COPY --from=0 $APP_DIR/sharder $APP_DIR/bin
