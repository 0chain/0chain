FROM zchain_base

RUN go get github.com/gocql/gocql
RUN go get github.com/stretchr/testify/assert

WORKDIR /go/src

#Add the source code
ADD ./code/go/src .

