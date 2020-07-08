package chain

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"runtime"
	"strings"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/metric"

	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/chain/get", common.Recover(common.ToJSONResponse(memorystore.WithConnectionHandler(GetChainHandler))))
	http.HandleFunc("/v1/chain/put", common.Recover(datastore.ToJSONEntityReqResponse(memorystore.WithConnectionEntityJSONHandler(PutChainHandler, chainEntityMetadata), chainEntityMetadata)))

	// Miner can only provide recent blocks, sharders can provide any block (for content other than full) and the block they store for full
	if node.Self.Underlying().Type == node.NodeTypeMiner {
		http.HandleFunc("/v1/block/get", common.UserRateLimit(common.ToJSONResponse(GetBlockHandler)))
	}
	http.HandleFunc("/v1/block/get/latest_finalized", common.UserRateLimit(common.ToJSONResponse(LatestFinalizedBlockHandler)))
	http.HandleFunc("/v1/block/get/latest_finalized_magic_block_summary", common.UserRateLimit(common.ToJSONResponse(LatestFinalizedMagicBlockSummaryHandler)))
	http.HandleFunc("/v1/block/get/latest_finalized_magic_block", common.UserRateLimit(common.ToJSONResponse(LatestFinalizedMagicBlockHandler)))
	http.HandleFunc("/v1/block/get/recent_finalized", common.UserRateLimit(common.ToJSONResponse(RecentFinalizedBlockHandler)))

	http.HandleFunc("/", common.UserRateLimit(HomePageHandler))
	http.HandleFunc("/_diagnostics", common.UserRateLimit(DiagnosticsHomepageHandler))
	http.HandleFunc("/_diagnostics/round_info", common.UserRateLimit(RoundInfoHandler))

	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	http.HandleFunc("/v1/transaction/put", common.UserRateLimit(datastore.ToJSONEntityReqResponse(datastore.DoAsyncEntityJSONHandler(memorystore.WithConnectionEntityJSONHandler(PutTransaction, transactionEntityMetadata), transaction.TransactionEntityChannel), transactionEntityMetadata)))

	http.HandleFunc("/_diagnostics/state_dump", common.UserRateLimit(StateDumpHandler))

	http.HandleFunc("/v1/block/get/latest_finalized_ticket", common.UserRateLimit(common.ToJSONResponse(LFBTicketHandler)))
}

/*GetChainHandler - given an id returns the chain information */
func GetChainHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, chainEntityMetadata, "id")
}

/*PutChainHandler - Given a chain data, it stores it */
func PutChainHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	return datastore.PutEntityHandler(ctx, entity)
}

/*GetMinersHandler - get the list of known miners */
func (c *Chain) GetMinersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	mb := c.GetCurrentMagicBlock()
	mb.Miners.Print(w)
}

/*GetShardersHandler - get the list of known sharders */
func (c *Chain) GetShardersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	mb := c.GetCurrentMagicBlock()
	mb.Sharders.Print(w)
}

/*GetBlockHandler - get the block from local cache */
func GetBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	hash := r.FormValue("block")
	content := r.FormValue("content")
	if content == "" {
		content = "header"
	}
	parts := strings.Split(content, ",")
	b, err := GetServerChain().GetBlock(ctx, hash)
	if err != nil {
		return nil, err
	}
	return GetBlockResponse(b, parts)
}

/*GetBlockResponse - a handler to get the block */
func GetBlockResponse(b *block.Block, contentParts []string) (interface{}, error) {
	data := make(map[string]interface{}, len(contentParts))
	for _, part := range contentParts {
		switch part {
		case "full":
			data["block"] = b
		case "header":
			data["header"] = b.GetSummary()
		case "merkle_tree":
			data["merkle_tree"] = b.GetMerkleTree().GetTree()
		}
	}
	return data, nil
}

/*LatestFinalizedMagicBlockSummaryHandler - provide the latest finalized magic block summary by this miner */
func LatestFinalizedMagicBlockSummaryHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetServerChain().GetLatestFinalizedMagicBlockSummary(), nil
}

