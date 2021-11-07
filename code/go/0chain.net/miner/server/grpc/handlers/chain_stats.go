package handlers

import (
	"context"
	"fmt"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/miner"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/0chain/errors"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetChainStats returns the chain stats
func (m *minerGRPCService) GetChainStats(ctx context.Context, req *minerproto.GetChainStatsRequest) (*minerproto.GetChainStatsResponse, error) {
	c := miner.GetMinerChain().Chain
	stats := diagnostics.GetStatistics(c, chain.SteadyStateFinalizationTimer, 1000000.0)

	v, ok := stats.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to convert stats to map")
	}

	str, err := structpb.NewStruct(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create struct")
	}

	return &minerproto.GetChainStatsResponse{
		Stats: str,
	}, nil
}
