package handlers

import (
	"context"
	"testing"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

func TestConfigUpdate(t *testing.T) {
	cases := []struct {
		name  string
		req   *minerproto.ConfigUpdateRequest
		want  interface{}
		isErr bool
	}{}

	//

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			// client := handlers.NewMinerGRPCService()
			client := NewMinerGRPCService()
			output, err := client.UnimplementedMinerServiceServer.ConfigUpdate(context.TODO(), c.req)
			if err != nil {
				t.Error(err)
			}

			if !assert.Equal(t, c.want, output) {
				t.Error(err)
			}
		})
	}
}

func TestConfigUpdateAll(t *testing.T) {
	cases := []struct {
		name  string
		req   *minerproto.ConfigUpdateRequest
		want  *httpbody.HttpBody
		isErr bool
	}{}

	//

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			// client := handlers.NewMinerGRPCService()
			client := NewMinerGRPCService()
			output, err := client.UnimplementedMinerServiceServer.ConfigUpdate(context.TODO(), c.req)
			if err != nil {
				t.Error(err)
			}

			if !assert.Equal(t, c.want, output) {
				t.Error(err)
			}
		})
	}
}
