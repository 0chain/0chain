//go:build integration_tests
// +build integration_tests

package sharder

import (
	"context"
	"net/http"

	"0chain.net/chaincore/chain"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/persistencestore"
	"0chain.net/core/util"
)

// SetupHandlers sets up the necessary API end points.
func SetupHandlers() {
	handlers := handlersMap()

	handlers["/v1/block/get"] = chain.BlockStats(
		handlers["/v1/block/get"],
		chain.BlockStatsConfigurator{
			HashKey: "block",
		},
	)

	setupHandlers(handlers)
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
	defer persistencestore.Close(ctx)

	var (
		state             = crpc.Client().State()
		sc                = GetSharderChain()
		confirmation, err = sc.GetTransactionConfirmation(ctx, hash)
	)

	if confirmation != nil && state.VerifyTransaction != nil {
		confirmation.Hash = util.RevertString(confirmation.Hash)
		confirmation.BlockHash = util.RevertString(confirmation.BlockHash)
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
			lfbSummary.Hash = util.RevertString(lfbSummary.Hash)
			lfbSummary.Round = lfbSummary.Round - 10
		}
		data["latest_finalized_block"] = lfbSummary
	}

	return data, nil
}
