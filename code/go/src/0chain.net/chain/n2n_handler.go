package chain

import (
	"net/http"

	"0chain.net/datastore"
	"0chain.net/node"
)

/*SetupNodeHandlers - setup the handlers for the chain */
func (c *Chain) SetupNodeHandlers() {
	http.HandleFunc("/_nh/list/m", c.GetMinersHandler)
	http.HandleFunc("/_nh/list/s", c.GetShardersHandler)
	http.HandleFunc("/_nh/list/b", c.GetBlobbersHandler)
}

/*MinerNotarizedBlockRequestor - reuqest a notarized block from a node*/
var MinerNotarizedBlockRequestor node.EntityRequestor

//BlockStateChangeRequestor - request state changes for the block
var BlockStateChangeRequestor node.EntityRequestor

/*SetupX2MRequestors - setup requestors */
func SetupX2MRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	MinerNotarizedBlockRequestor = node.RequestEntityHandler("/v1/_x2m/block/notarized_block/get", options, blockEntityMetadata)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_JSON, Compress: true}
	blockStateChangeEntityMetadata := datastore.GetEntityMetadata("block_state_change")
	BlockStateChangeRequestor = node.RequestEntityHandler("/v1/_x2m/block/state_change/get", options, blockStateChangeEntityMetadata)
}
