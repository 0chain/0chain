package handlers

import (
	"context"
	"testing"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/stretchr/testify/assert"
)

func TestPutTransaction(t *testing.T) {
	t.Parallel()

	//
	cases := []struct {
		name  string
		req   *minerproto.PutTransactionRequest
		want  interface{}
		isErr bool
	}{}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			client := NewMinerGRPCService()
			got, err := client.UnimplementedMinerServiceServer.PutTransaction(context.TODO(), c.req)
			if err != nil {
				t.Error(err)
			}

			if !assert.Equal(t, c.want, got) {
				t.Error(err)
			}
		})
	}
}
