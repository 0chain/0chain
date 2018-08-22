package chain

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"0chain.net/transaction"

	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/util"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
)

/*SetupHandlers sets up the necessary API end points */
func SetupHandlers() {
	http.HandleFunc("/v1/chain/get", common.ToJSONResponse(memorystore.WithConnectionHandler(GetChainHandler)))
	http.HandleFunc("/v1/chain/put", datastore.ToJSONEntityReqResponse(memorystore.WithConnectionEntityJSONHandler(PutChainHandler, chainEntityMetadata), chainEntityMetadata))

	// Miner can only provide recent blocks, sharders can provide any block (for content other than full) and the block they store for full
	if node.Self.Type == node.NodeTypeMiner {
		http.HandleFunc("/v1/block/get", common.ToJSONResponse(GetBlockHandler))
	}
	http.HandleFunc("/v1/block/get/latest_finalized", common.ToJSONResponse(LatestFinalizedBlockHandler))
	http.HandleFunc("/v1/block/get/recent_finalized", common.ToJSONResponse(RecentFinalizedBlockHandler))

	http.HandleFunc("/", HomePageHandler)
	http.HandleFunc("/_diagnostics", DiagnosticsHomepageHandler)

	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	http.HandleFunc("/v1/transaction/put", datastore.ToJSONEntityReqResponse(datastore.DoAsyncEntityJSONHandler(memorystore.WithConnectionEntityJSONHandler(PutTransaction, transactionEntityMetadata), transaction.TransactionEntityChannel), transactionEntityMetadata))
}

/*GetChainHandler - given an id returns the chain information */
func GetChainHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return datastore.GetEntityHandler(ctx, r, chainEntityMetadata, "id")
}

/*PutChainHandler - Given a chain data, it stores it */
func PutChainHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	return datastore.PutEntityHandler(ctx, entity)
}

/*StatusHandler - allows checking the status of the node */
func (c *Chain) StatusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		return
	}
	publicKey := r.FormValue("publicKey")
	timestamp := r.FormValue("timestamp")
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return
	}
	if !common.Within(ts, 5) {
		return
	}
	data := r.FormValue("data")
	hash := r.FormValue("hash")
	signature := r.FormValue("signature")
	if data == "" || hash == "" || signature == "" {
		return
	}
	addressParts := strings.Split(r.RemoteAddr, ":")
	node := c.Miners.GetNode(id)
	if node == nil {
		node = c.Sharders.GetNode(id)
		if node == nil {
			node = c.Blobbers.GetNode(id)
		}
	}
	if node == nil {
		return
	}
	if node.Host != addressParts[0] {
		// TODO: Node's ip address changed. Should we update ourselves?
	}
	if node.PublicKey == publicKey {
		ok, err := node.Verify(signature, hash)
		if !ok || err != nil {
			return
		}
		node.LastActiveTime = time.Now().UTC()
	} else {
		// TODO: private/public keys changed by the node. Should we update ourselves?
	}
}

/*GetMinersHandler - get the list of known miners */
func (c *Chain) GetMinersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	c.Miners.Print(w)
}

/*GetShardersHandler - get the list of known sharders */
func (c *Chain) GetShardersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	c.Sharders.Print(w)
}

/*GetBlobbersHandler - get the list of known blobbers */
func (c *Chain) GetBlobbersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	c.Blobbers.Print(w)
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

/*LatestFinalizedBlockHandler - provide the latest finalized block by this miner */
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetServerChain().LatestFinalizedBlock.GetSummary(), nil
}

/*RecentFinalizedBlockHandler - provide the latest finalized block by this miner */
func RecentFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	fbs := make([]*block.BlockSummary, 0, 10)
	for i, b := 0, GetServerChain().LatestFinalizedBlock; i < 10 && b != nil; i, b = i+1, b.PrevBlock {
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
	fmt.Fprintf(w, "<div>Working on the chain: %v</div>\n", sc.GetKey())
	fmt.Fprintf(w, "<div>I am a %v with set rank of (%v) <ul><li>id:%v</li><li>public_key:%v</li></ul></div>\n", node.Self.GetNodeTypeName(), node.Self.SetIndex, node.Self.GetKey(), node.Self.PublicKey)
}

