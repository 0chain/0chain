package handlers

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/miner"
	minerproto "0chain.net/miner/proto/api/src/proto"
)

// GetBlockStateChange returns the block state change for the given block hash.
func (m *minerGRPCService) GetBlockStateChange(ctx context.Context, req *minerproto.GetBlockStateChangeRequest) (*minerproto.GetBlockStateChangeResponse, error) {
	b, err := miner.GetNotarizedBlock(ctx, req.Round, req.Hash)
	if err != nil {
		return nil, err
	}

	if b.GetStateStatus() != block.StateSuccessful {
		return nil, common.NewError("state_not_verified", "state is not computed and validated locally")
	}

	var bsc = block.NewBlockStateChange(b)
	if state.Debug() {
		// logging.Logger.Info("block state change handler", zap.Int64("round", b.Round),
		// 	zap.String("block", b.Hash),
		// 	zap.Int("state_changes", b.ClientState.GetChangeCount()),
		// 	zap.Int("sc_nodes", len(bsc.Nodes)))
	}

	if bsc.GetRoot() == nil {
		// cr := miner.GetMinerChain().GetCurrentRound()
		// logging.Logger.Debug("get state changes - state nil root",
		// 	zap.Int64("round", b.Round),
		// 	zap.Int64("current_round", cr))
	}

	return &minerproto.GetBlockStateChangeResponse{
		PartialState: &minerproto.PartialState{
			Hash:      bsc.Hash,
			Version:   bsc.Version,
			StartRoot: bsc.StartRoot,
		},
		Block: bsc.Block,
	}, nil
}