/*RecentFinalizedBlockHandler - provide the latest finalized block by this miner */
func RecentFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	fbs := make([]*block.BlockSummary, 0, 10)
	for i, b := 0, GetServerChain().GetLatestFinalizedBlock(); i < 10 && b != nil; i, b = i+1, b.PrevBlock {
		fbs = append(fbs, b.GetSummary())
	}
	return fbs, nil
}

//StartTime - time when the server has started
var StartTime time.Time

/*HomePageHandler - provides basic info when accessing the home page of the server */
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	sc := GetServerChain()
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	PrintCSS(w)
	selfNode := node.Self.Underlying()
	fmt.Fprintf(w, "<div>I am %v working on the chain %v <ul><li>id:%v</li><li>public_key:%v</li><li>build_tag:%v</li></ul></div>\n",
		selfNode.GetPseudoName(), sc.GetKey(), selfNode.GetKey(), selfNode.PublicKey, build.BuildTag)
}

func (c *Chain) healthSummary(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<div>Health Summary</div>")
	c.healthSummaryInTables(w, r)
	fmt.Fprintf(w, "<div>&nbsp;</div>")
}

func (c *Chain) roundHealthInATable(w http.ResponseWriter, r *http.Request) {
	cr := c.GetRound(c.GetCurrentRound())

	vrfMsg := "N/A"
	notarizations := 0
	proposals := 0
	rrs := int64(0)

	mb := c.GetCurrentMagicBlock()

	if node.Self.Underlying().Type == node.NodeTypeMiner {
		var shares int
		check := "X"
		if cr != nil {

			shares = len(cr.GetVRFShares())
			notarizations = len(cr.GetNotarizedBlocks())
			proposals = len(cr.GetProposedBlocks())
			rrs = cr.GetRandomSeed()
		}

		thresholdByCount := config.GetThresholdCount()
		consensus := int(math.Ceil((float64(thresholdByCount) / 100) * float64(mb.Miners.Size())))
		if shares >= consensus {
			check = "&#x2714;"
		}
		vrfMsg = fmt.Sprintf("(%v/%v)%s", shares, consensus, check)
	}
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Current Round")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", c.GetCurrentRound())
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "VRFs")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", vrfMsg)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "RRS")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", rrs)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Proposals")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", proposals)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Notarizations")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", notarizations)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) chainHealthInATable(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Latest Finalized Round")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", c.GetLatestFinalizedBlock().Round)
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "Deterministic Finalized Round")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", c.LatestDeterministicBlock.Round)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Rollbacks")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", c.RollbackCount)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	cr := c.GetRound(c.GetCurrentRound())
	rtoc := c.GetRoundTimeoutCount()
	if cr != nil {
		rtoc = int64(cr.GetTimeoutCount())
	}
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Timeouts")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", c.RoundTimeoutsCount)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Round Timeout Count")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", rtoc)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) infraHealthInATable(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Go Routines")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", runtime.NumGoroutine())
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Heap Alloc")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", mstats.HeapAlloc)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "State missing nodes")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	ps := c.GetPruneStats()
	if ps != nil {
		fmt.Fprintf(w, "%v", ps.MissingNodes)
	} else {
		fmt.Fprintf(w, "pending")
	}
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")
	if node.Self.Underlying().Type == node.NodeTypeMiner {
		txn, ok := transaction.Provider().(*transaction.Transaction)
		if ok {
			transactionEntityMetadata := txn.GetEntityMetadata()
			collectionName := txn.GetCollectionName()
			ctx := common.GetRootContext()
			cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
			mstore, ok := transactionEntityMetadata.GetStore().(*memorystore.Store)
			if ok {
				fmt.Fprintf(w, "<tr class='active'>")
				fmt.Fprintf(w, "<td>")
				fmt.Fprintf(w, "Redis Collection")
				fmt.Fprintf(w, "</td>")
				fmt.Fprintf(w, "<td class='number'>")
				fmt.Fprintf(w, "%v", mstore.GetCollectionSize(cctx, transactionEntityMetadata, collectionName))
				fmt.Fprintf(w, "</td>")
				fmt.Fprintf(w, "</tr>")
			}
		}
	}
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) healthSummaryInTables(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<table class='menu' cellspacing='10' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td>Round Health</td><td>Chain Health</td><td>Infra Health</td></tr>")
	fmt.Fprintf(w, "<tr>")

	fmt.Fprintf(w, "<td valign='top'>")
	c.roundHealthInATable(w, r)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td valign='top'>")
	c.chainHealthInATable(w, r)
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "<td valign='top'>")
	c.infraHealthInATable(w, r)
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")

}

