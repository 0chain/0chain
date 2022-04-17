package zcnsc

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/benchmark"

	"0chain.net/core/encryption"
)

// How authorizer signs the message
type proofOfBurn struct {
	TxnID             string                    `json:"ethereum_txn_id"`
	Amount            int64                     `json:"amount"`
	ReceivingClientID string                    `json:"receiving_client_id"` // 0ZCN address
	Nonce             int64                     `json:"nonce"`
	Signature         string                    `json:"signature"`
	Scheme            benchmark.SignatureScheme `json:"-"`
}

func (pb *proofOfBurn) Hash() string {
	return encryption.Hash(fmt.Sprintf("%v:%v:%v:%v", pb.TxnID, pb.Amount, pb.Nonce, pb.ReceivingClientID))
}

func (pb *proofOfBurn) sign(privateKey string) (err error) {
	pb.Scheme.SetPrivateKey(privateKey)
	pb.Signature, err = pb.Scheme.Sign(pb.Hash())

	return
}

func (pb *proofOfBurn) verifySignature(publicKey string) error {
	err := pb.Scheme.SetPublicKey(publicKey)
	if err != nil {
		return errors.New("failed to set public key")
	}

	ok, err := pb.Scheme.Verify(pb.Signature, pb.Hash())
	if err != nil || !ok {
		return errors.New("failed to verify signature in benchmarks")
	}

	return nil
}
