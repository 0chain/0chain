package handlers

import (
	"context"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// GetChainStats returns the chain stats
func (m *minerGRPCService) GetChainStats(ctx context.Context, req *minerproto.GetChainStatsRequest) (*minerproto.GetChainStatsResponse, error) {
	c := chain.GetServerChain()

	self := node.Self.Underlying()

	output := []byte("")
	return &minerproto.GetChainStatsResponse{
		Body: &httpbody.HttpBody{
			ContentType: "text/html;charset=UTF-8",
			Data:        output,
		},
	}, nil
}