/*DiagnosticsHomepageHandler - handler to display the /_diagnostics page */
func DiagnosticsHomepageHandler(w http.ResponseWriter, r *http.Request) {
	sc := GetServerChain()
	HomePageHandler(w, r)
	fmt.Fprintf(w, "<div>Running since %v (%v) ...\n", StartTime.Format(common.DateTimeFormat), time.Since(StartTime))
	sc.healthSummary(w, r)
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td>Config</td><td>Stats</td><td>Info</td><td>Debug</td></tr>")
	fmt.Fprintf(w, "<tr>")
	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li><a href='/v1/config/get'>/v1/config/get</a></li>")
	selfNodeType := node.Self.Underlying().Type
	if selfNodeType == node.NodeTypeMiner && config.Development() {
		fmt.Fprintf(w, "<li><a href='/v1/config/update'>/v1/config/update</a></li>")
		fmt.Fprintf(w, "<li><a href='/v1/config/update_all'>/v1/config/update_all</a></li>")
	}
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li><a href='/_chain_stats'>/_chain_stats</a></li>")
	if selfNodeType == node.NodeTypeSharder {
		fmt.Fprintf(w, "<li><a href='/_health_check'>/_health_check</a></li>")
	}

	fmt.Fprintf(w, "<li><a href='/_diagnostics/miner_stats'>/_diagnostics/miner_stats</a>")
	if selfNodeType == node.NodeTypeMiner && config.Development() {
		fmt.Fprintf(w, "<li><a href='/_diagnostics/wallet_stats'>/_diagnostics/wallet_stats</a>")
	}
	fmt.Fprintf(w, "<li><a href='/_smart_contract_stats'>/_smart_contract_stats</a></li>")
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li><a href='/_diagnostics/info'>/_diagnostics/info</a> (with <a href='/_diagnostics/info?ts=1'>ts</a>)</li>")
	fmt.Fprintf(w, "<li><a href='/_diagnostics/n2n/info'>/_diagnostics/n2n/info</a></li>")
	if selfNodeType == node.NodeTypeMiner {
		//ToDo: For sharders show who all can store the blocks
		fmt.Fprintf(w, "<li><a href='/_diagnostics/round_info'>/_diagnostics/round_info</a>")
	}
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li>/_diagnostics/logs [Level <a href='/_diagnostics/logs?detail=1'>1</a>, <a href='/_diagnostics/logs?detail=2'>2</a>, <a href='/_diagnostics/logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li>/_diagnostics/n2n_logs [Level <a href='/_diagnostics/n2n_logs?detail=1'>1</a>, <a href='/_diagnostics/n2n_logs?detail=2'>2</a>, <a href='/_diagnostics/n2n_logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li>/_diagnostics/mem_logs [Level <a href='/_diagnostics/mem_logs?detail=1'>1</a>, <a href='/_diagnostics/mem_logs?detail=2'>2</a>, <a href='/_diagnostics/mem_logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li><a href='/debug/pprof/'>/debug/pprof/</a></li>")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")

	mb := sc.GetCurrentMagicBlock()
	if selfNodeType == node.NodeTypeMiner {
		fmt.Fprintf(w, "<div><div>Miners (%v) - median network time %.2f</div>", mb.Miners.Size(), mb.Miners.GetMedianNetworkTime()/1000000.)
	} else {
		fmt.Fprintf(w, "<div><div>Miners (%v)</div>", mb.Miners.Size())
	}
	sc.printNodePool(w, mb.Miners)
	fmt.Fprintf(w, "</div>")
	fmt.Fprintf(w, "<div><div>Sharders (%v)</div>", mb.Sharders.Size())
	sc.printNodePool(w, mb.Sharders)
	fmt.Fprintf(w, "</div>")
}

func (c *Chain) printNodePool(w http.ResponseWriter, np *node.Pool) {
	nodes := np.CopyNodes()
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td rowspan='2'>Set Index</td><td rowspan='2'>Node</td><td rowspan='2'>Sent</td><td rowspan='2'>Send Errors</td><td rowspan='2'>Received</td><td rowspan='2'>Last Active</td><td colspan='3' style='text-align:center'>Message Time</td><td rowspan='2'>Description</td><td colspan='4' style='text-align:center'>Remote Data</td></tr>")
	fmt.Fprintf(w, "<tr class='header'><td>Small</td><td>Large</td><td>Large Optimal</td><td>Build Tag</td><td>State Health</td><td title='median network time'>Miners MNT</td><td>Avg Block Size</td></tr>")
	r := c.GetRound(c.GetCurrentRound())
	hasRanks := r != nil && r.HasRandomSeed()
	lfb := c.GetLatestFinalizedBlock()
	for _, nd := range nodes {
		if nd.GetStatus() == node.NodeStatusInactive {
			fmt.Fprintf(w, "<tr class='inactive'>")
		} else {
			if node.Self.IsEqual(nd) && c.GetCurrentRound() > lfb.Round+10 {
				fmt.Fprintf(w, "<tr class='warning'>")
			} else {
				fmt.Fprintf(w, "<tr>")
			}
		}
		fmt.Fprintf(w, "<td>%d", nd.SetIndex)
		if nd.Type == node.NodeTypeMiner {
			if hasRanks && c.IsRoundGenerator(r, nd) {
				fmt.Fprintf(w, "<sup>%v</sup>", r.GetMinerRank(nd))
			}
		} else if nd.Type == node.NodeTypeSharder {
			if c.IsBlockSharder(lfb, nd) {
				fmt.Fprintf(w, "*")
			}
		}
		fmt.Fprintf(w, "</td>")
		if node.Self.IsEqual(nd) {
			fmt.Fprintf(w, "<td>%v</td>", nd.GetPseudoName())
		} else {
			fmt.Fprintf(w, "<td><a href='http://%v:%v/_diagnostics'>%v</a></td>", nd.Host, nd.Port, nd.GetPseudoName())
		}
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.Sent)
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.SendErrors)
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.Received)
		fmt.Fprintf(w, "<td>%v</td>", nd.GetLastActiveTime().Format(common.DateTimeFormat))
		fmt.Fprintf(w, "<td class='number'>%.2f</td>", nd.GetSmallMessageSendTimeSec())
		lmt := nd.GetLargeMessageSendTimeSec()
		fmt.Fprintf(w, "<td class='number'>%.2f</td>", lmt)
		olmt := nd.GetOptimalLargeMessageSendTime()
		if olmt < lmt {
			fmt.Fprintf(w, "<td class='number optimal'>%.2f</td>", olmt)

		} else {
			fmt.Fprintf(w, "<td class='number'>%.2f</td>", olmt)
		}
		fmt.Fprintf(w, "<td><div class='fixed-text' style='width:100px;' title='%s'>%s</div></td>", nd.Description, nd.Description)
		fmt.Fprintf(w, "<td><div class='fixed-text' style='width:100px;' title='%s'>%s</div></td>", nd.Info.BuildTag, nd.Info.BuildTag)
		if nd.Info.StateMissingNodes < 0 {
			fmt.Fprintf(w, "<td>pending</td>")
		} else {
			fmt.Fprintf(w, "<td class='number'>%v</td>", nd.Info.StateMissingNodes)
		}
		fmt.Fprintf(w, "<td class='number'>%v</td>", nd.Info.MinersMedianNetworkTime)
		fmt.Fprintf(w, "<td class='number'>%v</td>", nd.Info.AvgBlockTxns)
		fmt.Fprintf(w, "</tr>")
	}
	fmt.Fprintf(w, "</table>")
}

