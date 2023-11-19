//go:build integration_tests
// +build integration_tests

package chain

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	crpc "0chain.net/conductor/conductrpc"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type TxnHandler func(txn *transaction.Transaction, client *crpc.Entity) error

var txnHandlers = map[string]TxnHandler{
	"generate_challenge": func(txn *transaction.Transaction, client *crpc.Entity) error {
		client.ChallengeGenerated(txn.Hash)
		return nil
	},
	"challenge_response": func(txn *transaction.Transaction, client *crpc.Entity) error {
		switch txn.TransactionOutput {
		case "challenge passed by blobber":
			status := 0
			if txn.Status == 1 {
				status = 1
			}
			client.SendChallengeStatus(map[string]interface{}{
				"blobber_id": txn.ClientID,
				"status":     status,
			})
		case "Challenge Failed by Blobber":
			client.SendChallengeStatus(map[string]interface{}{
				"error":      txn.TransactionData,
				"status":     0,
				"response":   txn.TransactionOutput,
				"blobber_id": txn.ClientID,
			})
		}
		return nil
	},
}

func (c *Chain) postFinalize(ctx context.Context, fb *block.Block) error {
	client := crpc.Client()
	for _, txn := range fb.Txns {
		handler, ok := txnHandlers[txn.FunctionName]
		if !ok {
			continue
		}
		logging.Logger.Info("post_finalize processing txn",
			zap.Any("function_name", txn.FunctionName),
			zap.Any("hash", txn.Hash),
			zap.Any("output", txn.TransactionOutput),
		)
		err := handler(txn, client)
		if err != nil {
			logging.Logger.Error("post_finalize txn error",
				zap.Int64("round", fb.Round),
				zap.String("hash", fb.Hash),
				zap.Any("txn", txn),
				zap.Error(err),
			)
		}
	}

	return nil
}