/*DiagnosticsHomepageHandler - handler to display the /_diagnostics page */
func DiagnosticsHomepageHandler(w http.ResponseWriter, r *http.Request) {
	sc := GetServerChain()
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	fmt.Fprintf(w, "<div>Running since %v (%v) ...\n", StartTime, time.Since(StartTime))
	fmt.Fprintf(w, "<div>Working on the chain: %v</div>\n", sc.GetKey())
	fmt.Fprintf(w, "<div>I am a %v with set rank of (%v) <ul><li>id:%v</li><li>public_key:%v</li></ul></div>\n", node.Self.GetNodeTypeName(), node.Self.SetIndex, node.Self.GetKey(), node.Self.PublicKey)
	fmt.Fprintf(w, "<ul>")
	fmt.Fprintf(w, "<li><a href='/v1/config/get'>/v1/config/get</a></li>")
	fmt.Fprintf(w, "<li><a href='/_chain_stats'>/_chain_stats</a></li>")
	fmt.Fprintf(w, "<li><a href='/_diagnostics/info'>/_diagnostics/info</a></li>")
	fmt.Fprintf(w, "<li><a href='/_diagnostics/n2n/info'>/_diagnostics/n2n/info</a></li>")
	fmt.Fprintf(w, "<li>/_diagnostics/logs [Level <a href='/_diagnostics/logs?detail=1'>1</a>, <a href='/_diagnostics/logs?detail=2'>2</a>, <a href='/_diagnostics/logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li>/_diagnostics/n2n_logs [Level <a href='/_diagnostics/n2n_logs?detail=1'>1</a>, <a href='/_diagnostics/n2n_logs?detail=2'>2</a>, <a href='/_diagnostics/n2n_logs?detail=3'>3</a>]</li>")
	fmt.Fprintf(w, "<li><a href='/debug/pprof/'>/debug/pprof/</a></li>")
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "<div><div>Miners (%v)</div>", sc.Miners.Size())
	printNodePool(w, sc.Miners)
	fmt.Fprintf(w, "</div>")
	fmt.Fprintf(w, "<div><div>Sharders (%v)</div>", sc.Sharders.Size())
	printNodePool(w, sc.Sharders)
	fmt.Fprintf(w, "</div>")
}

func printNodePool(w http.ResponseWriter, np *node.Pool) {
	nodes := np.Nodes
	fmt.Fprintf(w, "<style>\n")
	fmt.Fprintf(w, ".number { text-align: right; }\n")
	fmt.Fprintf(w, "table, td, th { border: 1px solid black; }\n")
	fmt.Fprintf(w, "</style>")
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr><td>Set Index</td><td>Node</td><td>Sent</td><td>Received</td><td>Last Active</td></tr>")
	for _, nd := range nodes {
		fmt.Fprintf(w, "<tr>")
		fmt.Fprintf(w, "<td>%d</td>", nd.SetIndex)
		if nd == node.Self.Node {
			fmt.Fprintf(w, "<td>%v%.3d</td>", nd.GetNodeTypeName(), nd.SetIndex)
		} else {
			fmt.Fprintf(w, "<td><a href='http://%v:%v/'>%v%.3d</a></td>", nd.Host, nd.Port, nd.GetNodeTypeName(), nd.SetIndex)
		}
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.Sent)
		fmt.Fprintf(w, "<td class='number'>%d</td>", nd.Received)
		fmt.Fprintf(w, "<td>%v</td>", nd.LastActiveTime)
		fmt.Fprintf(w, "</tr>")
	}
	fmt.Fprintf(w, "</table>")
}

/*InfoHandler - handler to get the information of the chain */
func InfoHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	idx := 0
	chainInfo := ChainMetric.GetAll()
	for ; idx < len(chainInfo); idx++ {
		if chainInfo[idx].GetValue() == 0 {
			break
		}
	}
	info := make(map[string]interface{})
	info["chain_info"] = chainInfo[:idx]

	roundInfo := RoundMetric.GetAll()
	for idx = 0; idx < len(roundInfo); idx++ {
		if roundInfo[idx].GetValue() == 0 {
			break
		}
	}
	info["round_info"] = roundInfo[:idx]
	return info, nil
}

