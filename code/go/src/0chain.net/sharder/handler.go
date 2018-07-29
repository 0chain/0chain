package sharder

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"0chain.net/datastore"
	"0chain.net/persistencestore"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/diagnostics"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/block/get", common.ToJSONResponse(BlockHandler))
	http.HandleFunc("/v1/transaction/get/confirmation", common.ToJSONResponse(TransactionConfirmationHandler))
	http.HandleFunc("/v1/chain/get/stats", common.ToJSONResponse(ChainStatsHandler))
	http.HandleFunc("/_chain_stats", ChainStatsWriter)
}

/*BlockHandler - a handler to respond to block queries */
func BlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("block")
	content := r.FormValue("content")
	if content == "" {
		content = "header"
	}
	parts := strings.Split(content, ",")
	var err error
	var b *block.Block
	if hash == "" {
		return nil, common.InvalidRequest("Block hash is required")
	}
	b, err = chain.GetServerChain().GetBlock(ctx, hash)
	if err == nil {
		return chain.GetBlockResponse(b, parts)
	}
	sc := GetSharderChain()
	/*NOTE: We store DIR_ROUND_RANGE number of blocks in the same directory and that's a large number (10M).
	So, as long as people query the last 10M blocks most of the time, we only end up with 1 or 2 iterations.
	Anything older than that, there is a cost to query the database and get the round informatio anyway.
	*/
	for r := sc.LatestFinalizedBlock.Round; r > 0; r -= blockstore.DIR_ROUND_RANGE {
		b, err = sc.GetBlockFromStore(hash, r)
		if err != nil {
			return nil, err
		}
	}
	return chain.GetBlockResponse(b, parts)
}

/*ChainStatsHandler - a handler to provide block statistics */
func ChainStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := &GetSharderChain().Chain
	return diagnostics.GetStatistics(c, chain.FinalizationTimer, 1000000.0), nil
}

/*ChainStatsWriter - a handler to provide block statistics */
func ChainStatsWriter(w http.ResponseWriter, r *http.Request) {
	c := &GetSharderChain().Chain
	w.Header().Set("Content-Type", "text/html")
	diagnostics.WriteStatisticsCSS(w)
	fmt.Fprintf(w, "<h2>Block Finalization Statistics</h2>")
	diagnostics.WriteStatistics(w, c, chain.FinalizationTimer, 1000000.0)
}

/*TransactionConfirmationHandler - given a transaction hash, confirm it's presence in a block */
func TransactionConfirmationHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("hash")
	if hash == "" {
		return nil, common.InvalidRequest("transaction hash (parameter hash) is required")
	}
	transactionConfirmationEntityMetadata := datastore.GetEntityMetadata("txn_confirmation")
	ctx = persistencestore.WithEntityConnection(ctx, transactionConfirmationEntityMetadata)
	return GetTransactionConfirmation(ctx, hash)
}
