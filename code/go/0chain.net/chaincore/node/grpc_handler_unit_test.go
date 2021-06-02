package node_test

import (
	"context"
	"strings"
	"testing"

	"0chain.net/chaincore/client"

	"0chain.net/chaincore/node"

	"0chain.net/chaincore/mocks"

	"0chain.net/miner/minerGRPC"
)

func TestWhoAmIHandler(t *testing.T) {
	testcases := []struct {
		Description      string
		SelfNode         *node.Node
		ExpectedResponse string
	}{
		{
			Description:      "Nil value",
			SelfNode:         &node.Node{},
			ExpectedResponse: "m,,0,,",
		},
		{
			Description: "Success",
			SelfNode: &node.Node{
				Client: client.Client{
					PublicKey: "something",
				},
				N2NHost: "localhost",
				Host:    "localhost",
				Port:    3333,
			},
			ExpectedResponse: "m,localhost,3333,,something",
		},
	}

	for _, tc := range testcases {
		t.Run("WhoAmIHandler"+tc.Description, func(t *testing.T) {
			t.Parallel()

			mockedSelfNode := &mocks.ISelfNode{}
			server := node.NewGRPCMinerNodeService(mockedSelfNode)
			mockedSelfNode.On("Underlying").Return(tc.SelfNode)

			resp, err := server.WhoAmI(context.Background(), &minerGRPC.WhoAmIRequest{})
			if err != nil {
				t.Fatal(err)
			}

			if strings.TrimSpace(resp.Data) != tc.ExpectedResponse {
				t.Fatal("description - " + tc.Description + "\n\nincorrect response,\n\nexpected - " + tc.ExpectedResponse + "\n\ngot - " + resp.Data)
			}
		})
	}
}
