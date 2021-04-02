FROM zchain_build_base
ENV APP_DIR=/0chain

# Add the source code:
COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

# Build it:
RUN cd $SRC_DIR/go/0chain.net/smartcontract/multisigsc/test && \
    go build -v -tags bn256 -o $APP_DIR/bin/test_multisigsc

WORKDIR $APP_DIR
