package miner

import (
	"context"
	"fmt"
	"net/http"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"

	"0chain.net/chaincore/client"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
)

/*SetupHandlers - setup miner handlers */
func SetupHandlers() {
	http.HandleFunc("/_chain_stats", common.UserRateLimit(ChainStatsHandler))
	http.HandleFunc("/_diagnostics/wallet_stats", common.UserRateLimit(GetWalletStats))
}

/*ChainStatsHandler - a handler to provide block statistics */
func ChainStatsHandler(w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintf(w, "<h2>Block Verification Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, btvTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Processing Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, bpTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
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

func GetWalletStats(w http.ResponseWriter, r *http.Request) {
	// clients
	chain.PrintCSS(w)
	c := GetMinerChain().Chain
	entity := client.NewClient()
	emd := entity.GetEntityMetadata()
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), emd)
	collectionName := entity.GetCollectionName()
	lfb := c.LatestFinalizedBlock
	mstore, ok := emd.GetStore().(*memorystore.Store)
	if !ok {
		return
	}
	fmt.Fprintf(w, "Wallet stats as of round %v\n", lfb.Round)
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td>Client ID</td><td>Balance</td><td>Round</td></tr>")
	var handler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		cli, ok := qe.(*client.Client)
		if !ok {
			err := qe.Delete(ctx)
			if err != nil {
				fmt.Fprintf(w, "Error in deleting cli in redis: %v\n", err)
			}
		}
		lfb := c.LatestFinalizedBlock
		balance, err := c.GetState(lfb, cli.ID)
		if balance.Balance == 0 || err != nil {
			fmt.Fprintf(w, "<tr class='inactive'>")
		} else if balance.Balance < 10000000000 {
			fmt.Fprintf(w, "<tr class='warning'>")
		} else {
			fmt.Fprintf(w, "<tr>")
		}
		fmt.Fprintf(w, "<td>%v</td>", cli.ID)
		fmt.Fprintf(w, "<td>%v</td>", balance.Balance)
		fmt.Fprintf(w, "<td>%v</td>", balance.Round)
		return true
	}
	err := mstore.IterateCollectionAsc(ctx, emd, collectionName, handler)
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
	}

}
