package handlers_test

import (
	"context"
	"testing"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/grpc/metadata"
)

func TestSignHandler(t *testing.T) {
	t.Parallel()

	// init grpc
	client, err := makeTestClient(nil)
	if err != nil {
		t.Error(err)
		return
	}

	// server, err := makeTestServer()
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }

	cases := []struct {
		name             string
		context          metadata.MD
		input            *minerproto.SignRequest
		expectedFileName string
		expectingError   bool
	}{}

	for _, tc := range cases {
		ctx := context.Background()
		ctx = metadata.NewOutgoingContext(ctx, tc.context)

		resp, err := client.Sign(ctx, tc.input)
		if err != nil {
			if !tc.expectingError {
				t.Fatal(err)
			}
			continue
		}

		if tc.expectingError {
			t.Fatal("expected error")
		}

		t.Log(resp)
	}
}
