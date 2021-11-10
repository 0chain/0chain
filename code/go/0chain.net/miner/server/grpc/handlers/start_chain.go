package handlers

import (
	"context"
	"fmt"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/miner"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"go.uber.org/zap"
)

// StartChain - starts a chain
func (m *minerGRPCService) StartChain(ctx context.Context, req *minerproto.StartChainRequest) (*minerproto.StartChainResponse, error) {
	mc := miner.GetMinerChain()

	mb := mc.GetMagicBlock(req.Round)

	if mb == nil || !mb.Miners.HasNode(req.NodeId) {
		logging.Logger.Error("failed to send start chain", zap.Any("id", req.NodeId))
		return nil, common.NewError("failed to send start chain", "miner is not in active set")
	}

	if mc.GetCurrentRound() != req.Round {
		logging.Logger.Error("failed to send start chain -- different rounds", zap.Any("current_round", mc.GetCurrentRound()), zap.Any("requested_round", req.Round))
		return nil, common.NewError("failed to send start chain", fmt.Sprintf("differt_rounds -- current_round: %v, requested_round: %v", mc.GetCurrentRound(), req.Round))
	}

	return &minerproto.StartChainResponse{
		// Id: ,
		Start: mc.ChainStarted(ctx),
	}, nil
}
