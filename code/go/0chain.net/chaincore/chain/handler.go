package chain

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"net/http"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/metric"
	"go.uber.org/zap"

	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"

	"0chain.net/core/logging"

	"0chain.net/smartcontract/minersc"
)

const (
	getBlockV1Pattern = "/v1/block/get"
)

func handlersMap(c Chainer) map[string]func(http.ResponseWriter, *http.Request) {
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	m := map[string]func(http.ResponseWriter, *http.Request){
		"/v1/chain/get": common.Recover(
			common.ToJSONResponse(
				memorystore.WithConnectionHandler(
					GetChainHandler,
				),
			),
		),
		"/v1/chain/put": common.Recover(
			datastore.ToJSONEntityReqResponse(
				memorystore.WithConnectionEntityJSONHandler(PutChainHandler, chainEntityMetadata),
				chainEntityMetadata,
			),
		),
		"/v1/block/get/latest_finalized": common.UserRateLimit(
			common.ToJSONResponse(
				LatestFinalizedBlockHandler,
			),
		),
		"/v1/block/get/latest_finalized_magic_block_summary": common.UserRateLimit(
			common.ToJSONResponse(
				LatestFinalizedMagicBlockSummaryHandler,
			),
		),
		"/v1/block/get/latest_finalized_magic_block": common.UserRateLimit(
			common.ToJSONResponse(
				LatestFinalizedMagicBlockHandler(c),
			),
		),
		"/v1/block/get/recent_finalized": common.UserRateLimit(
			common.ToJSONResponse(
				RecentFinalizedBlockHandler,
			),
		),
		"/v1/block/get/fee_stats": common.UserRateLimit(
			common.ToJSONResponse(
				LatestBlockFeeStatsHandler,
			),
		),
		"/": common.UserRateLimit(
			HomePageAndNotFoundHandler,
		),
		"/_diagnostics": common.UserRateLimit(
			DiagnosticsHomepageHandler,
		),
		"/_diagnostics/current_mb_nodes": common.UserRateLimit(
			DiagnosticsNodesHandler,
		),
		"/_diagnostics/dkg_process": common.UserRateLimit(
			DiagnosticsDKGHandler,
		),
		"/_diagnostics/round_info": common.UserRateLimit(
			RoundInfoHandler(c),
		),
		"/v1/transaction/put": common.UserRateLimit(
			datastore.ToJSONEntityReqResponse(
				datastore.DoAsyncEntityJSONHandler(
					memorystore.WithConnectionEntityJSONHandler(PutTransaction, transactionEntityMetadata),
					transaction.TransactionEntityChannel,
				),
				transactionEntityMetadata,
			),
		),
		"/_diagnostics/state_dump": common.UserRateLimit(
			StateDumpHandler,
		),
		"/v1/block/get/latest_finalized_ticket": common.N2NRateLimit(
			common.ToJSONResponse(
				LFBTicketHandler,
			),
		),
	}
	if node.Self.Underlying().Type == node.NodeTypeMiner {
		m[getBlockV1Pattern] = common.UserRateLimit(
			common.ToJSONResponse(
				GetBlockHandler,
			),
		)
	}

	return m
}

/*setupHandlers sets up the necessary API end points */
func setupHandlers(handlersMap map[string]func(http.ResponseWriter, *http.Request)) {
	for pattern, handler := range handlersMap {
		http.HandleFunc(pattern, common.WithCORS(handler))
	}
}

func DiagnosticsNodesHandler(w http.ResponseWriter, r *http.Request) {
	sc := GetServerChain()
	mb := sc.GetCurrentMagicBlock()
	d, err := json.MarshalIndent(append(mb.Sharders.CopyNodes(), mb.Miners.CopyNodes()...), "", "\t")
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}

	fmt.Fprint(w, string(d))
}

/*GetChainHandler - given an id returns the chain information */
func GetChainHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, chainEntityMetadata, "id")
}

func LatestBlockFeeStatsHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetServerChain().FeeStats, nil
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

/*RecentFinalizedBlockHandler - provide the latest finalized block by this miner */
func RecentFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	fbs := make([]*block.BlockSummary, 0, 10)
	for i, b := 0, GetServerChain().GetLatestFinalizedBlock(); i < 10 && b != nil; i, b = i+1, b.PrevBlock {
		fbs = append(fbs, b.GetSummary())
	}
	return fbs, nil
}

// StartTime - time when the server has started.
var StartTime time.Time

/*HomePageAndNotFoundHandler - catch all handler that returns home page for root path and 404 for other paths */
func HomePageAndNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		NotFoundPageHandler(w, r)
		return
	}

	HomePageHandler(w, r)
}

/*HomePageHandler - provides basic info when accessing the home page of the server */
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
	sc := GetServerChain()
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	PrintCSS(w)
	selfNode := node.Self.Underlying()
	fmt.Fprintf(w, "<div>I am %v working on the chain %v <ul><li>id:%v</li><li>public_key:%v</li><li>build_tag:%v</li></ul></div>\n",
		selfNode.GetPseudoName(), sc.GetKey(), selfNode.GetKey(), selfNode.PublicKey, build.BuildTag)
}

/*NotFoundPageHandler - provides the 404 page */
func NotFoundPageHandler(w http.ResponseWriter, r *http.Request) {
	common.Respond(w, r, nil, common.ErrNoResource)
}

func (c *Chain) healthSummary(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<div>Health Summary</div>")
	c.healthSummaryInTables(w, r)
	fmt.Fprintf(w, "<div>&nbsp;</div>")
}

