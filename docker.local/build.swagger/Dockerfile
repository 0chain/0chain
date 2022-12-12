FROM zchain_build_base as cross

ARG commit_hash="dev"
ARG tag_name="dev"

RUN git clone https://github.com/go-swagger/go-swagger /work
WORKDIR /work

RUN apk --no-cache add ca-certificates shared-mime-info mailcap git build-base binutils-gold

RUN mkdir -p bin &&\
  LDFLAGS="$LDFLAGS -X github.com/go-swagger/go-swagger/cmd/swagger/commands.Commit=${commit_hash}" &&\
  LDFLAGS="$LDFLAGS -X github.com/go-swagger/go-swagger/cmd/swagger/commands.Version=${tag_name}" &&\
  CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -tags osusergo,netgo -o bin/swagger -ldflags "$LDFLAGS" -a ./cmd/swagger

FROM zchain_build_base

ENV SRC_DIR=/0chain
ENV GO111MODULE=on

# Download the dependencies:
# Will be cached if we don't change mod/sum files
COPY ./code/go/0chain.net/go.mod $SRC_DIR/go/0chain.net/
COPY ./code/go/0chain.net/go.sum $SRC_DIR/go/0chain.net/

RUN cd $SRC_DIR/go/0chain.net && go mod download -x

COPY ./code/go/0chain.net $SRC_DIR/code/go/0chain.net

COPY --from=cross /work/bin/swagger /usr/bin/swagger
COPY --from=cross /work/generator/templates/contrib /templates/

WORKDIR $SRC_DIR/code/go