/*InfoHandler - handler to get the information of the chain */
func InfoHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	idx := 0
	chainInfo := chainMetrics.GetAll()
	for ; idx < len(chainInfo); idx++ {
		if chainInfo[idx].GetKey() == 0 {
			break
		}
	}
	info := make(map[string]interface{})
	info["chain_info"] = chainInfo[:idx]

	roundInfo := roundMetrics.GetAll()
	for idx = 0; idx < len(roundInfo); idx++ {
		if roundInfo[idx].GetKey() == 0 {
			break
		}
	}
	info["round_info"] = roundInfo[:idx]
	return info, nil
}

/*InfoWriter - a handler to get the information of the chain */
func InfoWriter(w http.ResponseWriter, r *http.Request) {
	PrintCSS(w)
	showTs := r.FormValue("ts") != ""
	fmt.Fprintf(w, "<style>\n")
	fmt.Fprintf(w, "tr:nth-child(10n + 3) { background-color: #abb2b9; }\n")
	fmt.Fprintf(w, "</style>")
	fmt.Fprintf(w, "<div>%v - %v</div>", node.Self.Underlying().GetPseudoName(),
		node.Self.Underlying().Description)
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr>")
	if showTs {
		fmt.Fprintf(w, "<td>Time</td>")
	}
	fmt.Fprintf(w, "<th>Round</th>")
	fmt.Fprintf(w, "<th>Chain Weight</th><th>Block Hash</th><th>Client State Hash</th><th>Blocks Count</th></tr>")
	chainInfo := chainMetrics.GetAll()
	for idx := 0; idx < len(chainInfo); idx++ {
		cf := chainInfo[idx].(*Info)
		if cf.FinalizedRound == 0 {
			break
		}
		fmt.Fprintf(w, "<tr>")
		if showTs {
			fmt.Fprintf(w, "<td class='number'>%v</td>", metric.FormattedTime(cf))
		}
		fmt.Fprintf(w, "<td class='number'>%11d</td>", cf.GetKey())
		fmt.Fprintf(w, "<td class='number'>%.8f</td>", cf.ChainWeight)
		fmt.Fprintf(w, "<td>%s</td>", cf.BlockHash)
		fmt.Fprintf(w, "<td>%v</td>", util.ToHex(cf.ClientStateHash))
		fmt.Fprintf(w, "<td class='number'>%11d</td>", cf.FinalizedCount)
		fmt.Fprintf(w, "</tr>")
	}
	fmt.Fprintf(w, "</table>")
	fmt.Fprintf(w, "<br/>")
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr>")
	if showTs {
		fmt.Fprintf(w, "<th>Time</th>")
	}
	fmt.Fprintf(w, "<th>Round</th>")
	fmt.Fprintf(w, "<th>Notarized Blocks</th><th>Multi Block Rounds</th><th>Zero Block Rounds</th><th>Missed Blocks</th><th>Rollbacks</th><th>Max Rollback Length</th></tr>")
	roundInfo := roundMetrics.GetAll()
	for idx := 0; idx < len(roundInfo); idx++ {
		rf := roundInfo[idx].(*round.Info)
		if rf.Number == 0 {
			break
		}
		fmt.Fprintf(w, "<tr>")
		if showTs {
			fmt.Fprintf(w, "<td class='number'>%v</td>", metric.FormattedTime(rf))
		}
		fmt.Fprintf(w, "<td class='number'>%d</td>", rf.GetKey())
		fmt.Fprintf(w, "<td class='number'>%d</td>", rf.NotarizedBlocksCount)
		fmt.Fprintf(w, "<td class='number'>%d</td>", rf.MultiNotarizedBlocksCount)
		fmt.Fprintf(w, "<td class='number'>%6d</td>", rf.ZeroNotarizedBlocksCount)
		fmt.Fprintf(w, "<td class='number'>%6d</td>", rf.MissedBlocks)
		fmt.Fprintf(w, "<td class='number'>%6d</td>", rf.RollbackCount)
		fmt.Fprintf(w, "<td class='number'>%6d</td>", rf.LongestRollbackLength)
		fmt.Fprintf(w, "</tr>")
	}
	fmt.Fprintf(w, "</table>")
}