func (c *Chain) roundHealthInATable(w http.ResponseWriter, r *http.Request) {
	var rn = c.GetCurrentRound()
	cr := c.GetRound(rn)

	vrfMsg := "N/A"
	notarizations := 0
	proposals := 0
	rrs := int64(0)
	phase := "N/A"
	var mb = c.GetMagicBlock(rn)

	if node.Self.Underlying().Type == node.NodeTypeMiner {
		var shares int
		check := "✗"
		if cr != nil {

			shares = len(cr.GetVRFShares())
			notarizations = len(cr.GetNotarizedBlocks())
			proposals = len(cr.GetProposedBlocks())
			rrs = cr.GetRandomSeed()
			phase = round.GetPhaseName(cr.GetPhase())
		}

		vrfThreshold := mb.T
		if shares >= vrfThreshold {
			check = "&#x2714;"
		}
		vrfMsg = fmt.Sprintf("(%v/%v)%s", shares, vrfThreshold, check)
	}
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Round")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "<a style='display:flex;' href='_diagnostics/round_info'><span style='flex:1;'></span>%d</a>", rn)
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

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Phase")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", phase)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	var (
		crn     = c.GetCurrentRound()
		ahead   = int64(config.GetLFBTicketAhead())
		tk      = c.GetLatestLFBTicket(r.Context())
		tkRound int64
		class   = "active"
	)

	if tk != nil {
		tkRound = tk.Round

		if tkRound+ahead <= crn {
			class = "inactive"
		}
	}

	fmt.Fprintf(w, "<tr class='"+class+"'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "LFB Ticket")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v", tkRound)
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
	fmt.Fprintf(w, "<td>")
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

	var rn = c.GetCurrentRound()
	cr := c.GetRound(rn)
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

	var (
		mb            = c.GetMagicBlock(rn)
		fmb           = c.GetLatestFinalizedMagicBlockRound(rn)
		startingRound int64
	)
	if fmb != nil {
		startingRound = fmb.StartingRound
	}

	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Related MB / finalized MB")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%v / %v", mb.StartingRound, startingRound)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "</table>")
}

func yn(t bool) string {
	if t {
		return "Y"
	}
	return "N"
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
	if snt := node.Self.Underlying().Type; snt == node.NodeTypeMiner {
		txn, ok := transaction.Provider().(*transaction.Transaction)
		if ok {
			transactionEntityMetadata := txn.GetEntityMetadata()
			collectionName := txn.GetCollectionName()
			ctx := common.GetRootContext()
			cctx := memorystore.WithEntityConnection(ctx, transactionEntityMetadata)
			defer memorystore.Close(cctx)
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

		var lfb = c.GetLatestFinalizedBlock()

		fmt.Fprintf(w, "<tr class='active'>")
		fmt.Fprintf(w, "<td>")
		fmt.Fprintf(w, "LFB state (computed / initialized)")
		fmt.Fprintf(w, "</td>")
		fmt.Fprintf(w, "<td class='number'>")
		fmt.Fprintf(w, "%s / %s", yn(lfb.IsStateComputed()), yn(lfb.ClientState != nil))
		fmt.Fprintf(w, "</td>")
		fmt.Fprintf(w, "</tr>")

	} else if snt == node.NodeTypeSharder {
		var (
			lfb = c.GetLatestFinalizedBlock()
			pn  minersc.PhaseNode
			err = c.GetBlockStateNode(lfb, minersc.PhaseKey, &pn)

			phase    minersc.Phase = minersc.Unknown
			restarts int64         = -1
		)

		if err == nil {
			phase = pn.Phase
			restarts = pn.Restarts
		}

		fmt.Fprintf(w, "<tr class='active'>")
		fmt.Fprintf(w, "<td>")
		fmt.Fprintf(w, "DKG phase / restarts")
		fmt.Fprintf(w, "</td>")
		fmt.Fprintf(w, "<td class='number'>")
		if !c.ChainConfig.IsViewChangeEnabled() {
			fmt.Fprint(w, "DKG process disabled")
		} else {
			fmt.Fprintf(w, "%s / %d", phase.String(), restarts)
		}
		fmt.Fprintf(w, "</td>")
		fmt.Fprintf(w, "</tr>")
	}

	// add fetching statistics
	var (
		fqs = c.FetchStat(r.Context())
		fm  = config.AsyncBlocksFetchingMaxSimultaneousFromMiners()
		fs  = config.AsyncBlocksFetchingMaxSimultaneousFromSharders()
	)
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Fetching blocks from miners, sharders")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprintf(w, "%d / %d, %d / %d", fqs.Miners, fm, fqs.Sharders, fs)
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	// is active in chain pin
	fmt.Fprintf(w, "<tr class='active'>")
	fmt.Fprintf(w, "<td>")
	fmt.Fprintf(w, "Is active in chain")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td class='number'>")
	fmt.Fprint(w, boolString(c.IsActiveInChain()))
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")

	fmt.Fprintf(w, "</table>")
}

func trim(s string) string {
	if len(s) > 10 {
		return fmt.Sprintf("%.10s...", s)
	}
	return s
}

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}

