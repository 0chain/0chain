package sharder

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/node"
	"0chain.net/persistencestore"

	"0chain.net/block"
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
	round := r.FormValue("round")
	hash := r.FormValue("block")
	content := r.FormValue("content")
	if content == "" {
		content = "header"
	}
	parts := strings.Split(content, ",")
	if round != "" {
		roundNumber, err := strconv.ParseInt(round, 10, 64)
		if err != nil {
			return nil, err
		}
		sc := GetSharderChain()
		if roundNumber > sc.LatestFinalizedBlock.Round {
			return nil, common.InvalidRequest("Block not available")
		} else {
			r := sc.GetSharderRound(roundNumber)
			if r == nil {
				r, err = sc.GetRoundFromStore(ctx, roundNumber)
				if err != nil {
					return nil, err
				}
			}
			hash = r.BlockHash
		}
	}
	var err error
	var b *block.Block
	if hash == "" {
		return nil, common.InvalidRequest("Block hash or round number is required")
	}
	b, err = chain.GetServerChain().GetBlock(ctx, hash)
	if err == nil {
		return chain.GetBlockResponse(b, parts)
	}
	sc := GetSharderChain()
	/*NOTE: We store chain.RoundRange number of blocks in the same directory and that's a large number (10M).
	So, as long as people query the last 10M blocks most of the time, we only end up with 1 or 2 iterations.
	Anything older than that, there is a cost to query the database and get the round information anyway.
	*/
	for r := sc.LatestFinalizedBlock.Round; r > 0; r -= sc.RoundRange {
		b, err = sc.GetBlockFromStore(hash, r)
		if err != nil {
			return nil, err
		}
	}
	return chain.GetBlockResponse(b, parts)
}

/*ChainStatsHandler - a handler to provide block statistics */
func ChainStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetSharderChain().Chain
	return diagnostics.GetStatistics(c, chain.SteadyStateFinalizationTimer, 1000000.0), nil
}

/*ChainStatsWriter - a handler to provide block statistics */
func ChainStatsWriter(w http.ResponseWriter, r *http.Request) {
	sc := GetSharderChain()
	c := sc.Chain
	w.Header().Set("Content-Type", "text/html")
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Node
	fmt.Fprintf(w, "<div>%v - %v</div>", self.GetPseudoName(), self.Description)

	diagnostics.WriteConfiguration(w, c)
	fmt.Fprintf(w, "<br>")
	diagnostics.WriteCurrentStatus(w, c)
	fmt.Fprintf(w, "<br>")
	fmt.Fprintf(w, "<table><tr><td colspan='2'><h2>Summary</h2></td></tr>")
	fmt.Fprintf(w, "<tr><td>Sharded Blocks</td><td class='number'>%v</td>", sc.SharderStats.ShardedBlocksCount)
	fmt.Fprintf(w, "</table>")
	fmt.Fprintf(w, "<table><tr><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Steady State)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.SteadyStateFinalizationTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Start to Finish)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td col='2'>")
	fmt.Fprintf(w, "<p>Block finalization time = block generation + block verification + network time (1*large message + 2*small message)</p>")
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Txn Finalization Statistics (Start to Finish)</h2>")
	if config.Development() {
		diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTxnTimer, 1000000.0)
	} else {
		fmt.Fprintf(w, "Available only in development mode")
	}
	fmt.Fprintf(w, "</td><td  valign='top'>")
	fmt.Fprintf(w, "<h2>Finalization Lag Statistics</h2>")
	diagnostics.WriteHistogramStatistics(w, c, chain.FinalizationLagMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Transactions Save Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, txnSaveTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Save Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, blockSaveTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>State Save Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StateSaveTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h2>State Change Statistics</h2>")
	diagnostics.WriteHistogramStatistics(w, c, chain.StateChangeSizeMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>State Prune Update Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneUpdateTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>State Prune Delete Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneDeleteTimer, 1000000.0)
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "</table>")
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
