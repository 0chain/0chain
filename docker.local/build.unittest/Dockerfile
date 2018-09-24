FROM zchain_base

ENV SRC_DIR=/0chain
ENV GOPATH=$SRC_DIR/go

# Add the source code
ADD ./code/go/src $GOPATH/src

RUN for test in $(find $GOPATH/src/0chain.net -name \*_test.go); do dirname $test; done | sort | uniq > $GOPATH/test_pkgs.txt

# Download the dependencies
RUN go test -i $(cat $GOPATH/test_pkgs.txt)

CMD go test $(cat $GOPATH/test_pkgs.txt)