//N2NStatsWriter - writes the n2n stats of all the nodes
func (c *Chain) N2NStatsWriter(w http.ResponseWriter, r *http.Request) {
	PrintCSS(w)
	fmt.Fprintf(w, "<div>%v - %v</div>", node.Self.Underlying().GetPseudoName(),
		node.Self.Underlying().Description)
	c.healthSummary(w, r)
	mb := c.GetCurrentMagicBlock()
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr><td rowspan='2'>URI</td><td rowspan='2'>Count</td><td colspan='3'>Time</td><td colspan='3'>Size</td></tr>")
	fmt.Fprintf(w, "<tr><td>Min</td><td>Average</td><td>Max</td><td>Min</td><td>Average</td><td>Max</td></tr>")
	fmt.Fprintf(w, "<tr><td colspan='8'>Miners (%v/%v) - median network time = %.2f", mb.Miners.GetActiveCount(), mb.Miners.Size(), mb.Miners.GetMedianNetworkTime()/1000000)
	for _, nd := range mb.Miners.CopyNodes() {
		if node.Self.IsEqual(nd) {
			continue
		}
		lmt := nd.GetLargeMessageSendTimeSec()
		olmt := nd.GetOptimalLargeMessageSendTime()
		cls := ""
		if !nd.IsActive() {
			cls = "inactive"
		}
		if olmt < lmt {
			cls = cls + " optimal"
		}
		if olmt >= mb.Miners.GetMedianNetworkTime() {
			cls = cls + " slow"
		}
		fmt.Fprintf(w, "<tr class='%s'><td colspan='8'><b>%s</b> (%.2f/%.2f) - %s</td></tr>", cls, nd.GetPseudoName(), olmt, lmt, nd.Description)
		nd.PrintSendStats(w)
	}

	fmt.Fprintf(w, "<tr><td colspan='8'>Sharders (%v/%v) - median network time = %.2f", mb.Sharders.GetActiveCount(), mb.Sharders.Size(), mb.Sharders.GetMedianNetworkTime()/1000000)
	for _, nd := range mb.Sharders.CopyNodes() {
		if node.Self.IsEqual(nd) {
			continue
		}
		lmt := nd.GetLargeMessageSendTimeSec()
		olmt := nd.GetOptimalLargeMessageSendTime()
		cls := ""
		if !nd.IsActive() {
			cls = "inactive"
		}
		if olmt < lmt {
			cls = cls + " optimal"
		}
		if olmt >= mb.Sharders.GetMedianNetworkTime() {
			cls = cls + " slow"
		}
		fmt.Fprintf(w, "<tr class='%s'><td colspan='8'><b>%s</b> (%.2f/%.2f) - %s </td></tr>", cls, nd.GetPseudoName(), olmt, lmt, nd.Description)
		nd.PrintSendStats(w)
	}
	fmt.Fprintf(w, "</table>")
}

