package handlers

import (
	"context"

	"google.golang.org/genproto/googleapis/api/httpbody"

	minerproto "0chain.net/miner/proto/api/src/proto"

	"0chain.net/core/encryption"
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
