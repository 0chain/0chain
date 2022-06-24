FROM golang:1.18.3-alpine3.16 as zdns_build

RUN apk add --update --no-cache build-base linux-headers git cmake bash perl grep

# Install Herumi's cryptography
RUN apk add gmp gmp-dev openssl-dev && \
    cd /tmp && \
    wget -O - https://github.com/herumi/mcl/archive/master.tar.gz | tar xz && \
    wget -O - https://github.com/herumi/bls/archive/master.tar.gz | tar xz && \
    mv mcl* mcl && \
    mv bls* bls && \
    make -C mcl -j $(nproc) lib/libmclbn256.so install && \
    cp mcl/lib/libmclbn256.so /usr/local/lib && \
    make MCL_DIR=../mcl -C bls -j $(nproc) install && \
    rm -R /tmp/mcl && \
    rm -R /tmp/bls

ENV SRC_DIR=/0dns
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0dns.io/core/go.mod          ./code/go/0dns.io/core/go.sum          $SRC_DIR/go/0dns.io/core/
COPY ./code/go/0dns.io/zdnscore/go.mod     ./code/go/0dns.io/zdnscore/go.sum     $SRC_DIR/go/0dns.io/zdnscore/

WORKDIR $SRC_DIR/go/0dns.io/zdnscore/zdns
RUN go mod download

# Add the source code:
ADD ./code/go/0dns.io $SRC_DIR/go/0dns.io

RUN go build -v -tags "bn256 development" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

# Copy the build artifact into a minimal runtime image:
FROM golang:1.18.3-alpine3.16
ENV APP_DIR=/0dns
WORKDIR $APP_DIR
RUN apk add gmp gmp-dev openssl-dev
COPY --from=zdns_build  /usr/local/lib/libmcl*.so \
                        /usr/local/lib/libbls*.so \
                        /usr/local/lib/
COPY --from=zdns_build $APP_DIR/go/0dns.io/zdnscore/zdns/zdns $APP_DIR/bin/zdns