/*InfoWriter - a handler to get the information of the chain */
func InfoWriter(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<style>\n")
	fmt.Fprintf(w, ".number { text-align: right; }\n")
	fmt.Fprintf(w, "table, td, th { border: 1px solid black; }\n")
	fmt.Fprintf(w, "tr:nth-child(10n + 3) { background-color: #abb2b9; }\n")
	fmt.Fprintf(w, "</style>")
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr><th>Round</th><th>Chain Weight</th><th>Block Hash</th><th>Client State Hash</th><th>Blocks Count</th><th>Missed Blocks</th></tr>")
	chainInfo := ChainMetric.GetAll()
	for idx := 0; idx < len(chainInfo); idx++ {
		cf := chainInfo[idx].(*Info)
		if cf.FinalizedRound == 0 {
			break
		}
		fmt.Fprintf(w, "<tr>")
		fmt.Fprintf(w, "<td class='number'>%11d</td>", cf.FinalizedRound)
		fmt.Fprintf(w, "<td class='number'>%.8f</td>", cf.ChainWeight)
		fmt.Fprintf(w, "<td>%s</td>", cf.BlockHash)
		fmt.Fprintf(w, "<td>%v</td>", util.ToHex(cf.ClientStateHash))
		fmt.Fprintf(w, "<td class='number'>%11d</td>", cf.FinalizedCount)
		fmt.Fprintf(w, "<td class='number'>%6d</td>", cf.MissedBlocks)
		fmt.Fprintf(w, "</tr>")
	}
	fmt.Fprintf(w, "</table>")
	fmt.Fprintf(w, "<br/>")
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr><th>Round</th><th>Blocks Count</th><th>Multi Block Count</th><th>Zero Block Count</tr></tr>")
	roundInfo := RoundMetric.GetAll()
	for idx := 0; idx < len(roundInfo); idx++ {
		rf := roundInfo[idx].(*round.Info)
		if rf.Number == 0 {
			break
		}
		fmt.Fprintf(w, "<tr>")
		fmt.Fprintf(w, "<td class='number'>%d</td>", rf.Number)
		fmt.Fprintf(w, "<td class='number'>%d</td>", rf.NotarizedBlocksCount)
		fmt.Fprintf(w, "<td class='number'>%d</td>", rf.MultiNotarizedBlocksCount)
		fmt.Fprintf(w, "<td class='number'>%6d</td>", rf.ZeroNotarizedBlocksCount)
		fmt.Fprintf(w, "</tr>")
	}
	fmt.Fprintf(w, "</table>")
}

//SendStatsWriter - writes the send stats of all the nodes
func (c *Chain) SendStatsWriter(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<style>\n")
	fmt.Fprintf(w, ".number { text-align: right; }\n")
	fmt.Fprintf(w, "table, td, th { border: 1px solid black; }\n")
	fmt.Fprintf(w, "tr:nth-child(10n) { background-color: #f2f2f2; }\n")
	fmt.Fprintf(w, "</style>")
	fmt.Fprintf(w, "<table style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr><td>URI</td><td>Count</td><td>Min</td><td>Average</td><td>Max</td></tr>")
	for _, n := range c.Miners.Nodes {
		if n == node.Self.Node {
			continue
		}
		fmt.Fprintf(w, "<tr><th colspan='5'>%s</th></tr>", fmt.Sprintf("%v%3d", n.GetNodeTypeName(), n.SetIndex))
		n.PrintSendStats(w)
	}

	for _, n := range c.Sharders.Nodes {
		if n == node.Self.Node {
			continue
		}
		fmt.Fprintf(w, "<tr><th colspan='5'>%s</th></tr>", fmt.Sprintf("%v%3d", n.GetNodeTypeName(), n.SetIndex))
		n.PrintSendStats(w)
	}
	fmt.Fprintf(w, "</table>")
}

func PutTransaction(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	txn, ok := entity.(*transaction.Transaction)
	if !ok {
		return nil, fmt.Errorf("invalid request %T", entity)
	}
	max_txn_size := GetServerChain().TxnMaxPayload * 1024
	if max_txn_size > 0 {
		txn_size := getTxnPayloadSize(txn)
		if txn_size > max_txn_size {
			s := fmt.Sprintf("transaction payload exceeds the max payload (%d)", GetServerChain().TxnMaxPayload)
			return nil, common.NewError("txn_exceed_max_payload", s)
		}
	}
	return transaction.PutTransaction(ctx, txn)
}

func getTxnPayloadSize(txn *transaction.Transaction) int {
	var sizeInBytes int = 0

	sizeInBytes += len(txn.ClientID)
	sizeInBytes += int(reflect.TypeOf(txn.CreationDate).Size())
	sizeInBytes += len(txn.Hash)
	sizeInBytes += len(txn.Signature)
	sizeInBytes += len(txn.ToClientID)
	sizeInBytes += len(txn.TransactionData)
	sizeInBytes += int(reflect.TypeOf(txn.Value).Size())

	return sizeInBytes
}
