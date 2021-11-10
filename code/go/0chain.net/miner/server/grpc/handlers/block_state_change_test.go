package handlers

import (
	"context"
	"testing"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/stretchr/testify/assert"
)

// TestGetBlockStateChange
func TestGetBlockStateChange(t *testing.T) {
	t.Parallel()
	//
	tests := []struct {
		name    string
		req     *minerproto.GetBlockStateChangeRequest
		want    *minerproto.GetBlockStateChangeResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}

	//
	for _, c := range tests {
		t.Run("", func(t *testing.T) {
			client := NewMinerGRPCService()
			output, err := client.UnimplementedMinerServiceServer.GetBlockStateChange(context.TODO(), c.req)
			if err != nil {
				t.Error(err)
			}

			if !assert.Equal(t, c.want, output) {
				t.Error(err)
			}
		})
	}
}
