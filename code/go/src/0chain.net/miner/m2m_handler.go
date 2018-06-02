package miner

/*This file contains the Miner To Miner send/receive messages */
import (
	"context"
	"net/http"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/node"
)

var VBSender node.EntitySendHandler
var VTSender node.EntitySendHandler
var ConsensusSender node.EntitySendHandler

/*SetupM2MSenders - setup senders for miner to miner communication */
func SetupM2MSenders() {
	options := &node.SendOptions{Timeout: 2 * time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	VBSender = node.SendEntityHandler("/v1/_m2m/block/verify", options)

	options = &node.SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	VTSender = node.SendEntityHandler("/v1/_m2m/block/verification_ticket", options)

	options = &node.SendOptions{Timeout: time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	ConsensusSender = node.SendEntityHandler("/v1/_m2m/block/consensus", options)
}

/*SetupM2MReceivers - setup receivers for miner to miner communication */
func SetupM2MReceivers() {
	http.HandleFunc("/v1/_m2m/block/verify", node.ToN2NReceiveEntityHandler(memorystore.WithConnectionEntityJSONHandler(VerifyBlockHandler, datastore.GetEntityMetadata("block"))))

	http.HandleFunc("/v1/_m2m/block/verification_ticket", node.ToN2NReceiveEntityHandler(VerificationTicketReceiptHandler))

	http.HandleFunc("/v1/_m2m/block/consensus", node.ToN2NReceiveEntityHandler(ConsensusReceiptHandler))
}

/*VerifyBlockHandler - verify the block that is received */
func VerifyBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	// TODO: This should be async process where the block goes into the Rounds channel
	ok, err := mc.VerifyBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, common.InvalidRequest("Block couldnot be verified")
	}
	return true, nil
}

/*VerificationTicketReceiptHandler - Add a verification ticket to the block */
func VerificationTicketReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	bvt, ok := entity.(*block.BlockVerificationTicket)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	block, err := mc.GetBlock(ctx, bvt.BlockID)
	if err != nil {
		// TODO: If we didn't see this block so far, may be it's better to ask for it
		return nil, err
	}
	err = mc.VerifyTicket(bvt)
	if err != nil {
		return nil, err
	}
	mc.AddVerificationTicket(ctx, block, &bvt.VerificationTicket)
	return true, nil
}

/*ConsensusReceiptHandler - handles the receipt of a consensus for a block */
func ConsensusReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	//TODO
	// If we already saw this, ignore it
	// Validate consensus is correctly
	// Finalize the previous block of this consenus block
	return nil, nil
}