func (c *Chain) blocksHealthInATable(w http.ResponseWriter, r *http.Request) {

	// formats
	const (
		row  = "<tr%s><td>%s</td><td class='number'>%s</td></tr>"
		info = "<span style='display:flex;'>%.10s -> %.5s<span style='flex:1;'></span>%s</span>"
		lkmb = "<tr class='grey'><td>LFMB</td><td class='number'>%d %.10s</td></tr>"
	)

	var (
		ctx  = r.Context()
		rn   = c.GetCurrentRound()
		cr   = c.GetRound(rn)
		lfb  = c.GetLatestFinalizedBlock()
		plfb = c.GetLocalPreviousBlock(ctx, lfb)
		lfmb = c.GetLatestMagicBlock()

		next [4]*block.Block // blocks after LFB
	)

	for i := range next {
		var r = c.GetRound(lfb.Round + 1 + int64(i))
		if r == nil {
			continue // no round, no block
		}
		var hnb = r.GetHeaviestNotarizedBlock()
		if hnb != nil {
			next[i] = hnb // keep the block
			continue
		}
		var pbs = r.GetProposedBlocks()
		if len(pbs) == 0 {
			continue
		}
		next[i] = pbs[0] // use first one
	}

	type blockName struct {
		name  string
		style string
		block *block.Block
	}

	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	for i, bn := range []blockName{
		{itoa(lfb.Round - 1), " class='green'", plfb},
		{"LFB", " class='green'", lfb},
		{itoa(lfb.Round + 1), "", next[0]},
		{itoa(lfb.Round + 2), "", next[1]},
		{itoa(lfb.Round + 3), "", next[2]},
		{itoa(lfb.Round + 4), "", next[3]},
	} {
		if i == 5 && node.Self.Underlying().Type == node.NodeTypeMiner {
			continue
		}
		var hash = "-"
		if bn.block != nil {
			hash = fmt.Sprintf(info, bn.block.Hash,
				bn.block.PrevHash,
				boolString(bn.block.IsBlockNotarized()))
		}
		fmt.Fprintf(w, row, bn.style, bn.name, hash)
	}

	if node.Self.Underlying().Type == node.NodeTypeMiner {
		var blockHash string
		var numVerificationTickets int
		if cr != nil {
			b := cr.GetBestRankedProposedBlock()
			if b != nil {
				blockHash = b.Hash
				numVerificationTickets = len(b.GetVerificationTickets())
			}
		}
		consensus := int(math.Ceil((float64(config.GetThresholdCount()) / 100) * float64(lfmb.Miners.Size())))

		bvts := fmt.Sprintf("<span style='display:flex;'>%.10s<span style='flex:1;'></span>(%v/%v)%s</span>",
			blockHash, numVerificationTickets, consensus, boolString(numVerificationTickets > consensus))
		fmt.Fprintf(w, "<tr class='green'><td>CRB</td><td>%v</td></tr>", bvts)

	}

	// latest known magic block (finalized)
	fmt.Fprintf(w, lkmb, lfmb.StartingRound, lfmb.Hash)
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) healthSummaryInTables(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<table class='menu' cellspacing='10' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td>Round Health</td><td>Chain Health</td><td>Infra Health</td><td>Blocks</td></tr>")
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

	fmt.Fprintf(w, "<td valign='top'>")
	c.blocksHealthInATable(w, r)
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")

}

/*DiagnosticsHomepageHandler - handler to display the /_diagnostics page */
func DiagnosticsHomepageHandler(w http.ResponseWriter, r *http.Request) {
	sc := GetServerChain()
	isJSON := r.Header.Get("Accept") == "application/json"
	if isJSON {
		JSONHandler(w, r)
		return
	}
	HomePageHandler(w, r)
	fmt.Fprintf(w, "<div>Running since %v (%v) ...\n", StartTime.Format(common.DateTimeFormat), time.Since(StartTime))
	sc.healthSummary(w, r)
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td>Config</td><td>Stats</td><td>Info</td><td>Debug</td></tr>")
	fmt.Fprintf(w, "<tr>")
	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li><a href='v1/config/get'>/v1/config/get</a></li>")
	selfNodeType := node.Self.Underlying().Type
	if node.NodeType(selfNodeType) == node.NodeTypeMiner && config.Development() {
		fmt.Fprintf(w, "<li><a href='v1/config/update'>/v1/config/update</a></li>")
		fmt.Fprintf(w, "<li><a href='v1/config/update_all'>/v1/config/update_all</a></li>")
	}
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li><a href='_chain_stats'>/_chain_stats</a></li>")
	if node.NodeType(selfNodeType) == node.NodeTypeSharder {
		fmt.Fprintf(w, "<li><a href='_healthcheck'>/_healthcheck</a></li>")
	}

	fmt.Fprintf(w, "<li><a href='_diagnostics/miner_stats'>/_diagnostics/miner_stats</a>")
	if node.NodeType(selfNodeType) == node.NodeTypeMiner && config.Development() {
		fmt.Fprintf(w, "<li><a href='_diagnostics/wallet_stats'>/_diagnostics/wallet_stats</a>")
	}
	fmt.Fprintf(w, "<li><a href='_smart_contract_stats'>/_smart_contract_stats</a></li>")
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li><a href='_diagnostics/info'>/_diagnostics/info</a> (with <a href='_diagnostics/info?ts=1'>ts</a>)</li>")
	fmt.Fprintf(w, "<li><a href='_diagnostics/n2n/info'>/_diagnostics/n2n/info</a></li>")
	if node.NodeType(selfNodeType) == node.NodeTypeMiner {
		//ToDo: For sharders show who all can store the blocks
		fmt.Fprintf(w, "<li><a href='_diagnostics/round_info'>/_diagnostics/round_info</a>")
	}
	fmt.Fprintf(w, "<li><a href='_diagnostics/dkg_process'>/_diagnostics/dkg_process</a></li>")
	fmt.Fprintf(w, "</td>")

	fmt.Fprintf(w, "<td valign='top'>")
	fmt.Fprintf(w, "<li>/_diagnostics/logs [Level <a href='_diagnostics/logs?detail=1'>1</a>, <a href='_diagnostics/logs?detail=2'>2</a>, <a href='_diagnostics/logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li>/_diagnostics/n2n_logs [Level <a href='_diagnostics/n2n_logs?detail=1'>1</a>, <a href='_diagnostics/n2n_logs?detail=2'>2</a>, <a href='_diagnostics/n2n_logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li>/_diagnostics/mem_logs [Level <a href='_diagnostics/mem_logs?detail=1'>1</a>, <a href='_diagnostics/mem_logs?detail=2'>2</a>, <a href='_diagnostics/mem_logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li><a href='debug/pprof/'>/debug/pprof/</a></li>")
	fmt.Fprintf(w, "</td>")
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")

	mb := sc.GetCurrentMagicBlock()
	if selfNodeType == node.NodeTypeMiner {
		fmt.Fprintf(w, "<div><div>Miners (%v) - median network time %.2f - current MB start round: (%v)</div>", mb.Miners.Size(), mb.Miners.GetMedianNetworkTime()/1000000., mb.StartingRound)
	} else {
		fmt.Fprintf(w, "<div><div>Miners (%v)</div> - current MB starting round: (%v)", mb.Miners.Size(), mb.StartingRound)
	}
	sc.printNodePool(w, mb.Miners)
	fmt.Fprintf(w, "</div>")
	fmt.Fprintf(w, "<div><div>Sharders (%v)</div>", mb.Sharders.Size())
	sc.printNodePool(w, mb.Sharders)
	fmt.Fprintf(w, "</div>")
}

