package miner

/*This file contains the Miner To Miner send/receive messages */
import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
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
func SetupM2MReceivers() {
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
		Logger.Info("VRFShare: returning invalid Entity")
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
		Logger.Info("Rejecting VRFShare: old round",
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
			Logger.Info("Rejecting VRFShare: missing miner round",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
		if mr.Block == nil || !mr.Block.IsBlockNotarized() {
			Logger.Info("Rejecting VRFShare: missing HNB for the round"+
				" or it's not notarized",
				zap.Bool("is_not_notarized", mr.Block != nil),
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
		// var hnb = mr.GetHeaviestNotarizedBlock()
		var hnb = mr.Block
		if hnb.GetStateStatus() != block.StateSuccessful {
			Logger.Info("Rejecting VRFShare: HNB state is not successful",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()),
				zap.String("hash", hnb.Hash))
			return nil, nil
		}
		var (
			mb    = mc.GetMagicBlock(vrfs.Round)
			party = node.GetSender(ctx)
		)
		if mb == nil {
			Logger.Info("Rejecting VRFShare: missing MB for the round",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}
		if party == nil {
			Logger.Info("Rejecting VRFShare: missing party",
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
			Logger.Info("Rejecting VRFShare: missing party in MB",
				zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()))
			return nil, nil
		}

		// send verify block message, then send notarized block
		go func() {
			mb.Miners.SendTo(VerifyBlockSender(hnb), found.ID)
			mb.Miners.SendTo(MinerNotarizedBlockSender(hnb), found.ID)
		}()

		Logger.Info("Rejecting VRFShare: push not. block message for the miner behind",
			zap.Int64("vrfs_round_num", vrfs.GetRoundNumber()),
			zap.String("to_miner_id", found.ID),
			zap.String("to_miner_url", found.GetN2NURLBase()))
		return nil, nil
	}

	var msg = NewBlockMessage(MessageVRFShare, node.GetSender(ctx), nil, nil)
	vrfs.SetParty(msg.Sender)
	msg.VRFShare = vrfs
	mc.GetBlockMessageChannel() <- msg
	return nil, nil
}

// VerifyBlockHandler - verify the block that is received.
func VerifyBlockHandler(ctx context.Context, entity datastore.Entity) (
	interface{}, error) {

	var b, ok = entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}

	var mc = GetMinerChain()
	if b.MinerID == node.Self.Underlying().GetKey() {
		return nil, nil
	}
	var lfb = mc.GetLatestFinalizedBlock()
	if b.Round < lfb.Round {
		Logger.Debug("verify block handler", zap.Int64("round", b.Round), zap.Int64("lf_round", lfb.Round))
		return nil, nil
	}

	var err error
	if err = b.Validate(ctx); err != nil {
		Logger.Debug("verify block handler -- can't validate",
			zap.Int64("round", b.Round), zap.Error(err))
		return nil, err
	}

	var msg = NewBlockMessage(MessageVerify, node.GetSender(ctx), nil, b)
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
		Logger.Debug("notarization receipt handler",
			zap.Int64("round", notarization.Round),
			zap.Int64("lf_round", lfb.Round))
		return nil, nil
	}

	var msg = NewBlockMessage(MessageNotarization, node.GetSender(ctx), nil, nil)
	msg.Notarization = notarization
	mc.GetBlockMessageChannel() <- msg
	return nil, nil
}

// NotarizedBlockHandler - handles a notarized block.
func NotarizedBlockHandler(ctx context.Context, entity datastore.Entity) (
	resp interface{}, err error) {

	var b, ok = entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}

	var mc = GetMinerChain()
	if b.Round < mc.GetCurrentRound()-1 {
		Logger.Debug("notarized block handler (round older than the current round)",
			zap.String("block", b.Hash), zap.Any("round", b.Round))
		return
	}

	var r = mc.getOrStartRoundNotAhead(ctx, b.Round)
	if r == nil {
		if mc.isAheadOfSharders(ctx, b.Round) {
			Logger.Debug("notarized block handler -- is ahead or no pr",
				zap.String("block", b.Hash), zap.Any("round", b.Round),
				zap.Bool("has_pr", mc.GetMinerRound(b.Round-1) != nil))
			return
		}
		return // can't handle yet
	}

	if r.IsFinalizing() || r.IsFinalized() {
		return // doesn't need a not. block
	}

	var lfb = mc.GetLatestFinalizedBlock()
	if b.Round <= lfb.Round {
		return nil, nil // doesn't need the not. block
	}

	if mc.GetMinerRound(b.Round-1) == nil {
		Logger.Error("not. block handler -- no previous round (ignore)",
			zap.Int64("round", b.Round), zap.Int64("prev_round", b.Round-1))
		return nil, nil // no previous round
	}

	if err := mc.VerifyNotarization(ctx, b, b.GetVerificationTickets(),
		r.GetRoundNumber()); err != nil {
		return nil, err
	}

	if r.GetRandomSeed() == 0 {
		mc.SetRandomSeed(r, b.GetRoundRandomSeed())
	}

	var msg = &BlockMessage{
		Sender: node.GetSender(ctx),
		Type:   MessageNotarizedBlock,
		Block:  b,
	}
	mc.GetBlockMessageChannel() <- msg
	return nil, nil
}

// NotarizedBlockSendHandler - handles a request for a notarized block.
func NotarizedBlockSendHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return getNotarizedBlock(ctx, r)
}

// BlockStateChangeHandler - provide the state changes associated with a block.
func BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {

	var b, err = getNotarizedBlock(ctx, r)
	if err != nil {
		return nil, err
	}

	if b.GetStateStatus() != block.StateSuccessful {
		return nil, common.NewError("state_not_verified",
			"state is not computed and validated locally")
	}

	var bsc = block.NewBlockStateChange(b)
	if state.Debug() {
		Logger.Info("block state change handler", zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int("state_changes", len(b.ClientState.GetChangeCollector().GetChanges())),
			zap.Int("sc_nodes", len(bsc.Nodes)))
	}

	return bsc, nil
}

// PartialStateHandler - return the partial state from a given root.
func PartialStateHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	node := r.FormValue("node")
	mc := GetMinerChain()
	nodeKey, err := hex.DecodeString(node)
	if err != nil {
		return nil, err
	}
	ps, err := mc.GetStateFrom(ctx, nodeKey)
	if err != nil {
		Logger.Error("partial state handler", zap.String("key", node), zap.Error(err))
		return nil, err
	}
	return ps, nil
}

func getNotarizedBlock(ctx context.Context, r *http.Request) (*block.Block, error) {

	var (
		round = r.FormValue("round")
		hash  = r.FormValue("block")

		mc = GetMinerChain()
	)

	errBlockNotAvailable := common.NewError("block_not_available",
		fmt.Sprintf("Requested block is not available, current round: %d, request round: %d, request hash: %s",
			mc.GetCurrentRound(), round, hash))

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

	if round == "" {
		return nil, common.NewError("none_round_or_hash_provided",
			"no block hash or round number is provided")
	}

	roundN, err := strconv.ParseInt(round, 10, 63)
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
