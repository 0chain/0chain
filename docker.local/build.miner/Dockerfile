FROM zchain_base

ENV SRC_DIR=/0chain
ENV GOPATH=$SRC_DIR/go

# Download the dependencies
ADD ./code/go/src/0chain.net/Gopkg.toml ./code/go/src/0chain.net/Gopkg.lock $GOPATH/src/0chain.net/
WORKDIR $GOPATH/src/0chain.net
RUN dep ensure -vendor-only

# Add the source code after installing dependencies
ADD ./code/go/src $GOPATH/src

# Ensures the code tree, manifest, lock, and vendor are in sync with each other
RUN dep ensure

# Build it:
WORKDIR $SRC_DIR
RUN go build 0chain.net/miner/miner

FROM zchain_base
ENV APP_DIR=/0chain
WORKDIR $APP_DIR
RUN mkdir -p $APP_DIR/bin
COPY --from=0 $APP_DIR/miner $APP_DIR/bin
