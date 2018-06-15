package miner

import (
	"time"

	"0chain.net/node"
)

/*This file contains the Miner To Sharder send/receive messages */

/*FinalizedBlockSender - Send the block to a node */
var FinalizedBlockSender node.EntitySendHandler

/*SetupM2SSenders - setup message senders from miners to sharders */
func SetupM2SSenders() {
	options := &node.SendOptions{Timeout: 2 * time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	FinalizedBlockSender = node.SendEntityHandler("/v1/_m2s/block/finalized", options)
}