/*PutTransaction - for validation of transactions using chain level parameters */
func PutTransaction(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	txn, ok := entity.(*transaction.Transaction)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", entity)
	}
	if GetServerChain().TxnMaxPayload > 0 {
		if len(txn.TransactionData) > GetServerChain().TxnMaxPayload {
			s := fmt.Sprintf("transaction payload exceeds the max payload (%d)", GetServerChain().TxnMaxPayload)
			return nil, common.NewError("txn_exceed_max_payload", s)
		}
	}
	return transaction.PutTransaction(ctx, txn)
}

//RoundInfoHandler collects and writes information about current round
func RoundInfoHandler(w http.ResponseWriter, r *http.Request) {
	PrintCSS(w)
	sc := GetServerChain()
	fmt.Fprintf(w, "<div class='bold'>Current Round Number: %v</div>", sc.GetCurrentRound())
	fmt.Fprintf(w, "<div>&nbsp;</div>")
	if node.Self.Underlying().Type != node.NodeTypeMiner {
		//ToDo: Add Sharder related round info
		return
	}
	cr := sc.GetRound(sc.GetCurrentRound())
	mb := sc.GetCurrentMagicBlock()
	if sc.GetCurrentRound() > 0 && cr != nil {

		rrs := int64(0)
		if cr.HasRandomSeed() {
			rrs = cr.GetRandomSeed()
		}
		thresholdByCount := config.GetThresholdCount()
		consensus := int(math.Ceil((float64(thresholdByCount) / 100) * float64(mb.Miners.Size())))

		fmt.Fprintf(w, "<div>Consensus: %v RRS: %v </div>", consensus, rrs)
		fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
		fmt.Fprintf(w, "<tr><th>Node</th><th>VRF</th></tr>")

		shares := cr.GetVRFShares()
		for _, share := range shares {
			n := share.GetParty()
			fmt.Fprintf(w, "<tr>")
			fmt.Fprintf(w, "<td valign='top' style='padding:2px'>")
			fmt.Fprintf(w, n.GetPseudoName())
			fmt.Fprintf(w, "</td>")
			fmt.Fprintf(w, "<td valign='top' style='padding:2px'>")
			fmt.Fprintf(w, "%v", share.Share)
			fmt.Fprintf(w, "</td>")
			fmt.Fprintf(w, "</tr>")
		}
		//ToDo: Add more RoundInfo
	}

}

