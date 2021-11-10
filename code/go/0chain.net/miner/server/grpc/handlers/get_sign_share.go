package handlers

import (
	"context"

	minerproto "0chain.net/miner/proto/api/src/proto"
)

// GetSignShare -
func (m *minerGRPCService) GetSignShare(ctx context.Context, req *minerproto.GetSignShareRequest) (*minerproto.GetSignShareResponse, error) {
	// TODO (twiny): Implement.

	return &minerproto.GetSignShareResponse{
		DkgKeyShare: &minerproto.DKGKeyShare{
			// Id:      message.ID,
			// Message: message.Message,
			// Share:   message.Share,
			// Sign:    message.Sign,
		},
	}, nil
}
