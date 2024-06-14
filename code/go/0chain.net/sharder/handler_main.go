//go:build !integration_tests
// +build !integration_tests

package sharder

import (
	"context"
	"net/http"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

// SetupHandlers sets up the necessary API end points.
func SetupHandlers() {
	setupHandlers(handlersMap())
}

/*TransactionConfirmationHandler - given a transaction hash, confirm it's presence in a block */
// swagger:route GET /v1/transaction/get/confirmation sharder GetTransactionConfirmation
// Get transaction confirmation.
// Get the confirmation of the transaction from the sharders.
// If content == confirmation, only the confirmation is returned. Otherwise, the confirmation and the latest finalized block are returned.
//
// parameters:
//    +name: hash
//      in: query
//      required: true
//      type: string
//      description: Transaction hash
//    +name: content
//      in: query
//      required: false
//      type: string
//      description: confirmation or error
//      default: confirmation
//
// responses:
//    200: ConfirmationResponse
//    400:
func TransactionConfirmationHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("hash")
	if hash == "" {
		return nil, common.InvalidRequest("transaction hash (parameter hash) is required")
	}
	content := r.FormValue("content")
	if content == "" {
		content = "confirmation"
	}
	transactionSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	ctx = ememorystore.WithEntityConnection(ctx, transactionSummaryEntityMetadata)
	defer ememorystore.Close(ctx)
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
