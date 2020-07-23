// +build integration_tests

package sharder

import (
	"context"
	"net/http"

	"0chain.net/core/datastore"
	"0chain.net/core/persistencestore"

	"0chain.net/core/common"

	crpc "0chain.net/conductor/conductrpc"
)

func revertString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

/*TransactionConfirmationHandler - given a transaction hash, confirm it's presence in a block */
func TransactionConfirmationHandler(ctx context.Context, r *http.Request) (
	interface{}, error) {

	var hash = r.FormValue("hash")
	if hash == "" {
		return nil, common.InvalidRequest("transaction hash (parameter hash)" +
			" is required")
	}

	var content = r.FormValue("content")
	if content == "" {
		content = "confirmation"
	}

	var transactionConfirmationEntityMetadata = datastore.GetEntityMetadata(
		"txn_confirmation")
	ctx = persistencestore.WithEntityConnection(ctx,
		transactionConfirmationEntityMetadata)

	var (
		state             = crpc.Client().State()
		sc                = GetSharderChain()
		confirmation, err = sc.GetTransactionConfirmation(ctx, hash)
	)

	if confirmation != nil && state.VerifyTransaction != nil {
		confirmation.Hash = revertString(confirmation.Hash)
		confirmation.BlockHash = revertString(confirmation.BlockHash)
		confirmation.Round = confirmation.Round - 10
	}

	if content == "confirmation" {
		return confirmation, err
	}

	var data = make(map[string]interface{}, 2)
	if err == nil {
		data["confirmation"] = confirmation
	} else {
		data["error"] = err
	}

	if lfbSummary := sc.GetLatestFinalizedBlockSummary(); lfbSummary != nil {
		if state.VerifyTransaction != nil {
			lfbSummary.Hash = revertString(lfbSummary.Hash)
			lfbSummary.Round = lfbSummary.Round - 10
		}
		data["latest_finalized_block"] = lfbSummary
	}

	return data, nil
}
