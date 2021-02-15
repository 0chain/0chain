package sharder

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/core/common"
)

/* SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/block/get", common.UserRateLimit(common.ToJSONResponse(BlockHandler)))
	http.HandleFunc("/v1/block/magic/get", common.UserRateLimit(common.ToJSONResponse(MagicBlockHandler)))
	http.HandleFunc("/v1/transaction/get/confirmation", common.UserRateLimit(common.ToJSONResponse(TransactionConfirmationHandler)))
	http.HandleFunc("/v1/chain/get/stats", common.UserRateLimit(common.ToJSONResponse(ChainStatsHandler)))
	http.HandleFunc("/_chain_stats", common.UserRateLimit(ChainStatsWriter))
	http.HandleFunc("/_health_check", common.UserRateLimit(HealthCheckWriter))
	http.HandleFunc("/v1/sharder/get/stats", common.UserRateLimit(common.ToJSONResponse(SharderStatsHandler)))
}

/*BlockHandler - a handler to respond to block queries */
func BlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	roundData := r.FormValue("round")
	hash := r.FormValue("block")
	content := r.FormValue("content")
	if content == "" {
		content = "header"
	}
	parts := strings.Split(content, ",")
	sc := GetSharderChain()
	lfb := sc.GetLatestFinalizedBlock()
	if roundData != "" {
		roundNumber, err := strconv.ParseInt(roundData, 10, 64)
		if err != nil {
			return nil, err
		}
		if roundNumber > lfb.Round {
			return nil, common.InvalidRequest("Block not available")
		}
		roundEntity := sc.GetSharderRound(roundNumber)
		if roundEntity == nil {
			roundEntity, err = sc.GetRoundFromStore(ctx, roundNumber)
			if err != nil {
				return nil, err
			}
		}
		hash = roundEntity.BlockHash

		hash, err = sc.GetBlockHash(ctx, roundNumber)
		if err != nil {
			return nil, err
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
	/*NOTE: We store chain.RoundRange number of blocks in the same directory and that's a large number (10M).
	So, as long as people query the last 10M blocks most of the time, we only end up with 1 or 2 iterations.
	Anything older than that, there is a cost to query the database and get the round information anyway.
	*/
	for roundEntity := lfb.Round; roundEntity > 0; roundEntity -= sc.RoundRange {
		b, err = sc.GetBlockFromStore(hash, roundEntity)
		if err != nil {
			return nil, err
		}
	}
	return chain.GetBlockResponse(b, parts)
}

/*MagicBlockHandler - a handler to respond to magic block queries */
func MagicBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	magicBlockNumber := r.FormValue("magic_block_number")
	sc := GetSharderChain()
	mbm, err := sc.GetMagicBlockMap(ctx, magicBlockNumber)
	if err != nil {
		return nil, err
	}
	b, err := chain.GetServerChain().GetBlock(ctx, mbm.Hash)
	if err != nil {
		lfb := sc.GetLatestFinalizedBlock()
		for roundEntity := lfb.Round; roundEntity > 0; roundEntity -= sc.RoundRange {
			b, err = sc.GetBlockFromStore(mbm.Hash, roundEntity)
			if err != nil {
				return nil, err
			}
		}
	}
	return b, nil
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

	self := node.Self.Underlying()
	fmt.Fprintf(w, "<div>%v - %v</div>", self.GetPseudoName(), self.Description)

	diagnostics.WriteConfiguration(w, c)
	fmt.Fprintf(w, "<br>")
	diagnostics.WriteCurrentStatus(w, c)
	fmt.Fprintf(w, "<br>")
	fmt.Fprintf(w, "<table><tr><td colspan='2'><h2>Summary</h2></td></tr>")
	fmt.Fprintf(w, "<tr><td>Sharded Blocks</td><td class='number'>%v</td></tr>", sc.SharderStats.ShardedBlocksCount)
	fmt.Fprintf(w, "<tr><td>QOS Round</td><td class='number'>%v</td></tr>", sc.SharderStats.QOSRound)
	fmt.Fprintf(w, "</table>")
	fmt.Fprintf(w, "<br>")
	fmt.Fprintf(w, "<table><tr><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Steady State)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.SteadyStateFinalizationTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Start to Finish)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td colspan='2'>")
	fmt.Fprintf(w, "<p>Steady state block finalization time = block generation + block processing + network time (1*large message + 2*small message)</p>")
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

	fmt.Fprintf(w, "<br>")
	if c.GetPruneStats() != nil {
		diagnostics.WritePruneStats(w, c.GetPruneStats())
	}
}

func SharderStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	bss := sc.BlockSyncStats
	cc := bss.getCycleControl(ProximityScan)
	previous := &cc.counters.previous
	var previousElapsed string
	if previous.CycleStart.IsZero() {
		previousElapsed = "n/a"
	} else {
		previousElapsed = previous.CycleDuration.Round(time.Second).String()
	}
	selfNodeInfo := node.Self.Underlying().Info
	return ExplorerStats{LastFinalizedRound: sc.Chain.GetLatestFinalizedBlock().Round,
		StateHealth:            selfNodeInfo.StateMissingNodes,
		AverageBlockSize:       selfNodeInfo.AvgBlockTxns,
		PrevInvocationCount:    previous.HealthCheckInvocations,
		PrevInvocationScanTime: previousElapsed,
		MeanScanBlockStatsTime: cc.BlockSyncTimer.Mean() / 1000000.0,
	}, nil
}
