package miner

/*This file contains the Miner To Miner send/receive messages */
import (
	"context"
	"net/http"
	"strconv"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/state"
	"0chain.net/threshold/bls"
	"go.uber.org/zap"
)

/*RoundStartSender - Start a new round */
var RoundStartSender node.EntitySendHandler

/*RoundVRFSender - Send the round vrf */
var RoundVRFSender node.EntitySendHandler

/*VerifyBlockSender - Send the block to a node */
var VerifyBlockSender node.EntitySendHandler

/*VerificationTicketSender - Send a verification ticket to a node */
var VerificationTicketSender node.EntitySendHandler

/*BlockNotarizationSender - Send the block notarization to a node */
var BlockNotarizationSender node.EntitySendHandler

/*MinerNotarizedBlockSender - Send a notarized block to a node */
var MinerNotarizedBlockSender node.EntitySendHandler

/*DKGShareSender - Send dkg share to a node*/
var DKGShareSender node.EntitySendHandler

/*MinerLatestFinalizedBlockRequestor - RequestHandler for latest finalized block to a node */
var MinerLatestFinalizedBlockRequestor node.EntityRequestor

/*SetupM2MSenders - setup senders for miner to miner communication */
func SetupM2MSenders() {

	options := &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	RoundVRFSender = node.SendEntityHandler("/v1/_m2m/round/vrf_share", options)

	//TODO: changes options and url as per requirements
	options = &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	DKGShareSender = node.SendEntityHandler("/v1/_m2m/dkg/share", options)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	VerifyBlockSender = node.SendEntityHandler("/v1/_m2m/block/verify", options)
	MinerNotarizedBlockSender = node.SendEntityHandler("/v1/_m2m/block/notarized_block", options)

	options = &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	VerificationTicketSender = node.SendEntityHandler("/v1/_m2m/block/verification_ticket", options)

	options = &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	BlockNotarizationSender = node.SendEntityHandler("/v1/_m2m/block/notarization", options)

}

/*SetupM2MReceivers - setup receivers for miner to miner communication */
func SetupM2MReceivers() {
	http.HandleFunc("/v1/_m2m/dkg/share", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(DKGShareHandler, nil)))
	http.HandleFunc("/v1/_m2m/round/vrf_share", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(VRFShareHandler, nil)))
	http.HandleFunc("/v1/_m2m/block/verify", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(memorystore.WithConnectionEntityJSONHandler(VerifyBlockHandler, datastore.GetEntityMetadata("block")), nil)))
	http.HandleFunc("/v1/_m2m/block/verification_ticket", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(VerificationTicketReceiptHandler, nil)))
	http.HandleFunc("/v1/_m2m/block/notarization", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(NotarizationReceiptHandler, nil)))
	http.HandleFunc("/v1/_m2m/block/notarized_block", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(NotarizedBlockHandler, nil)))
}

/*SetupX2MResponders - setup responders */
func SetupX2MResponders() {
	http.HandleFunc("/v1/_x2m/block/notarized_block/get", common.N2NRateLimit(node.ToN2NSendEntityHandler(NotarizedBlockSendHandler)))
	http.HandleFunc("/v1/_x2m/block/state_change/get", common.N2NRateLimit(node.ToN2NSendEntityHandler(BlockStateChangeHandler)))
}

/*SetupM2SRequestors - setup all requests to sharder by miner */
func SetupM2SRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	MinerLatestFinalizedBlockRequestor = node.RequestEntityHandler("/v1/_m2s/block/latest_finalized/get", options, blockEntityMetadata)
}

/*VRFShareHandler - handle the vrf share */
func VRFShareHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	vrfs, ok := entity.(*round.VRFShare)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	if vrfs.GetRoundNumber() < mc.LatestFinalizedBlock.Round {
		return nil, nil
	}
	mr := mc.GetMinerRound(vrfs.GetRoundNumber())
	if mr != nil && mr.IsVRFComplete() {
		return nil, nil
	}
	msg := NewBlockMessage(MessageVRFShare, node.GetSender(ctx), nil, nil)
	vrfs.SetParty(msg.Sender)
	msg.VRFShare = vrfs
	mc.GetBlockMessageChannel() <- msg
	return nil, nil
}

