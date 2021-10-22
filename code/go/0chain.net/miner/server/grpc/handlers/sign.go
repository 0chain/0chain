package handlers

import (
	"context"
	"encoding/hex"
	"fmt"

	"0chain.net/core/encryption"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/0chain/errors"
)

// Sign
func (m *minerGRPCService) Sign(ctx context.Context, req *minerproto.SignRequest) (*minerproto.SignResponse, error) {
	key, err := hex.DecodeString(req.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode string")
	}
	clientID := encryption.Hash(key)
	var hashdata string
	if req.TimeStamp != "" {
		hashdata = fmt.Sprintf("%v:%v:%v", clientID, req.TimeStamp, req.Data)
	} else {
		hashdata = fmt.Sprintf("%v:%v", clientID, req.Data)
	}
	hash := encryption.Hash(hashdata)
	signature, err := encryption.Sign(req.PrivateKey, hash)
	if err != nil {
		return nil, errors.Wrap(err, "unable to sign data")
	}

	return &minerproto.SignResponse{
		ClientId:  clientID,
		Hash:      hash,
		Signature: signature,
	}, nil
}
