package miner

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"

	"0chain.net/chaincore/client"
	"0chain.net/core/memorystore"
)

/*SetupHandlers - setup miner handlers */
func SetupHandlers() {
	http.HandleFunc("/v1/chain/get/stats", common.UserRateLimit(common.ToJSONResponse(ChainStatsHandler)))
	http.HandleFunc("/_chain_stats", common.UserRateLimit(ChainStatsWriter))
	http.HandleFunc("/_diagnostics/wallet_stats", common.UserRateLimit(GetWalletStats))
}

/*ChainStatsHandler - a handler to provide block statistics */
func ChainStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetMinerChain().Chain
	return diagnostics.GetStatistics(c, chain.SteadyStateFinalizationTimer, 1000000.0), nil
}

//ChainStatsWriter - display the current chain stats
func ChainStatsWriter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	c := GetMinerChain().Chain
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Node
	fmt.Fprintf(w, "<div>%v - %v</div>", self.GetPseudoName(), self.Description)

	diagnostics.WriteConfiguration(w, c)
	fmt.Fprintf(w, "<br>")
	diagnostics.WriteCurrentStatus(w, c)
	fmt.Fprintf(w, "<br>")
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Steady state)</h2>")
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
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h2>Finalization Lag Statistics</h2>")
	diagnostics.WriteHistogramStatistics(w, c, chain.FinalizationLagMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Generation Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, bgTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Round Block Generation Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, rbgTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Processing Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, bpTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Verification Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, btvTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Txns Statistics</h2>")
	diagnostics.WriteHistogramStatistics(w, c, bsHistogram)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Smart Contract Execution Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.SmartContractExecutionTimer, 1000000.0)
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
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>RRS Generation Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, vrfTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "</table>")

	fmt.Fprintf(w, "<br>")
	if c.GetPruneStats() != nil {
		diagnostics.WritePruneStats(w, c.GetPruneStats())
	}
}

func GetWalletStats(w http.ResponseWriter, r *http.Request) {
	// clients
	chain.PrintCSS(w)
	walletsWithTokens, walletsWithoutTokens, totalWallets, round := GetWalletTable(false)
	fmt.Fprintf(w, "Wallet stats as of round %v\n", round)
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr><td>Wallets With Tokens</td><td>%v</td></tr>", walletsWithTokens)
	fmt.Fprintf(w, "<tr><td>Wallets Without Tokens</td><td>%v</td></tr>", walletsWithoutTokens)
	fmt.Fprintf(w, "<tr><td>Total Wallets</td><td>%v</td></tr>", totalWallets)
	fmt.Fprintf(w, "</table>")
	fmt.Fprintf(w, "<br>")
}

func GetWalletTable(latest bool) (int64, int64, int64, int64) {
	c := GetMinerChain().Chain
	entity := client.NewClient()
	emd := entity.GetEntityMetadata()
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), emd)
	collectionName := entity.GetCollectionName()
	mstore, ok := emd.GetStore().(*memorystore.Store)
	var b *block.Block
	if !ok {
		return 0, 0, 0, 0
	}
	if latest {
		b = c.GetRoundBlocks(c.CurrentRound - 1)[0]
	} else {
		b = c.LatestFinalizedBlock
	}
	var walletsWithTokens, walletsWithoutTokens, totalWallets int64
	walletsWithTokens = b.ClientState.GetNodeDB().Size(ctx)
	totalWallets = mstore.GetCollectionSize(ctx, emd, collectionName)
	walletsWithoutTokens = totalWallets - walletsWithTokens
	return walletsWithTokens, walletsWithoutTokens, totalWallets, b.Round
}
