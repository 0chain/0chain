package handlers

import (
	"context"
	"encoding/hex"

	"0chain.net/core/logging"
	"0chain.net/miner"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"go.uber.org/zap"
)

// GetPartialState - get the partial state of a node
func (m *minerGRPCService) GetPartialState(ctx context.Context, req *minerproto.GetPartialStateRequest) (*minerproto.GetPartialStateResponse, error) {
	mc := miner.GetMinerChain()
	nodeKey, err := hex.DecodeString(req.Node)
	if err != nil {
		return nil, err
	}
	ps, err := mc.GetStateFrom(ctx, nodeKey)
	if err != nil {
		logging.Logger.Error("partial state handler", zap.String("key", req.Node), zap.Error(err))
		return nil, err
	}

	return &minerproto.GetPartialStateResponse{
		PartialState: &minerproto.PartialState{
			Hash:      ps.Hash,
			Version:   ps.Version,
			StartRoot: ps.StartRoot,
		},
	}, nil
}
