FROM zchain_build_base

ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/go.mod $SRC_DIR/go/0chain.net/
COPY ./code/go/0chain.net/go.sum $SRC_DIR/go/0chain.net/

RUN cd $SRC_DIR/go/0chain.net && go mod download -x

COPY ./code/go/0chain.net $SRC_DIR/code/go/0chain.net

RUN git clone https://github.com/go-swagger/go-swagger
WORKDIR ./go-swagger
RUN go install ./cmd/swagger

WORKDIR $SRC_DIR/code/go