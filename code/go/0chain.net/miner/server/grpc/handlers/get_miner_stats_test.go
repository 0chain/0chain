package handlers

import (
	"context"
	"testing"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/stretchr/testify/assert"
)

// GetMinerStatsTest - test for GetMinerStats
func TestGetMinerStats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		req   *minerproto.GetMinerStatsRequest
		want  interface{}
		isErr bool
	}{
		// TODO: Add test cases.
	}

	//
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			client := NewMinerGRPCService()
			output, err := client.UnimplementedMinerServiceServer.GetMinerStats(context.TODO(), c.req)
			if err != nil {
				t.Error(err)
			}

			if !assert.Equal(t, c.want, output) {
				t.Error(err)
			}
		})
	}
}
