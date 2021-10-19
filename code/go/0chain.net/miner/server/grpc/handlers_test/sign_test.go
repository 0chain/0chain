package handlers_test

import (
	"context"
	"log"
	"testing"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestSignHandler(t *testing.T) {
	tests := []struct {
		name   string
		req    *minerproto.SignRequest
		resp   *minerproto.SignResponse
		status codes.Code
		err    string
	}{
		{
			name: "bad request",
		},
		{
			name: "bad public key",
		},
		{
			name: "bad private key",
		},
		{
			name: "empty data",
		},
	}

	// created new grpc conn with dialer()
	conn, err := grpc.DialContext(context.Background(), "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// grpc client used "bufconn"
	client := minerproto.NewMinerServiceClient(conn)

	for _, tt := range tests {
		ctx := context.Background()

		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Sign(ctx, tt.req)
			if err != nil {
				t.Fatal(err)
			}

			if resp != tt.resp {
				t.Fatal("not expected", err)
			}
		})
	}
}
