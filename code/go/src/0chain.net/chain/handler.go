package chain

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"0chain.net/node"

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
		http.HandleFunc("/v1/block/get", common.ToJSONResponse(BlockHandler))
	}
	http.HandleFunc("/v1/block/get/latest_finalized", common.ToJSONResponse(LatestFinalizedBlockHandler))
	http.HandleFunc("/v1/block/get/recent_finalized", common.ToJSONResponse(RecentFinalizedBlockHandler))
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

func BlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
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
