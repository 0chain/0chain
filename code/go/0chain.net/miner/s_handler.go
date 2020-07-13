package miner

import (
	"0chain.net/chaincore/node"
)

/*This file contains the Miner To Sharder send/receive messages */

/*FinalizedBlockSender - Send the block to a node */
var FinalizedBlockSender node.EntitySendHandler

/*NotarizedBlockSender - Send a notarized block to a node */
var NotarizedBlockSender node.EntitySendHandler

// NotarizedBlockForcePushSender is notarized blocks sender that
// pushes the blocks instead of push-to-pull strategy.
var NotarizedBlockForcePushSender node.EntitySendHandler

/*SetupM2SSenders - setup message senders from miners to sharders */
func SetupM2SSenders() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true, Pull: true}
	FinalizedBlockSender = node.SendEntityHandler("/v1/_m2s/block/finalized", options)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true, Pull: true}
	NotarizedBlockSender = node.SendEntityHandler("/v1/_m2s/block/notarized", options)

	NotarizedBlockForcePushSender = node.SendEntityHandler(
		"/v1/_m2s/block/notarized",
		&node.SendOptions{
			Timeout:            node.TimeoutSmallMessage,
			MaxRelayLength:     0,
			CurrentRelayLength: 0,
			CODEC:              node.CODEC_MSGPACK,
			Compress:           true,
			Pull:               false,
		})
}