/*DKGShareHandler - handles the dkg share it receives from a node */
func DKGShareHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	dg, ok := entity.(*bls.Dkg)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	//ToDo: Need to make sure SENDER is not byzantine
	nodeID := node.GetSender(ctx).SetIndex
	Logger.Debug("received DKG share", zap.String("share", dg.Share), zap.Int("Node Id", nodeID))
	AppendDKGSecShares(nodeID, dg.Share)
	return nil, nil
}

/*VerifyBlockHandler - verify the block that is received */
func VerifyBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	if b.MinerID == node.Self.GetKey() {
		return nil, nil
	}
	if b.Round < mc.LatestFinalizedBlock.Round {
		Logger.Debug("verify block handler", zap.Int64("round", b.Round), zap.Int64("lf_round", mc.LatestFinalizedBlock.Round))
		return nil, nil
	}
	err := b.Validate(ctx)
	if err != nil {
		return nil, err
	}
	msg := NewBlockMessage(MessageVerify, node.GetSender(ctx), nil, b)
	mc.GetBlockMessageChannel() <- msg
	return nil, nil
}

/*VerificationTicketReceiptHandler - Add a verification ticket to the block */
func VerificationTicketReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	bvt, ok := entity.(*block.BlockVerificationTicket)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	msg := NewBlockMessage(MessageVerificationTicket, node.GetSender(ctx), nil, nil)
	msg.BlockVerificationTicket = bvt
	GetMinerChain().GetBlockMessageChannel() <- msg
	return nil, nil
}

/*NotarizationReceiptHandler - handles the receipt of a notarization for a block */
func NotarizationReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	notarization, ok := entity.(*Notarization)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	if notarization.Round < mc.LatestFinalizedBlock.Round {
		Logger.Debug("notarization receipt handler", zap.Int64("round", notarization.Round), zap.Int64("lf_round", mc.LatestFinalizedBlock.Round))
		return nil, nil
	}
	msg := NewBlockMessage(MessageNotarization, node.GetSender(ctx), nil, nil)
	msg.Notarization = notarization
	mc.GetBlockMessageChannel() <- msg
	return nil, nil
}

/*NotarizedBlockHandler - handles a notarized block*/
func NotarizedBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	if b.Round < mc.CurrentRound-1 {
		Logger.Debug("notarized block handler (round older than the current round)", zap.String("block", b.Hash), zap.Any("round", b.Round))
		return nil, nil
	}
	if err := mc.VerifyNotarization(ctx, b.Hash, b.VerificationTickets); err != nil {
		return nil, err
	}
	msg := &BlockMessage{Sender: node.GetSender(ctx), Type: MessageNotarizedBlock, Block: b}
	mc.GetBlockMessageChannel() <- msg
	return nil, nil
}

//NotarizedBlockSendHandler - handles a request for a notarized block
func NotarizedBlockSendHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return getNotarizedBlock(ctx, r)
}

//BlockStateChangeHandler - provide the state changes associated with a block
func BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	b, err := getNotarizedBlock(ctx, r)
	if err != nil {
		return nil, err
	}
	if b.GetStateStatus() != block.StateSuccessful {
		return nil, common.NewError("state_not_verified", "State is not computed and validated locally")
	}
	bsc := block.NewBlockStateChange(b)
	if state.Debug() {
		Logger.Info("block state change handler", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("state_changes", len(b.ClientState.GetChangeCollector().GetChanges())), zap.Int("sc_nodes", len(bsc.Nodes)))
	}
	return bsc, nil
}

func getNotarizedBlock(ctx context.Context, r *http.Request) (*block.Block, error) {
	mc := GetMinerChain()
	round := r.FormValue("round")
	hash := r.FormValue("block")
	if round != "" {
		roundN, err := strconv.ParseInt(round, 10, 63)
		if err != nil {
			return nil, err
		}
		r := mc.GetRound(roundN)
		if r != nil {
			b := r.GetHeaviestNotarizedBlock()
			if b != nil {
				return b, nil
			}
		}
	} else if hash != "" {
		b, err := mc.GetBlock(ctx, hash)
		if err != nil {
			return nil, err
		}
		if b.IsBlockNotarized() {
			return b, nil
		}
	} else {
		for r := mc.GetRound(mc.CurrentRound); r != nil; r = mc.GetRound(r.GetRoundNumber() - 1) {
			b := r.GetHeaviestNotarizedBlock()
			if b != nil {
				return b, nil
			}
		}
	}
	return nil, common.NewError("block_not_available", "Requested block is not available")
}
