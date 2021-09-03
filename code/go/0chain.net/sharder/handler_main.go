// +build !integration_tests

package sharder

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/core/datastore"
	"0chain.net/core/persistencestore"

	"0chain.net/core/common"
)

/*TransactionConfirmationHandler - given a transaction hash, confirm it's presence in a block */
func TransactionConfirmationHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("hash")
	fmt.Println("Hash", hash)
	if hash == "" {
		return nil, common.InvalidRequest("transaction hash (parameter hash) is required")
	}
	content := r.FormValue("content")
	if content == "" {
		content = "confirmation"
	}

	transactionConfirmationEntityMetadata := datastore.GetEntityMetadata("txn_confirmation")
	fmt.Println("txn meta", transactionConfirmationEntityMetadata)

	ctx = persistencestore.WithEntityConnection(ctx, transactionConfirmationEntityMetadata)
	fmt.Println("ctx", ctx)

	defer persistencestore.Close(ctx)

	sc := GetSharderChain()
	fmt.Println("Sharder chain", sc)
	confirmation, err := sc.GetTransactionConfirmation(ctx, hash)
	fmt.Println("confirmation", confirmation)

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