func (c *Chain) printNodePool(w http.ResponseWriter, np *node.Pool) {
	r := c.GetRound(c.GetCurrentRound())
	hasRanks := r != nil && r.HasRandomSeed()
	lfb := c.GetLatestFinalizedBlock()
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td rowspan='2'>Set Index</td><td rowspan='2'>Node</td><td rowspan='2'>Sent</td><td rowspan='2'>Send Errors</td><td rowspan='2'>Received</td><td rowspan='2'>Last Active</td><td colspan='3' style='text-align:center'>Message Time</td><td rowspan='2'>Description</td><td colspan='4' style='text-align:center'>Remote Data</td></tr>")
	fmt.Fprintf(w, "<tr class='header'><td>Small</td><td>Large</td><td>Large Optimal</td><td>Build Tag</td><td>State Health</td><td title='median network time'>Miners MNT</td><td>Avg Block Size</td></tr>")
	nodes := np.CopyNodes()
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].SetIndex < nodes[j].SetIndex
	})
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
			if len(nd.Path) > 0 {
				fmt.Fprintf(w, "<td><a href='https://%v/%v/_diagnostics'>%v</a></td>", nd.Host, nd.Path, nd.GetPseudoName())
			} else {
				fmt.Fprintf(w, "<td><a href='http://%v:%v/_diagnostics'>%v</a></td>", nd.Host, nd.Port, nd.GetPseudoName())
			}
		}
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.GetSent())
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.GetSendErrors())
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.GetReceived())
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

type dkgInfo struct {
	Phase        *minersc.PhaseNode
	AllMiners    *minersc.MinerNodes
	AllSharders  *minersc.MinerNodes
	DKGMiners    *minersc.DKGMinerNodes
	ShardersKeep *minersc.MinerNodes
	MPKs         *block.Mpks
	GSoS         *block.GroupSharesOrSigns //
	MB           *block.MagicBlock         // prepared magic block (miner SC MB)
	CMB          *block.MagicBlock         // current magic block
}

func boolString(t bool) string {
	if t {
		return "✔"
	}
	return "✗"
}

func (dkgi *dkgInfo) HasMPKs(id string) string {
	if dkgi.DKGMiners == nil || dkgi.DKGMiners.SimpleNodes == nil ||
		dkgi.MPKs == nil || dkgi.MPKs.Mpks == nil {
		return boolString(false)
	}
	if _, ok := dkgi.DKGMiners.SimpleNodes[id]; !ok {
		return boolString(false)
	}
	if _, ok := dkgi.MPKs.Mpks[id]; !ok {
		return boolString(false)
	}
	return boolString(true)
}

func (dkgi *dkgInfo) HasGSoS(id string) string {
	if dkgi.DKGMiners == nil || dkgi.DKGMiners.SimpleNodes == nil ||
		dkgi.GSoS == nil || dkgi.GSoS.Shares == nil {
		return boolString(false)
	}
	if _, ok := dkgi.DKGMiners.SimpleNodes[id]; !ok {
		return boolString(false)
	}
	if _, ok := dkgi.GSoS.Shares[id]; !ok {
		return boolString(false)
	}
	return boolString(true)
}

func (dkgi *dkgInfo) HasWait(id string) string {
	if dkgi.DKGMiners == nil || dkgi.DKGMiners.SimpleNodes == nil ||
		dkgi.DKGMiners.Waited == nil {
		return boolString(false)
	}
	if _, ok := dkgi.DKGMiners.SimpleNodes[id]; !ok {
		return boolString(false)
	}
	return boolString(dkgi.DKGMiners.Waited[id])
}

func (dkgi *dkgInfo) IsFromPrevSet(typ, id string) string {
	if dkgi.CMB == nil {
		return "unknown"
	}
	if typ == "miner" {
		return boolString(dkgi.CMB.Miners.HasNode(id))
	}
	return boolString(dkgi.CMB.Sharders.HasNode(id))
}

func (c *Chain) dkgInfo(cmb *block.MagicBlock) (dkgi *dkgInfo, err error) {

	dkgi = new(dkgInfo)

	dkgi.Phase = new(minersc.PhaseNode)
	dkgi.AllMiners = new(minersc.MinerNodes)
	dkgi.AllSharders = new(minersc.MinerNodes)
	dkgi.DKGMiners = new(minersc.DKGMinerNodes)
	dkgi.ShardersKeep = new(minersc.MinerNodes)
	dkgi.MPKs = new(block.Mpks)
	dkgi.GSoS = new(block.GroupSharesOrSigns)
	dkgi.MB = new(block.MagicBlock)
	dkgi.CMB = cmb

	var (
		lfb = c.GetLatestFinalizedBlock()
	)

	type keySeri struct {
		name string               // for errors
		key  string               // key
		inst util.MPTSerializable // instance
	}

	for _, ks := range []keySeri{
		{"phase", minersc.PhaseKey, dkgi.Phase},
		{"all_miners", minersc.AllMinersKey, dkgi.AllMiners},
		{"all_shardres", minersc.AllShardersKey, dkgi.AllSharders},
		{"dkg_miners", minersc.DKGMinersKey, dkgi.DKGMiners},
		{"sharder_keep", minersc.ShardersKeepKey, dkgi.ShardersKeep},
		{"mpks", minersc.MinersMPKKey, dkgi.MPKs},
		{"gsos", minersc.GroupShareOrSignsKey, dkgi.GSoS},
		{"MB", minersc.MagicBlockKey, dkgi.MB},
	} {
		err = c.GetBlockStateNode(lfb, ks.key, ks.inst)
		if err != nil {
			if err != util.ErrValueNotPresent {
				return nil, fmt.Errorf("can't get %s node: %v", ks.name, err)
			}

			err = nil // reset the error and leave the value blank
			continue
		}
	}

	return
}