/*MinerStatsHandler - handler for the miner stats */
func (c *Chain) MinerStatsHandler(w http.ResponseWriter, r *http.Request) {
	mb := c.GetCurrentMagicBlock()
	PrintCSS(w)
	fmt.Fprintf(w, "<div>%v - %v</div>", node.Self.Underlying().GetPseudoName(),
		node.Self.Underlying().Description)
	c.healthSummary(w, r)
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td colspan='3' style='text-align:center'>")
	c.notarizedBlockCountsStats(w)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<tr><th>Generation Counts</th><th>Verification Counts</th><th>Finalization Counts</th></tr>")
	fmt.Fprintf(w, "<tr><td>")
	c.generationCountStats(w)
	fmt.Fprintf(w, "</td><td>")
	c.verificationCountStats(w)
	fmt.Fprintf(w, "</td><td>")
	c.finalizationCountStats(w)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "</table>")
	if node.Self.Underlying().Type == node.NodeTypeMiner {
		fmt.Fprintf(w, "<br>")
		fmt.Fprintf(w, "<table>")
		fmt.Fprintf(w, "<tr><td>Miner</td><td>Verification Failures</td></tr>")
		for _, nd := range mb.Miners.CopyNodes() {
			ms := nd.ProtocolStats.(*MinerStats)
			fmt.Fprintf(w, "<tr><td>%v</td><td class='number'>%v</td></tr>", nd.GetPseudoName(), ms.VerificationFailures)
		}
		fmt.Fprintf(w, "</table>")
	}
}

func (c *Chain) generationCountStats(w http.ResponseWriter) {
	mb := c.GetCurrentMagicBlock()
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Miner</td>")
	for i := 0; i < c.NumGenerators; i++ {
		fmt.Fprintf(w, "<td>Rank %d</td>", i)
	}
	fmt.Fprintf(w, "<td>Total</td></tr>")
	totals := make([]int64, c.NumGenerators)
	for _, nd := range mb.Miners.CopyNodes() {
		fmt.Fprintf(w, "<tr><td>%v</td>", nd.GetPseudoName())
		ms := nd.ProtocolStats.(*MinerStats)
		var total int64
		for i := 0; i < c.NumGenerators; i++ {
			fmt.Fprintf(w, "<td class='number'>%v</td>", ms.GenerationCountByRank[i])
			totals[i] += ms.GenerationCountByRank[i]
			total += ms.GenerationCountByRank[i]
		}
		fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	}
	fmt.Fprintf(w, "<tr><td>Totals</td>")
	var total int64
	for i := 0; i < c.NumGenerators; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", totals[i])
		total += totals[i]
	}
	fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) verificationCountStats(w http.ResponseWriter) {
	mb := c.GetCurrentMagicBlock()
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Miner</td>")
	for i := 0; i < c.NumGenerators; i++ {
		fmt.Fprintf(w, "<td>Rank %d</td>", i)
	}
	fmt.Fprintf(w, "<td>Total</td></tr>")
	totals := make([]int64, c.NumGenerators)
	for _, nd := range mb.Miners.CopyNodes() {
		fmt.Fprintf(w, "<tr><td>%v</td>", nd.GetPseudoName())
		ms := nd.ProtocolStats.(*MinerStats)
		var total int64
		for i := 0; i < c.NumGenerators; i++ {
			fmt.Fprintf(w, "<td class='number'>%v</td>", ms.VerificationTicketsByRank[i])
			totals[i] += ms.VerificationTicketsByRank[i]
			total += ms.VerificationTicketsByRank[i]
		}
		fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	}
	fmt.Fprintf(w, "<tr><td>Totals</td>")
	var total int64
	for i := 0; i < c.NumGenerators; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", totals[i])
		total += totals[i]
	}
	fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) finalizationCountStats(w http.ResponseWriter) {
	mb := c.GetCurrentMagicBlock()
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Miner</td>")
	for i := 0; i < c.NumGenerators; i++ {
		fmt.Fprintf(w, "<td>Rank %d</td>", i)
	}
	fmt.Fprintf(w, "<td>Total</td></tr>")
	totals := make([]int64, c.NumGenerators)
	for _, nd := range mb.Miners.CopyNodes() {
		fmt.Fprintf(w, "<tr><td>%v</td>", nd.GetPseudoName())
		ms := nd.ProtocolStats.(*MinerStats)
		var total int64
		for i := 0; i < c.NumGenerators; i++ {
			fmt.Fprintf(w, "<td class='number'>%v</td>", ms.FinalizationCountByRank[i])
			totals[i] += ms.FinalizationCountByRank[i]
			total += ms.FinalizationCountByRank[i]
		}
		fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	}
	fmt.Fprintf(w, "<tr><td>Totals</td>")
	var total int64
	for i := 0; i < c.NumGenerators; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", totals[i])
		total += totals[i]
	}
	fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) notarizedBlockCountsStats(w http.ResponseWriter) {
	fmt.Fprintf(w, "<table style='width:100%%;'>")
	fmt.Fprintf(w, "<tr><td colspan='%v'>Rounds with notarized blocks (0 to %v)</td></tr>", c.NumGenerators+2, c.NumGenerators)
	fmt.Fprintf(w, "<tr><td>Notarized Blocks</td>")
	for i := 0; i <= c.NumGenerators; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", i)
	}
	fmt.Fprintf(w, "</tr><tr><td>Rounds</td>")
	for _, v := range c.NotariedBlocksCounts {
		fmt.Fprintf(w, "<td class='number'>%v</td>", v)
	}
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")
}

