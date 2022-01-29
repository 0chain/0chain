package miner

/*This file contains the Miner To Miner send/receive messages */
import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"context"
	"encoding/hex"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

var (
	// RoundStartSender - Start a new round.
	RoundStartSender node.EntitySendHandler
	// RoundVRFSender - Send the round vrf.
	RoundVRFSender node.EntitySendHandler
	// VerifyBlockSender - Send the block to a node.
	VerifyBlockSender node.EntitySendHandler

	// VerificationTicketSender - Send a verification ticket to a node.
	VerificationTicketSender node.EntitySendHandler
	// BlockNotarizationSender - Send the block notarization to a node.
	BlockNotarizationSender node.EntitySendHandler
	// MinerNotarizedBlockSender - Send a notarized block to a node.
	MinerNotarizedBlockSender node.EntitySendHandler
	// DKGShareSender - Send dkg share to a node
	DKGShareSender node.EntityRequestor
	// ChainStartSender - Send whether or not to start chain
	ChainStartSender node.EntityRequestor
	// MinerLatestFinalizedBlockRequestor - RequestHandler for latest finalized
	// block to a node.
	MinerLatestFinalizedBlockRequestor node.EntityRequestor
	// LatestFinalizedMagicBlockRequestor - RequestHandler for latest finalized
	// magic block to a node.
	BlockRequestor node.EntityRequestor
)

/*SetupM2MSenders - setup senders for miner to miner communication */
func SetupM2MSenders() {

	options := &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	RoundVRFSender = node.SendEntityHandler("/v1/_m2m/round/vrf_share", options)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	VerifyBlockSender = node.SendEntityHandler("/v1/_m2m/block/verify", options)
	MinerNotarizedBlockSender = node.SendEntityHandler("/v1/_m2m/block/notarized_block", options)

	options = &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	VerificationTicketSender = node.SendEntityHandler("/v1/_m2m/block/verification_ticket", options)

	options = &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	BlockNotarizationSender = node.SendEntityHandler("/v1/_m2m/block/notarization", options)

}

/*SetupM2MReceivers - setup receivers for miner to miner communication */
func SetupM2MReceivers(c node.Chainer) {
	http.HandleFunc("/v1/_m2m/round/vrf_share",
		common.N2NRateLimit(node.ToN2NReceiveEntityHandler(VRFShareHandler, nil)))
	http.HandleFunc("/v1/_m2m/block/verification_ticket",
		common.N2NRateLimit(node.StopOnBlockSyncingHandler(c,
			node.ToN2NReceiveEntityHandler(
				VerificationTicketReceiptHandler, nil))))
	http.HandleFunc("/v1/_m2m/block/verify",
		common.N2NRateLimit(node.ToN2NReceiveEntityHandler(memorystore.WithConnectionEntityJSONHandler(
			VerifyBlockHandler, datastore.GetEntityMetadata("block")), nil)))
	http.HandleFunc("/v1/_m2m/block/notarization",
		common.N2NRateLimit(node.ToN2NReceiveEntityHandler(NotarizationReceiptHandler, nil)))
	http.HandleFunc("/v1/_m2m/block/notarized_block",
		common.N2NRateLimit(node.ToN2NReceiveEntityHandler(
			NotarizedBlockHandler, nil)))
}

/*SetupX2MResponders - setup responders */
func SetupX2MResponders() {
	http.HandleFunc("/v1/_x2m/block/notarized_block/get", common.N2NRateLimit(node.ToN2NSendEntityHandler(NotarizedBlockSendHandler)))

	http.HandleFunc("/v1/_x2m/state/get", common.N2NRateLimit(node.ToN2NSendEntityHandler(PartialStateHandler)))
	http.HandleFunc("/v1/_m2m/dkg/share", common.N2NRateLimit(node.ToN2NSendEntityHandler(SignShareRequestHandler)))
	http.HandleFunc("/v1/_m2m/chain/start", common.N2NRateLimit(node.ToN2NSendEntityHandler(StartChainRequestHandler)))
}

/*SetupM2SRequestors - setup all requests to sharder by miner */
func SetupM2SRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	MinerLatestFinalizedBlockRequestor = node.RequestEntityHandler("/v1/_m2s/block/latest_finalized/get", options, blockEntityMetadata)
	BlockRequestor = node.RequestEntityHandler("/v1/block/get", options, blockEntityMetadata)
}

