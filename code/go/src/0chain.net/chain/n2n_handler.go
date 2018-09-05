package chain

import (
	"net/http"
	"time"

	"0chain.net/datastore"
	"0chain.net/node"
)

/*SetupNodeHandlers - setup the handlers for the chain */
func (c *Chain) SetupNodeHandlers() {
	http.HandleFunc("/_nh/status", c.StatusHandler)
	http.HandleFunc("/_nh/list/m", c.GetMinersHandler)
	http.HandleFunc("/_nh/list/s", c.GetShardersHandler)
	http.HandleFunc("/_nh/list/b", c.GetBlobbersHandler)
}

/*MinerNotarizedBlockRequestor - reuqest a notarized block from a node*/
var MinerNotarizedBlockRequestor node.EntityRequestor

/*SetupX2MRequestors - setup requestors */
func SetupX2MRequestors() {
	options := &node.SendOptions{Timeout: 2 * time.Second, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	MinerNotarizedBlockRequestor = node.RequestEntityHandler("/v1/_x2m/block/notarized_block/get", options, blockEntityMetadata)
}