//PrintCSS - print the common css elements
func PrintCSS(w http.ResponseWriter) {
	fmt.Fprintf(w, "<style>\n")
	fmt.Fprintf(w, ".number { text-align: right; }\n")
	fmt.Fprintf(w, ".fixed-text { overflow:hidden;white-space: nowrap;word-break: break-all;word-wrap: break-word; text-overflow: ellipsis; }\n")
	fmt.Fprintf(w, ".menu li { list-style-type: none; }\n")
	fmt.Fprintf(w, "table, td, th { border: 1px solid black;  border-collapse: collapse;}\n")
	fmt.Fprintf(w, "tr.header { background-color: #E0E0E0;  }\n")
	fmt.Fprintf(w, ".inactive { background-color: #F44336; }\n")
	fmt.Fprintf(w, ".warning { background-color: #FFEB3B; }\n")
	fmt.Fprintf(w, ".optimal { color: #1B5E20; }\n")
	fmt.Fprintf(w, ".slow { font-style: italic; }\n")
	fmt.Fprintf(w, ".bold {font-weight:bold;}")
	fmt.Fprintf(w, "</style>")
}

//StateDumpHandler - a handler to dump the state
func StateDumpHandler(w http.ResponseWriter, r *http.Request) {
	c := GetServerChain()
	lfb := c.GetLatestFinalizedBlock()
	contract := r.FormValue("smart_contract")
	mpt := lfb.ClientState
	if contract == "" {
		contract = "global"
	} else {
		//TODO: get the smart contract as an optional parameter and pick the right state hash
	}
	mptRootHash := util.ToHex(mpt.GetRoot())
	fileName := fmt.Sprintf("mpt_%v_%v_%v.txt", contract, lfb.Round, mptRootHash)
	file, err := ioutil.TempFile("", fileName)
	if err != nil {
		return
	}
	go func() {
		writer := bufio.NewWriter(file)
		defer func() {
			writer.Flush()
			file.Close()
		}()
		fmt.Fprintf(writer, "round: %v\n", lfb.Round)
		fmt.Fprintf(writer, "global state hash: %v\n", util.ToHex(lfb.ClientStateHash))
		fmt.Fprintf(writer, "mpt state hash: %v\n", mptRootHash)
		writer.Flush()
		fmt.Fprintf(writer, "BEGIN {\n")
		mpt.PrettyPrint(writer)
		fmt.Fprintf(writer, "END }\n")
	}()
	fmt.Fprintf(w, "Writing to file : %v\n", file.Name())
}