func SetupM2MRequestors() {
	dkgShareEntityMetadata := datastore.GetEntityMetadata("dkg_share")
	options := &node.SendOptions{Timeout: node.TimeoutSmallMessage, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	DKGShareSender = node.RequestEntityHandler("/v1/_m2m/dkg/share", options, dkgShareEntityMetadata)

	chainStartEntityMetadata := datastore.GetEntityMetadata("start_chain")
	ChainStartSender = node.RequestEntityHandler("/v1/_m2m/chain/start", options, chainStartEntityMetadata)
}

// VRFShareHandler - handle the vrf share.
func VRFShareHandler(ctx context.Context, entity datastore.Entity) (
	interface{}, error) {
	vrfs, ok := entity.(*round.VRFShare)
	if !ok {
		logging.Logger.Info("VRFShare: returning invalid Entity")
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()

	// skip all VRFS before LFB-ticket (sharders' LFB)
	var tk = mc.GetLatestLFBTicket(ctx)
	if tk == nil {
		return nil, common.NewError("Reject VRFShare", "context done")
	}
	var (
		lfb   = mc.GetLatestFinalizedBlock()
		bound = tk.Round
	)

	if lfb.Round < tk.Round {
		bound = lfb.Round // use lower one
	}
	if vrfs.GetRoundNumber() < bound {
		logging.Logger.Info("Reject VRFShare: old round",
			zap.Int64("vrfs_round", vrfs.GetRoundNumber()),
			zap.Int64("lfb_ticket_round", tk.Round),
			zap.Int64("lfb_round", lfb.Round),
			zap.Int64("bound", bound))
		return nil, nil
	}

	// push not. block to a miner behind
	if vrfs.Round < mc.GetCurrentRound() {
		var mr = mc.GetMinerRound(vrfs.Round)
		if mr == nil {
			logging.Logger.Info("Reject VRFShare: missing miner round",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
		if mr.Block == nil || !mr.Block.IsBlockNotarized() {
			logging.Logger.Info("Reject VRFShare: missing HNB for the round"+
				" or it's not notarized",
				zap.Bool("is_not_notarized", mr.Block != nil),
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
		// var hnb = mr.GetHeaviestNotarizedBlock()
		var hnb = mr.Block
		if hnb.GetStateStatus() != block.StateSuccessful && hnb.GetStateStatus() != block.StateSynched {
			logging.Logger.Info("Reject VRFShare: HNB state is not successful",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()),
				zap.String("hash", hnb.Hash))
			return nil, nil
		}
		var (
			mb    = mc.GetMagicBlock(vrfs.Round)
			party = node.GetSender(ctx)
		)
		if mb == nil {
			logging.Logger.Info("Reject VRFShare: missing MB for the round",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
		if party == nil {
			logging.Logger.Info("Reject VRFShare: missing party",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
		var found *node.Node
		for _, miner := range mb.Miners.Nodes {
			if miner.ID == party.ID {
				found = miner
				break
			}
		}
		if found == nil {
			logging.Logger.Info("Reject VRFShare: missing party in MB",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}

		if err := node.ValidateSenderSignature(ctx); err != nil {
			return nil, err
		}

		// send notarized block
		go mb.Miners.SendTo(ctx, MinerNotarizedBlockSender(hnb), found.ID)

		logging.Logger.Info("Reject VRFShare: push not. block message for the miner behind",
			zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()),
			zap.String("to_miner_id", found.ID),
			zap.String("to_miner_url", found.GetN2NURLBase()))
		return nil, nil
	}

	sender := node.GetSender(ctx)
	vrfs.SetParty(sender)
	if mr := mc.GetMinerRound(vrfs.Round); mr != nil {
		if mr.IsVRFComplete() {
			logging.Logger.Info("Reject VRFShare: VRF is complete for this round",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()),
				zap.Int("vrfs_sender_index", sender.SetIndex),
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}

		if mr.VRFShareExist(vrfs) {
			logging.Logger.Info("Reject VRFShare: VRF is already exist",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()),
				zap.Int("vrfs_sender_index", sender.SetIndex),
				zap.String("vrfs_id", vrfs.GetKey()),
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
	}

	if err := node.ValidateSenderSignature(ctx); err != nil {
		return nil, err
	}

	var msg = NewBlockMessage(MessageVRFShare, sender, nil, nil)
	msg.VRFShare = vrfs
	mc.PushBlockMessageChannel(msg)
	return nil, nil
}

// VerifyBlockHandler - verify the block that is received.
func VerifyBlockHandler(ctx context.Context, entity datastore.Entity) (
	interface{}, error) {

	var b, ok = entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}

	mc := GetMinerChain()

	if b.MinerID == node.Self.Underlying().GetKey() {
		return nil, nil
	}

	var lfb = mc.GetLatestFinalizedBlock()
	if b.Round < lfb.Round {
		logging.Logger.Debug("handle verify block", zap.Int64("round", b.Round), zap.Int64("lf_round", lfb.Round))
		return nil, nil
	}

	var pr = mc.GetMinerRound(b.Round - 1)
	if pr == nil {
		logging.Logger.Error("handle verify block -- no previous round (ignore)",
			zap.Int64("round", b.Round), zap.Int64("prev_round", b.Round-1))
		return nil, nil
	}

	if b.Round < mc.GetCurrentRound()-1 {
		logging.Logger.Debug("verify block - round mismatch",
			zap.Int64("current_round", mc.GetCurrentRound()),
			zap.Int64("block_round", b.Round))
		return nil, nil
	}

	//if mr := mc.getOrCreateRound(ctx, b.Round); mr != nil {
	if mr := mc.GetMinerRound(b.Round); mr != nil {
		//use proposed blocks as current block cache, since we store blocks there before they are added to the round
		if mr.IsVerificationComplete() {
			logging.Logger.Debug("handle verify block - received block for round with finished verification phase")
			return nil, nil
		}
		for _, blocks := range mr.GetProposedBlocks() {
			if blocks.Hash == b.Hash {
				logging.Logger.Debug("handle verify block - block already received, ignore",
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash))
				return nil, nil
			}
		}
	}

	// return if the block already in local chain and its previous block is notarized
	_, err := mc.GetBlock(ctx, b.Hash)
	if err == nil { // block already exist in local chain
		// check if previous block exist and is notarized
		pb, err := mc.GetBlock(ctx, b.PrevHash)
		if err == nil && pb != nil && pb.IsBlockNotarized() {
			logging.Logger.Debug("handle verify block - block already exist, ignore",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return nil, nil
		}
	}

	//cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	//defer cancel()
	//if err := mc.isVRFComplete(cctx, b.Round, b.GetRoundRandomSeed()); err != nil {
	//	logging.Logger.Debug("handle verify block - vrf not complete yet",
	//		zap.Int64("round", b.Round),
	//		zap.String("block", b.Hash),
	//		zap.Error(err))
	//	return nil, nil
	//}

	if err := node.ValidateSenderSignature(ctx); err != nil {
		return nil, err
	}

	var msg = NewBlockMessage(MessageVerify, node.GetSender(ctx), nil, b)
	mc.PushBlockMessageChannel(msg)
	return nil, nil
}

/*VerificationTicketReceiptHandler - Add a verification ticket to the block */
func VerificationTicketReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	bvt, ok := entity.(*block.BlockVerificationTicket)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}

	var (
		rn = bvt.Round
		mc = GetMinerChain()
	)

	logging.Logger.Debug("handle vt. msg - verification ticket",
		zap.Int64("round", bvt.Round),
		zap.String("block", bvt.BlockID))

	if mc.GetMinerRound(rn-1) == nil {
		logging.Logger.Error("handle vt. msg -- no previous round (ignore)",
			zap.Int64("round", rn), zap.Int64("pr", rn-1))
		return nil, nil
	}

	b, err := mc.GetBlock(ctx, bvt.BlockID)
	if err == nil {
		var lfb = mc.GetLatestFinalizedBlock()
		if b.Round < lfb.Round {
			logging.Logger.Debug("verification message (round mismatch)",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Int64("lfb", lfb.Round))
			return nil, nil
		}
	}

	var mr = mc.getOrCreateRound(ctx, rn)
	if mr == nil {
		logging.Logger.Error("handle vt. msg -- can't create round for ticket",
			zap.Int64("round", rn))
		return nil, nil
	}

	// check if the ticket has already verified
	if mr.IsTicketCollected(&bvt.VerificationTicket) {
		logging.Logger.Debug("handle vt. msg -- ticket already collected",
			zap.Int64("round", rn), zap.String("block", bvt.BlockID))
		return nil, nil
	}

	if err := node.ValidateSenderSignature(ctx); err != nil {
		return nil, err
	}

	msg := NewBlockMessage(MessageVerificationTicket, node.GetSender(ctx), nil, nil)
	msg.BlockVerificationTicket = bvt
	mc.PushBlockMessageChannel(msg)
	return nil, nil
}

// NotarizationReceiptHandler - handles the receipt of a notarization
// for a block.
func NotarizationReceiptHandler(ctx context.Context, entity datastore.Entity) (
	interface{}, error) {

	var notarization, ok = entity.(*Notarization)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}

	var (
		mc  = GetMinerChain()
		lfb = mc.GetLatestFinalizedBlock()
	)

	if notarization.Round < lfb.Round {
		logging.Logger.Debug("notarization receipt handler",
			zap.Int64("round", notarization.Round),
			zap.Int64("lf_round", lfb.Round))
		return nil, nil
	}

	b, _ := mc.GetBlock(ctx, notarization.BlockID)
	if b != nil && b.IsBlockNotarized() && b.IsStateComputed() {
		return nil, nil
	}

	if mc.isNotarizing(notarization.BlockID) {
		return nil, nil
	}

	if err := node.ValidateSenderSignature(ctx); err != nil {
		return nil, err
	}

	var msg = NewBlockMessage(MessageNotarization, node.GetSender(ctx), nil, nil)
	msg.Notarization = notarization
	mc.PushBlockMessageChannel(msg)
	return nil, nil
}

// NotarizedBlockHandler - handles a notarized block.
func NotarizedBlockHandler(ctx context.Context, entity datastore.Entity) (
	resp interface{}, err error) {

	var nb, ok = entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}

	mc := GetMinerChain()

	//reject cur_round -2 is a locked round, there can't be new notarization important for us
	if nb.Round < mc.GetCurrentRound()-1 {
		logging.Logger.Debug("notarized block handler (round older than the current round)",
			zap.String("block", nb.Hash), zap.Any("round", nb.Round))
		return
	}

	var lfb = mc.GetLatestFinalizedBlock()
	if nb.Round <= lfb.Round {
		return // doesn't need the not. block
	}

	//TODO in case there is no previous round create it, since notarization can't be rejected
	if mc.GetMinerRound(nb.Round-1) == nil {
		logging.Logger.Error("not. block handler -- no previous round (ignore)",
			zap.Int64("round", nb.Round), zap.Int64("prev_round", nb.Round-1))
		return // no previous round
	}

	//this check is not correct, we won't transit to the new round, but should save notarization block
	//if mc.isAheadOfSharders(ctx, nb.Round) {
	//	return
	//}

	mr := mc.GetMinerRound(nb.Round)
	if mr != nil {
		if mr.IsFinalizing() || mr.IsFinalized() {
			return // doesn't need a not. block
		}

		//it does not matter, we should add notarization even for complete round
		//if mr.IsVerificationComplete() {
		//	return // verification for the round complete
		//}

		for _, blk := range mr.GetNotarizedBlocks() {
			if blk.Hash == nb.Hash {
				return // already have
			}
		}
	}

	if err = node.ValidateSenderSignature(ctx); err != nil {
		return
	}

	var msg = &BlockMessage{
		Sender: node.GetSender(ctx),
		Type:   MessageNotarizedBlock,
		Block:  nb,
	}

	mc.PushBlockMessageChannel(msg)
	return nil, nil
}

// NotarizedBlockSendHandler - handles a request for a notarized block.
func NotarizedBlockSendHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return getNotarizedBlock(ctx, r)
}

// PartialStateHandler - return the partial state from a given root.
func PartialStateHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	n := r.FormValue("node")
	mc := GetMinerChain()
	nodeKey, err := hex.DecodeString(n)
	if err != nil {
		return nil, err
	}
	ps, err := mc.GetStateFrom(ctx, nodeKey)
	if err != nil {
		logging.Logger.Error("partial state handler", zap.String("key", n), zap.Error(err))
		return nil, err
	}
	return ps, nil
}

func getNotarizedBlock(ctx context.Context, req *http.Request) (*block.Block, error) {

	var (
		r    = req.FormValue("round")
		hash = req.FormValue("block")

		mc = GetMinerChain()
		cr = mc.GetCurrentRound()
	)

	errBlockNotAvailable := common.NewError("block_not_available",
		fmt.Sprintf("Requested block is not available, current round: %d, request round: %s, request hash: %s",
			cr, r, hash))

	if hash != "" {
		b, err := mc.GetBlock(ctx, hash)
		if err != nil {
			return nil, err
		}

		if b.IsBlockNotarized() {
			return b, nil
		}
		return nil, errBlockNotAvailable
	}

	if r == "" {
		return nil, common.NewError("none_round_or_hash_provided",
			"no block hash or round number is provided")
	}

	roundN, err := strconv.ParseInt(r, 10, 64)
	if err != nil {
		return nil, err
	}

	rd := mc.GetRound(roundN)
	if rd == nil {
		return nil, errBlockNotAvailable
	}

	b := rd.GetHeaviestNotarizedBlock()
	if b == nil {
		return nil, errBlockNotAvailable
	}

	return b, nil
}
