//go:build !integration_tests
// +build !integration_tests

package sharder

import (
	"context"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/persistencestore"
)

// SetupHandlers sets up the necessary API end points.
func SetupHandlers() {
	setupHandlers(handlersMap())
}

/*TransactionConfirmationHandler - given a transaction hash, confirm it's presence in a block */
func TransactionConfirmationHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("hash")
	if hash == "" {
		return nil, common.InvalidRequest("transaction hash (parameter hash) is required")
	}
	content := r.FormValue("content")
	if content == "" {
		content = "confirmation"
	}
	transactionConfirmationEntityMetadata := datastore.GetEntityMetadata("txn_confirmation")
	ctx = persistencestore.WithEntityConnection(ctx, transactionConfirmationEntityMetadata)
	defer persistencestore.Close(ctx)
	sc := GetSharderChain()
	confirmation, err := sc.GetTransactionConfirmation(ctx, hash)

	if content == "confirmation" {
		return confirmation, err
	}
	data := make(map[string]interface{}, 2)
	if err == nil {
		data["confirmation"] = confirmation
	} else {
		data["error"] = err
	}
	if lfbSummary := sc.GetLatestFinalizedBlockSummary(); lfbSummary != nil {
		data["latest_finalized_block"] = lfbSummary
	}
	return data, nil
}

func GetBlockSummaryHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("hash")
	bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	defer ememorystore.Close(bctx)
	return GetSharderChain().GetBlockSummary(bctx, hash)
}
