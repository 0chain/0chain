package handlers

import (
	"context"

	"0chain.net/core/encryption"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// Hash is the handler for the Hash RPC
func (m *minerGRPCService) Hash(ctx context.Context, req *minerproto.HashRequest) (*httpbody.HttpBody, error) {

	text := req.Text

	hash := encryption.Hash(text)

	return &httpbody.HttpBody{
		ContentType: "text/html",
		Data:        []byte(hash),
	}, nil
}
