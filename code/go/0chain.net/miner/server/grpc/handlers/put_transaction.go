package handlers

import (
	"context"
	"fmt"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	minerproto "0chain.net/miner/proto/api/src/proto"
)

// PutTransactionHandler is a gRPC handler for PutTransaction
func (s *minerGRPCService) PutTransaction(ctx context.Context, req *minerproto.PutTransactionRequest) (*minerproto.PutTransactionResponse, error) {
	txn := &transaction.Transaction{
		ClientID: req.Transaction.ClientId,
		// PublicKey: req.Transaction.Hash,
		ToClientID:        req.Transaction.ToClientId,
		ChainID:           req.Transaction.ChainId,
		TransactionData:   req.Transaction.TransactionData,
		Value:             req.Transaction.TransactionValue,
		Signature:         req.Transaction.Signature,
		CreationDate:      common.Timestamp(req.Transaction.CreationDate),
		Fee:               req.Transaction.TransactionFee,
		TransactionType:   int(req.Transaction.TransactionType),
		TransactionOutput: req.Transaction.TransactionOutput,
		OutputHash:        req.Transaction.TxnOutputHash,
		Status:            int(req.Transaction.TransactionStatus),
	}
	//
	if chain.GetServerChain().TxnMaxPayload > 0 {
		if len(txn.TransactionData) > chain.GetServerChain().TxnMaxPayload {
			s := fmt.Sprintf("transaction payload exceeds the max payload (%d)", chain.GetServerChain().TxnMaxPayload)
			return nil, common.NewError("txn_exceed_max_payload", s)
		}
	}

	// Calculate and update fee
	if err := txn.ValidateFee(); err != nil {
		return nil, err
	}

	//

	return &minerproto.PutTransactionResponse{
		Transaction: &minerproto.Transaction{
			ClientId:           txn.ClientID,
			ToClientId:         txn.ToClientID,
			ChainId:            txn.ChainID,
			TransactionData:    txn.TransactionData,
			TransactionValue:   txn.Value,
			Signature:          txn.Signature,
			CreationDate:       txn.CreationDate,
			TransactionFee:     txn.Fee,
			TransactionType:    minerproto.TransactionType(txn.TransactionType),
			TransactionOutput:  txn.TransactionOutput,
			TxnOutputHash:      txn.OutputHash,
			TransactionStatus:  minerproto.TransactionStatus(txn.Status),
			TransactionPayload: txn.TransactionPayload,
		},
	}, nil
}
