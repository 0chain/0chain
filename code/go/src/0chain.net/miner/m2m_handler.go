package miner

/*This file contains the Miner To Miner send/receive messages */
import (
	"context"
	"net/http"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/node"
)

var VBSender node.EntitySendHandler
var VTSender node.EntitySendHandler

/*SetupM2MSenders - setup senders for miner to miner communication */
func SetupM2MSenders() {
	options := &node.SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: true}
	VBSender = node.SendEntityHandler("/v1/_m2m/block/verify", options)

	options = &node.SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	VTSender = node.SendEntityHandler("/v1/_m2m/block/verification_ticket", options)
}

/*SetupM2MReceivers - setup receivers for miner to miner communication */
func SetupM2MReceivers() {
	http.HandleFunc("/v1/_m2m/block/verify", node.ToN2NReceiveEntityHandler(memorystore.WithConnectionEntityJSONHandler(VerifyBlockHandler, datastore.GetEntityMetadata("block"))))

	http.HandleFunc("/v1/_m2m/block/verification_ticket", node.ToN2NReceiveEntityHandler(VerificationTicketReceiptHandler))
}

/*VerifyBlockHandler - verify the block that is received */
func VerifyBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	// TODO: This should be async process where the block goes into the Rounds channel
	ok, err := b.VerifyBlock(ctx)
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
		return nil, err
	}
	sender := node.GetSender(ctx)
	if !datastore.IsEqual(sender.GetKey(), bvt.VerifierID) {
		return nil, common.InvalidRequest("Verifier and original sender are not the same")
	}
	if ok, _ := sender.Verify(bvt.Signature, block.Signature); !ok {
		return nil, common.InvalidRequest("Couldn't verify the signature")
	}
	block.AddVerificationTicket(&bvt.VerificationTicket)
	return true, nil
}
