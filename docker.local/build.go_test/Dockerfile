FROM zchain_base

ENV SRC_DIR=/0chain
ENV GOPATH=$SRC_DIR/go

# Download the dependencies
COPY ./0chain.net/Gopkg.toml ./0chain.net/Gopkg.lock $GOPATH/src/0chain.net/
WORKDIR $GOPATH/src/0chain.net
RUN dep ensure -vendor-only

# Add the source code after installing dependencies
COPY . $GOPATH/src

# Ensures the code tree, manifest, lock, and vendor are in sync with each other
RUN dep ensure
