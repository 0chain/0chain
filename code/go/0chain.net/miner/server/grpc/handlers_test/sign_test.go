package handlers_test

import (
	"context"
	"testing"
	"time"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"0chain.net/miner/server/grpc/handlers"
	"github.com/stretchr/testify/assert"
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
			req: &minerproto.SignRequest{
				PublicKey:  "",
				PrivateKey: "",
				Data:       "",
				TimeStamp:  time.Now().String(),
			},
			resp: &minerproto.SignResponse{
				ClientId:  "",
				Hash:      "",
				Signature: "",
			},
		},
		{
			name: "bad public key",
			req: &minerproto.SignRequest{
				PublicKey:  "",
				PrivateKey: "",
				Data:       "",
				TimeStamp:  time.Now().String(),
			},
			resp: &minerproto.SignResponse{
				ClientId:  "",
				Hash:      "",
				Signature: "",
			},
		},
		{
			name: "bad private key",
			req: &minerproto.SignRequest{
				PublicKey:  "",
				PrivateKey: "",
				Data:       "",
				TimeStamp:  time.Now().String(),
			},
			resp: &minerproto.SignResponse{
				ClientId:  "",
				Hash:      "",
				Signature: "",
			},
		},
		{
			name: "empty data",
			req: &minerproto.SignRequest{
				PublicKey:  "",
				PrivateKey: "",
				Data:       "",
				TimeStamp:  time.Now().String(),
			},
			resp: &minerproto.SignResponse{
				ClientId:  "",
				Hash:      "",
				Signature: "",
			},
		},
		{
			name: "valid request",
			req: &minerproto.SignRequest{
				PublicKey:  "",
				PrivateKey: "",
				Data:       "",
				TimeStamp:  time.Now().String(),
			},
			resp: &minerproto.SignResponse{
				ClientId:  "",
				Hash:      "",
				Signature: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := handlers.NewMinerGRPCService()

			resp, err := client.UnimplementedMinerServiceServer.Sign(context.TODO(), tt.req)
			if err != nil {
				t.Fatal(err)
			}

			if !assert.Equal(t, tt.resp, resp) {
				t.Fatal("not expected", err)
			}
		})
	}
}