func DiagnosticsDKGHandler(w http.ResponseWriter, r *http.Request) {
	c := GetServerChain()
	if !c.ChainConfig.IsViewChangeEnabled() {
		w.Header().Set("Content-Type", "text/html;charset=UTF-8")
		ss := []byte(`<doctype html><html><head>
<title>DKG process informations</title></head><body>
<h1>DKG process disabled</h1></body></html>`)

		if _, err := w.Write(ss); err != nil {
			logging.Logger.Error("diagnostics DKG handler - http write failed", zap.Error(err))
			return
		}
	}

	var (
		cmb       = c.GetCurrentMagicBlock()
		dkgi, err = c.dkgInfo(cmb)
	)

	if err != nil {
		http.Error(w, "error getting DKG info: "+err.Error(), 500)
		return
	}

	const templ = `
<doctype html>
<html>
<head>
  <title>DKG process informations</title>
    <style>
      .number {
      	text-align: right; }
      .fixed-text {
      	overflow: hidden;
      	white-space: nowrap;
      	word-break: break-all;
      	word-wrap: break-word;
      	text-overflow: ellipsis; }
      .menu li {
      	list-style-type: none; }
      table, td, th {
      	border: 1px solid black;
      	border-collapse: collapse;
        padding: .2em; }
      tr.header {
      	background-color: #E0E0E0; }
      .inactive {
      	background-color: #F44336; }
      .warning {
      	background-color: #FFEB3B; }
      .optimal {
      	color: #1B5E20; }
      .slow {
      	font-style: italic; }
      .bold {
      	font-weight:bold; }
    </style>
</head>
<body>
  <h1>DKG process information</h1>

  <p>
    <h3>Phase</h3>
    <table>
    <tr>
      <th>phase</th>
      <th>start round</th>
      <th>current round</th>
      <th>restarts</th>
    </tr>
    <tr>
      <td>{{ .Phase.Phase }}</td>
      <td>{{ .Phase.StartRound }}</td>
      <td>{{ .Phase.CurrentRound }}</td>
      <td>{{ .Phase.Restarts }}</td>
    </tr>
    </table>
  </p>

  <p>
    <h3>All registered miners</h3>
    {{ if .AllMiners.Nodes }}
      <table>
      <tr>
        <th>ID</th>
        <th>Host</th>
        <th>Total stake</th>
      </tr>
      {{ range $n := .AllMiners.Nodes }}
        <tr>
          <td>{{ trim $n.ID }}</td>
          <td>{{ $n.N2NHost }}</td>
          <td>{{ $n.TotalStaked }}</td>
        </tr>
      {{ end }}
      </table>
    {{ else }}
      no miners registered yet
    {{ end }}
  </p>

  <p>
    <h3>All registered sharders</h3>
    {{ if .AllSharders.Nodes }}
      <table>
      <tr>
        <th>ID</th>
        <th>Host</th>
        <th>Total stake</th>
      </tr>
      {{ range $n := .AllSharders.Nodes }}
        <tr>
          <td>{{ trim $n.ID }}</td>
          <td>{{ $n.N2NHost }}</td>
          <td>{{ $n.TotalStaked }}</td>
        </tr>
      {{ end }}
      </table>
    {{ else }}
      no sharders registered yet
    {{ end }}
  </p>

  <p>
    <h3>Sharders keep list</h3>
    {{ if .ShardersKeep.Nodes }}
      {{ $dot := . }}
      <table>
      <tr>
        <th>ID</th>
        <th>Host</th>
        <th>Total stake</th>
        <th>Is from previous set</th>
      </tr>
      {{ range $n := .ShardersKeep.Nodes }}
        <tr>
          <td>{{ trim $n.ID }}</td>
          <td>{{ $n.N2NHost }}</td>
          <td>{{ $n.TotalStaked }}</td>
          <td>{{ $dot.IsFromPrevSet "sharder" $n.ID }}</td>
        </tr>
      {{ end }}
      </table>
    {{ else }}
      empty list for now
    {{ end }}
  </p>

  <p>
    <h3>DKG miners</h3>
    <table>
    <tr>
      <th>T</th>
      <th>K</th>
      <th>N</th>
      <th>start round</th>
    </tr>
    <tr>
      <td>{{ .DKGMiners.T }}</td>
      <td>{{ .DKGMiners.K }}</td>
      <td>{{ .DKGMiners.N }}</td>
      <td>{{ .DKGMiners.StartRound }}</td>
     </tr>
    </table>
    {{ if .DKGMiners.SimpleNodes }}
      {{ $dot := . }}
      <table>
      <tr>
        <th>ID</th>
        <th>Host</th>
        <th>Total Staked</th>
        <th>MPKs</th>
        <th>GSoS</th>
        <th>Wait</th>
        <th>Is from previous set</th>
      </tr>
      {{ range $id, $val := .DKGMiners.SimpleNodes }}
        <tr>
          <td>{{ trim $id }}</td>
          <td>{{ $val.N2NHost }}</td>
          <td>{{ $val.TotalStaked }}</td>
          <td>{{ $dot.HasMPKs $id }}</td>
          <td>{{ $dot.HasGSoS $id }}</td>
          <td>{{ $dot.HasWait $id }}</td>
          <td>{{ $dot.IsFromPrevSet "miner" $id }}</td>
        </tr>
      {{ end }}
      </table>
    {{ else }}
      empty DKG miners list
    {{ end }}
  </p>

</body>
</html>
`

	var pt = template.New("root").Funcs(map[string]interface{}{
		"trim": trim,
		"typ": func(val interface{}) string {
			return fmt.Sprintf("%T", val)
		},
	})

	if pt, err = pt.Parse(templ); err != nil {
		http.Error(w, "parsing template error: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	if err = pt.Execute(w, dkgi); err != nil {
		http.Error(w, "executing template error: "+err.Error(), 500)
		return
	}
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
		return nil, fmt.Errorf("put_transaction: invalid request %T", entity)
	}

	sc := GetServerChain()
	if sc.TxnMaxPayload() > 0 {
		if len(txn.TransactionData) > sc.TxnMaxPayload() {
			s := fmt.Sprintf("transaction payload exceeds the max payload (%d)", GetServerChain().TxnMaxPayload())
			return nil, common.NewError("txn_exceed_max_payload", s)
		}
	}

	// Calculate and update fee
	if err := txn.ValidateFee(sc.ChainConfig.TxnExempt(), sc.ChainConfig.MinTxnFee()); err != nil {
		return nil, err
	}
	if err := txn.ValidateNonce(); err != nil {
		return nil, err
	}

	// save validated transactions to cache for miners only
	if node.Self.Underlying().Type == node.NodeTypeMiner {
		return transaction.PutTransaction(ctx, txn)
	}

	return transaction.PutTransaction(ctx, txn)
}

