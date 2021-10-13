ackage handlers

import (
	"context"

	minerproto "0chain.net/miner/proto/api/src/proto"

	"0chain.net/core/encryption"

)

// Hash is the handler for the Hash RPC
func (m *minerGRPCService) Hash(ctx context.Context, req *minerproto.HashRequest) (*minerproto.HashResponse, error) {
	
	text := req.Text

	hash := encryption.Hash(text)

	return &minerproto.HashResponse{
		Hash: hash,
	}, nil
}