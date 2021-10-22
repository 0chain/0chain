package handlers

import (
	"context"
	"testing"
	"time"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/stretchr/testify/assert"
)

func TestSignHandler(t *testing.T) {
	cases := []struct {
		name  string
		req   *minerproto.SignRequest
		want  *minerproto.SignResponse
		got   *minerproto.SignResponse
		isErr bool
	}{
		{
			name: "bad request",
			req: &minerproto.SignRequest{
				PublicKey:  "test",
				PrivateKey: "test",
				Data:       "test",
				TimeStamp:  time.Now().String(),
			},
			want: &minerproto.SignResponse{
				ClientId:  "random client id",
				Hash:      "127e6fbfe24a750e72930c220a8e138275656b8e5d8f48a98c3c92df2caba935",
				Signature: "kEwRnn1YAQ1o4Orw9LOZ",
			},
			isErr: true,
		},
		{
			name: "valid request",
			req: &minerproto.SignRequest{
				PublicKey:  "valid public key",
				PrivateKey: "valid privte key",
				Data:       "valid data",
				TimeStamp:  time.Now().String(),
			},
			want: &minerproto.SignResponse{
				ClientId:  "random client id",
				Hash:      "127e6fbfe24a750e72930c220a8e138275656b8e5d8f48a98c3c92df2caba935",
				Signature: "kEwRnn1YAQ1o4Orw9LOZ",
			},
			isErr: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMinerGRPCService()
			got, err := client.UnimplementedMinerServiceServer.Sign(context.TODO(), tt.req)
			if err != nil {
				t.Fatal(err)
			}

			if !assert.Equal(t, tt.want, got) {
				t.Fatal("not expected", err)
			}
		})
	}
}
