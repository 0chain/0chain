package chain_test

import (
	"context"
	"testing"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/chain"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/block"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/mocks"
	"github.com/0chain/0chain/code/go/0chain.net/miner/minerGRPC"
)

func TestChain_GetLatestFinalizedBlockSummary(t *testing.T) {
	testcases := []struct {
		Description      string
		Summary          *block.BlockSummary
		ExpectedResponse string
	}{
		{
			Description:      "Nil value",
			Summary:          &block.BlockSummary{},
			ExpectedResponse: "block_summary:{}",
		},
		{
			Description: "Success",
			Summary: &block.BlockSummary{
				Hash:                  "something",
				MinerID:               "something1",
				MerkleTreeRoot:        "something2",
				ReceiptMerkleTreeRoot: "something3",
			},
			ExpectedResponse: `block_summary:{hash:"something"  miner_id:"something1"  merkle_tree_root:"something2"  receipt_merkle_tree_root:"something3"}`,
		},
	}

	for _, v := range testcases {
		tc := v
		t.Run("GetLatestFinalizedBlockSummary"+tc.Description, func(t *testing.T) {
			t.Parallel()

			mockedChain := &mocks.IChain{}
			server := chain.NewGRPCMinerChainService(mockedChain)
			mockedChain.On("GetLatestFinalizedBlockSummary").Return(tc.Summary)

			resp, err := server.GetLatestFinalizedBlockSummary(context.Background(), &minerGRPC.GetLatestFinalizedBlockSummaryRequest{})
			if err != nil {
				t.Fatal(err)
			}

			got := resp.String()

			if got != tc.ExpectedResponse {
				t.Fatal("description - " + tc.Description + "\n\nincorrect response,\n\nexpected - " + tc.ExpectedResponse + "\n\ngot - " + got)
			}
		})
	}
}