//RoundInfoHandler collects and writes information about current round
func RoundInfoHandler(c Chainer) common.ReqRespHandlerf {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				http.Error(w, fmt.Sprintf("<pre>%s</pre>", string(debug.Stack())), http.StatusInternalServerError)
			}
		}()

		roundParamQuery := ""

		rn := c.GetCurrentRound()
		roundParam := r.URL.Query().Get("round")
		if roundParam != "" {
			roundParamQuery = "?" + r.URL.RawQuery
			_rn, err := strconv.ParseInt(roundParam, 10, 64)
			if err != nil {
				http.Redirect(w, r, r.URL.Path, http.StatusTemporaryRedirect)
				return
			}
			rn = _rn
		}

		rnd := c.GetRound(rn)
		if rn == 0 || rnd == nil {
			http.Error(w, fmt.Sprintf("Round not found: round=%d", rn), http.StatusNotFound)
			return
		}

		PrintCSS(w)
		fmt.Fprintf(w, "<h3>Round: %v</h3>", rn)
		fmt.Fprintf(w, "<div>&nbsp;</div>")
		if node.Self.Underlying().Type != node.NodeTypeMiner {
			//ToDo: Add Sharder related round info
			return
		}

		mb := c.GetMagicBlock(rn)
		if mb == nil {
			lfmb := c.GetLatestFinalizedMagicBlockRound(rn)
			if lfmb != nil {
				mb = lfmb.MagicBlock
			}
		}
		if mb == nil {
			fmt.Fprintf(w, "<h3>MagicBlock not found for round %d</h3>", rn)
			return
		}

		rrs := int64(0)
		if rnd.HasRandomSeed() {
			rrs = rnd.GetRandomSeed()
		}
		thresholdByCount := config.GetThresholdCount()
		consensus := int(math.Ceil((float64(thresholdByCount) / 100) * float64(mb.Miners.Size())))

		fmt.Fprintf(w, "<table>")
		fmt.Fprintf(w, "<tr><td class='active'>Consensus</td><td class='number'>%d</td>", consensus)
		fmt.Fprintf(w, "<tr><td class='active'>Random Seed</td><td class='number'>%d</td>", rrs)
		fmt.Fprintf(w, "</table>")

		roundHasRanks := rnd != nil && rnd.HasRandomSeed()

		getNodeLink := func(n *node.Node) string {
			if node.Self.IsEqual(n) {
				return fmt.Sprintf("%v", n.GetPseudoName())
			}
			if len(n.Path) > 0 {
				return fmt.Sprintf("<a href='https://%v/%v/_diagnostics/round_info%s'>%v</a>", n.Host, n.Path, roundParamQuery, n.GetPseudoName())
			}
			return fmt.Sprintf("<a href='http://%v:%v/_diagnostics/round_info%s'>%v</a>", n.Host, n.Port, roundParamQuery, n.GetPseudoName())
		}

		// Verification and Notarization
		blocksMap := make(map[string]*block.Block)
		for _, b := range rnd.GetProposedBlocks() {
			blocksMap[b.Hash] = b
		}
		for _, b := range rnd.GetNotarizedBlocks() {
			blocksMap[b.Hash] = b
		}

		blocks := make([]*block.Block, 0, len(blocksMap))
		for _, b := range blocksMap {
			blocks = append(blocks, b)
		}

		if roundHasRanks {
			sort.SliceStable(blocks, func(i, j int) bool {
				b1, b2 := blocks[i], blocks[j]
				rank1, rank2 := math.MaxInt64, math.MaxInt64
				if m1 := mb.Miners.GetNode(b1.MinerID); m1 != nil {
					rank1 = rnd.GetMinerRank(m1)
				}
				if m2 := mb.Miners.GetNode(b2.MinerID); m2 != nil {
					rank2 = rnd.GetMinerRank(m2)
				}
				if rank1 == rank2 {
					return b1.RoundTimeoutCount > b2.RoundTimeoutCount ||
						b1.CreationDate > b2.CreationDate
				}
				return rank1 < rank2
			})
		}

		fmt.Fprintf(w, "<h3>Block Verification and Notarization</h3>")

		fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")

		fmt.Fprintf(w, "<tr class='header'>")
		fmt.Fprintf(w, "<th>SetIndex</th> <th>Generator</th> <th>RRS</th> <th>RTC</th> <th>Block</th> <th>Generated At (UTC)</th> <th>Verification</th> <th>Notarization</th>")
		fmt.Fprintf(w, "</tr>")

		for _, b := range blocks {
			fmt.Fprintf(w, "<tr><td>")

			n := mb.Miners.GetNode(b.MinerID)
			if n != nil {
				fmt.Fprintf(w, "%d", n.SetIndex)                   // SetIndex
				fmt.Fprintf(w, "</td><td>%s</td>", getNodeLink(n)) // Generator
			} else {
				fmt.Fprintf(w, "-")               // SetIndex
				fmt.Fprintf(w, "</td><td>-</td>") // Generator
			}

			fmt.Fprintf(w, "<td>%d (%s)</td>", b.RoundRandomSeed, boolString(b.RoundRandomSeed == rnd.GetRandomSeed())) // RRS
			fmt.Fprintf(w, "<td>%d</td>", b.RoundTimeoutCount)                                                          // RTC
			fmt.Fprintf(w, "<td title='%s'>%.8s</td>", b.Hash, b.Hash)                                                  // Block ID
			fmt.Fprintf(w, "<td>%s</td>", common.ToTime(b.CreationDate).UTC().Format("2006-01-02T15:04:05"))            // Block Creation Date

			tickets := b.GetVerificationTickets()

			fmt.Fprintf(w, "<td style='padding: 0px;'>")
			fmt.Fprintf(w, "<div style='display:flex;flex-direction:row;'>")
			fmt.Fprintf(w, "  <div style='flex:1;display:flex;flex-direction:column;padding:5px;min-width:60px;'>")
			fmt.Fprintf(w, "    <div style='flex:1;'></div><div>%d (%s)</div><div style='flex:1;'></div>", len(tickets), boolString(len(tickets) >= consensus))
			fmt.Fprintf(w, "  </div>")
			if len(tickets) > 0 {
				verifiers := make([]*node.Node, 0, len(tickets))
				for _, ticket := range tickets {
					verifiers = append(verifiers, mb.Miners.GetNode(ticket.VerifierID))
				}
				sortByVerifierSetIndex := func(i, j int) bool {
					v1, v2 := verifiers[i], verifiers[j]
					if v1 != nil && v2 != nil {
						return v1.SetIndex < v2.SetIndex
					}
					return v1 != nil || v2 == nil
				}
				sort.SliceStable(tickets, sortByVerifierSetIndex)
				sort.SliceStable(verifiers, sortByVerifierSetIndex)

				fmt.Fprintf(w, "<div style='display:flex;flex-direction:column;padding:5px;border-left:1px solid black;'>")
				for i, ticket := range tickets {
					if i%4 == 0 {
						if i > 0 {
							fmt.Fprintf(w, "</div>")
						}
						fmt.Fprintf(w, "<div style='display:flex;flex-direction:row;'>")
					}
					if n := verifiers[i]; n != nil {
						fmt.Fprintf(w, "<div title='%s'>%s</div>,", ticket.VerifierID, getNodeLink(n))
						continue
					}
					fmt.Fprintf(w, "<div title='%s'>%.8s</div>,", ticket.VerifierID, ticket.VerifierID)
					if i == len(tickets)-1 {
						fmt.Fprintf(w, "</div>")
					}
				}
				fmt.Fprintf(w, "</div>")
			}
			fmt.Fprintf(w, "</div></td>")

			fmt.Fprintf(w, "<td>")
			fmt.Fprintf(w, "-")
			fmt.Fprintf(w, "</td></tr>")
		}
		fmt.Fprintf(w, "</table>")

		if !roundHasRanks {
			return
		}
		// VRFS
		vrfSharesMap := rnd.GetVRFShares()
		vrfShares := make([]*round.VRFShare, 0, len(vrfSharesMap))
		for _, share := range vrfSharesMap {
			vrfShares = append(vrfShares, share)
		}
		sort.SliceStable(vrfShares, func(i, j int) bool {
			return vrfShares[i].GetParty().SetIndex < vrfShares[j].GetParty().SetIndex
		})
		fmt.Fprintf(w, "<h3>VRF Shares</h3>")
		fmt.Fprintf(w, "<table>")
		fmt.Fprintf(w, "<tr class='header'><th>Set Index</th><th>Node</th><th>VRFS (%d/%d)</th></tr>", len(vrfShares), mb.Miners.Size())
		for _, share := range vrfShares {
			fmt.Fprintf(w, "<tr><td>")
			n := share.GetParty()
			if n != nil {
				fmt.Fprintf(w, "%d", n.SetIndex)
				if c.IsRoundGenerator(rnd, n) {
					fmt.Fprintf(w, "<sup>%d</sup>", rnd.GetMinerRank(n))
				}
				fmt.Fprintf(w, "</td><td>%s</td>", getNodeLink(n))

			} else {
				fmt.Fprintf(w, "-</td><td>-</td>")
			}
			fmt.Fprintf(w, "<td>%v</td></tr>", share.Share)
		}
		fmt.Fprintf(w, "</table>")
	}
}

