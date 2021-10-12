package handlers

import (
	"context"

	minerproto "0chain.net/miner/proto/api/src/proto"
)

// Sign
func (m *minerGRPCService) Sign(ctx context.Context, req *minerproto.SignRequest) (*minerproto.SignResponse, error) {
	// implement new logic
	//
	return &minerproto.SignResponse{}, nil
}
