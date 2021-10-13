package handlers

import (
	"context"
	"encoding/hex"
	"fmt"

	"0chain.net/core/encryption"
	minerproto "0chain.net/miner/proto/api/src/proto"
)

// Sign
func (m *minerGRPCService) Sign(ctx context.Context, req *minerproto.SignRequest) (*minerproto.SignResponse, error) {
	privateKey := req.PrivateKey
	publicKey := req.PublicKey
	data := req.Data
	timestamp := req.TimeStamp
	key, err := hex.DecodeString(publicKey)
	if err != nil {
		return nil, err
	}
	clientID := encryption.Hash(key)
	var hashdata string
	if timestamp != "" {
		hashdata = fmt.Sprintf("%v:%v:%v", clientID, timestamp, data)
	} else {
		hashdata = fmt.Sprintf("%v:%v", clientID, data)
	}
	hash := encryption.Hash(hashdata)
	signature, err := encryption.Sign(privateKey, hash)
	if err != nil {
		return nil, err
	}

	return &minerproto.SignResponse{
		ClientId:  clientID,
		Hash:      hash,
		Signature: signature,
	}, nil
}