/*MinerStatsHandler - handler for the miner stats */
func (c *Chain) MinerStatsHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recover() != nil {
			http.Error(w, fmt.Sprintf("<pre>%s</pre>", string(debug.Stack())), http.StatusInternalServerError)
		}
	}()
	mb := c.GetCurrentMagicBlock()
	numGenerators := c.GetGeneratorsNumOfMagicBlock(mb)
	PrintCSS(w)
	fmt.Fprintf(w, "<div>%v - %v</div>", node.Self.Underlying().GetPseudoName(),
		node.Self.Underlying().Description)
	c.healthSummary(w, r)
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td colspan='3' style='text-align:center'>")
	c.notarizedBlockCountsStats(w, numGenerators)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<tr><th>Generation Counts</th><th>Verification Counts</th><th>Finalization Counts</th></tr>")
	fmt.Fprintf(w, "<tr><td>")
	c.generationCountStats(w)
	fmt.Fprintf(w, "</td><td>")
	c.verificationCountStats(w, numGenerators)
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
	generatorsNum := c.GetGeneratorsNumOfMagicBlock(mb)
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Miner</td>")
	for i := 0; i < generatorsNum; i++ {
		fmt.Fprintf(w, "<td>Rank %d</td>", i)
	}
	fmt.Fprintf(w, "<td>Total</td></tr>")
	totals := make([]int64, generatorsNum)
	for _, nd := range mb.Miners.CopyNodes() {
		fmt.Fprintf(w, "<tr><td>%v</td>", nd.GetPseudoName())
		ms := nd.ProtocolStats.(*MinerStats)
		var total int64
		for i := 0; i < generatorsNum; i++ {
			fmt.Fprintf(w, "<td class='number'>%v</td>", ms.GenerationCountByRank[i])
			totals[i] += ms.GenerationCountByRank[i]
			total += ms.GenerationCountByRank[i]
		}
		fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	}
	fmt.Fprintf(w, "<tr><td>Totals</td>")
	var total int64
	for i := 0; i < generatorsNum; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", totals[i])
		total += totals[i]
	}
	fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) verificationCountStats(w http.ResponseWriter, numGenerators int) {
	mb := c.GetCurrentMagicBlock()
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Miner</td>")
	for i := 0; i < numGenerators; i++ {
		fmt.Fprintf(w, "<td>Rank %d</td>", i)
	}
	fmt.Fprintf(w, "<td>Total</td></tr>")
	totals := make([]int64, numGenerators)
	for _, nd := range mb.Miners.CopyNodes() {
		fmt.Fprintf(w, "<tr><td>%v</td>", nd.GetPseudoName())
		ms := nd.ProtocolStats.(*MinerStats)
		var total int64
		for i := 0; i < numGenerators; i++ {
			fmt.Fprintf(w, "<td class='number'>%v</td>", ms.VerificationTicketsByRank[i])
			totals[i] += ms.VerificationTicketsByRank[i]
			total += ms.VerificationTicketsByRank[i]
		}
		fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	}
	fmt.Fprintf(w, "<tr><td>Totals</td>")
	var total int64
	for i := 0; i < numGenerators; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", totals[i])
		total += totals[i]
	}
	fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) finalizationCountStats(w http.ResponseWriter) {
	mb := c.GetCurrentMagicBlock()
	numGenerators := c.GetGeneratorsNumOfMagicBlock(mb)
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Miner</td>")
	for i := 0; i < numGenerators; i++ {
		fmt.Fprintf(w, "<td>Rank %d</td>", i)
	}
	fmt.Fprintf(w, "<td>Total</td></tr>")
	totals := make([]int64, numGenerators)
	for _, nd := range mb.Miners.CopyNodes() {
		fmt.Fprintf(w, "<tr><td>%v</td>", nd.GetPseudoName())
		ms := nd.ProtocolStats.(*MinerStats)
		var total int64
		for i := 0; i < numGenerators; i++ {
			fmt.Fprintf(w, "<td class='number'>%v</td>", ms.FinalizationCountByRank[i])
			totals[i] += ms.FinalizationCountByRank[i]
			total += ms.FinalizationCountByRank[i]
		}
		fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	}
	fmt.Fprintf(w, "<tr><td>Totals</td>")
	var total int64
	for i := 0; i < numGenerators; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", totals[i])
		total += totals[i]
	}
	fmt.Fprintf(w, "<td class='number'>%v</td></tr>", total)
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) notarizedBlockCountsStats(w http.ResponseWriter, numGenerators int) {
	fmt.Fprintf(w, "<table style='width:100%%;'>")
	fmt.Fprintf(w, "<tr><td colspan='%v'>Rounds with notarized blocks (0 to %v)</td></tr>", numGenerators+2, numGenerators)
	fmt.Fprintf(w, "<tr><td>Notarized Blocks</td>")
	for i := 0; i <= numGenerators; i++ {
		fmt.Fprintf(w, "<td class='number'>%v</td>", i)
	}
	fmt.Fprintf(w, "</tr><tr><td>Rounds</td>")
	for _, v := range c.NotarizedBlocksCounts {
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
	fmt.Fprintf(w, ".tname { width: 70%%}\n")
	fmt.Fprintf(w, "tr.header { background-color: #E0E0E0;  }\n")
	fmt.Fprintf(w, ".inactive { background-color: #F44336; }\n")
	fmt.Fprintf(w, ".warning { background-color: #FFEB3B; }\n")
	fmt.Fprintf(w, ".optimal { color: #1B5E20; }\n")
	fmt.Fprintf(w, ".slow { font-style: italic; }\n")
	fmt.Fprintf(w, ".bold {font-weight:bold;}")
	fmt.Fprintf(w, "tr.green td {background-color:light-green;}")
	fmt.Fprintf(w, "tr.grey td {background-color:light-grey;}")
	fmt.Fprintf(w, "</style>")
}

//StateDumpHandler - a handler to dump the state
func StateDumpHandler(w http.ResponseWriter, r *http.Request) {
	c := GetServerChain()
	lfb := c.GetLatestFinalizedBlock()
	contract := r.FormValue("smart_contract")
	mpt := lfb.ClientState
	if mpt == nil {
		errMsg := struct {
			Err string `json:"error"`
		}{
			Err: fmt.Sprintf("last finalized block with nil state, round: %d", lfb.Round),
		}

		out, err := json.MarshalIndent(errMsg, "", "    ")
		if err != nil {
			logging.Logger.Error("Dump state failed", zap.Error(err))
			return
		}
		fmt.Fprint(w, string(out))
		return
	}

	if contract == "" {
		contract = "global"
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
