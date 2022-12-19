package miner

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	http.HandleFunc("/v1/miner/get/stats", common.UserRateLimit(common.ToJSONResponse(MinerStatsHandler)))
}

// swagger:route GET /v1/chain/get/stats chainstatus
// a handler to provide block statistics
//
// responses:
//  200: ChainStats
//  500: Internal Server Error

func ChainStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetMinerChain().Chain
	return diagnostics.GetStatistics(c, chain.SteadyStateFinalizationTimer, 1000000.0), nil
}

// ChainStatsWriter - display the current chain stats
func ChainStatsWriter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	c := GetMinerChain().Chain
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Underlying()
	fmt.Fprintf(w, "<h2>%v - %v</h2>", self.GetPseudoName(), self.Description)
	fmt.Fprintf(w, "<br>")

	fmt.Fprintf(w, "<table>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Configuration <a href='v1/config/get'>...</a></h3>")
	diagnostics.WriteConfiguration(w, c)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Current Status</h3>")
	diagnostics.WriteCurrentStatus(w, c)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Finalization Statistics (Steady state)</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.SteadyStateFinalizationTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Block Finalization Statistics (Start to Finish)</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td colspan='2'>")
	fmt.Fprintf(w, "<p>Steady state block finalization time = block generation + block processing + network time (1*large message + 2*small message)</p>")
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Txn Finalization Statistics (Start to Finish)</h3>")
	if config.Development() {
		diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTxnTimer, 1000000.0)
	} else {
		fmt.Fprintf(w, "Available only in development mode")
	}
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Finalization Lag Statistics</h3>")
	diagnostics.WriteHistogramStatistics(w, c, chain.FinalizationLagMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Generation Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, bgTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Round Block Generation Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, rbgTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Processing Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, bpTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Block Verification Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, btvTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>Block Txns Statistics</h3>")
	diagnostics.WriteHistogramStatistics(w, c, bsHistogram)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>Smart Contract Execution Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.SmartContractExecutionTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>State Save Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, block.StateSaveTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>State Change Statistics</h3>")
	diagnostics.WriteHistogramStatistics(w, c, block.StateChangeSizeMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>State Prune Update Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneUpdateTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h3>State Prune Delete Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneDeleteTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h3>RRS Generation Statistics</h3>")
	diagnostics.WriteTimerStatistics(w, c, vrfTimer, 1000000.0)
	if c.GetPruneStats() != nil {
		fmt.Fprintf(w, "</td><td valign='top'>")
		fmt.Fprintf(w, "<h3>Prune Stats</h3>")
		diagnostics.WritePruneStats(w, c.GetPruneStats())
	}
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "</table>")
}

// GetWalletStats -
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

// GetWalletTable -
func GetWalletTable(latest bool) (int64, int64, int64, int64) {
	c := GetMinerChain().Chain
	entity := client.NewClient()
	emd := entity.GetEntityMetadata()
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), emd)
	defer memorystore.Close(ctx)
	collectionName := entity.GetCollectionName()
	mstore, ok := emd.GetStore().(*memorystore.Store)
	var b *block.Block
	if !ok {
		return 0, 0, 0, 0
	}
	if latest {
		b = c.GetRoundBlocks(c.GetCurrentRound() - 1)[0]
	} else {
		b = c.GetLatestFinalizedBlock()
	}
	var walletsWithTokens, walletsWithoutTokens, totalWallets int64
	walletsWithTokens = b.ClientState.GetNodeDB().Size(ctx)
	totalWallets = mstore.GetCollectionSize(ctx, emd, collectionName)
	walletsWithoutTokens = totalWallets - walletsWithTokens
	return walletsWithTokens, walletsWithoutTokens, totalWallets, b.Round
}

func MinerStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	c := GetMinerChain().Chain
	var total int64
	ms := node.Self.Underlying().ProtocolStats.(*chain.MinerStats)
	for i := 0; i < c.GetGeneratorsNum(); i++ {
		total += ms.FinalizationCountByRank[i]
	}
	cr := c.GetRound(c.GetCurrentRound())
	rtoc := c.GetRoundTimeoutCount()
	if cr != nil {
		rtoc = int64(cr.GetTimeoutCount())
	}
	networkTimes := make(map[string]time.Duration)
	mb := c.GetCurrentMagicBlock()
	for k, v := range mb.Miners.CopyNodesMap() {
		networkTimes[k] = v.Info.MinersMedianNetworkTime
	}
	for k, v := range mb.Sharders.CopyNodesMap() {
		networkTimes[k] = v.Info.MinersMedianNetworkTime
	}

	return ExplorerStats{BlockFinality: chain.SteadyStateFinalizationTimer.Mean() / 1000000.0,
		LastFinalizedRound: c.GetLatestFinalizedBlock().Round,
		BlocksFinalized:    total,
		StateHealth:        node.Self.Underlying().Info.StateMissingNodes,
		CurrentRound:       c.GetCurrentRound(),
		RoundTimeout:       rtoc,
		Timeouts:           c.RoundTimeoutsCount,
		AverageBlockSize:   node.Self.Underlying().Info.AvgBlockTxns,
		NetworkTime:        networkTimes,
	}, nil
}
