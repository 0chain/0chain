package miner

/*This file contains the Miner To Miner send/receive messages */
import (
	"context"
	"net/http"
	"strconv"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*RoundStartSender - Start a new round */
var RoundStartSender node.EntitySendHandler

/*VerifyBlockSender - Send the block to a node */
var VerifyBlockSender node.EntitySendHandler

/*VerificationTicketSender - Send a verification ticket to a node */
var VerificationTicketSender node.EntitySendHandler

/*BlockNotarizationSender - Send the block notarization to a node */
var BlockNotarizationSender node.EntitySendHandler

/*MinerNotarizedBlockSender - Send a notarized block to a node*/
var MinerNotarizedBlockSender node.EntitySendHandler

/*SetupM2MSenders - setup senders for miner to miner communication */
func SetupM2MSenders() {

	options := &node.SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	RoundStartSender = node.SendEntityHandler("/v1/_m2m/round/start", options)

	options = &node.SendOptions{Timeout: 2 * time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	VerifyBlockSender = node.SendEntityHandler("/v1/_m2m/block/verify", options)
	MinerNotarizedBlockSender = node.SendEntityHandler("/v1/_m2m/block/notarized_block", options)

	options = &node.SendOptions{MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	VerificationTicketSender = node.SendEntityHandler("/v1/_m2m/block/verification_ticket", options)

	options = &node.SendOptions{Timeout: time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, CODEC: node.CODEC_MSGPACK, Compress: true}
	BlockNotarizationSender = node.SendEntityHandler("/v1/_m2m/block/notarization", options)
}

/*SetupM2MReceivers - setup receivers for miner to miner communication */
func SetupM2MReceivers() {
	// TODO: This is going to abstract the random beacon for now
	http.HandleFunc("/v1/_m2m/round/start", node.ToN2NReceiveEntityHandler(StartRoundHandler))

	http.HandleFunc("/v1/_m2m/block/verify", node.ToN2NReceiveEntityHandler(memorystore.WithConnectionEntityJSONHandler(VerifyBlockHandler, datastore.GetEntityMetadata("block"))))
	http.HandleFunc("/v1/_m2m/block/verification_ticket", node.ToN2NReceiveEntityHandler(VerificationTicketReceiptHandler))
	http.HandleFunc("/v1/_m2m/block/notarization", node.ToN2NReceiveEntityHandler(NotarizationReceiptHandler))
	http.HandleFunc("/v1/_m2m/block/notarized_block", node.ToN2NReceiveEntityHandler(NotarizedBlockHandler))
}

/*SetupX2MResponders - setup responders */
func SetupX2MResponders() {
	http.HandleFunc("/v1/_x2m/block/notarized_block/get", node.ToN2NSendEntityHandler(NotarizedBlockSendHandler))
}

/*StartRoundHandler - handles the starting of a new round */
func StartRoundHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	r, ok := entity.(*round.Round)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	if r.Number < mc.LatestFinalizedBlock.Round {
		return false, nil
	}
	mr := mc.CreateRound(r)
	msg := NewBlockMessage(MessageStartRound, node.GetSender(ctx), mr, nil)
	mc.GetBlockMessageChannel() <- msg
	return true, nil
}

/*VerifyBlockHandler - verify the block that is received */
func VerifyBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	if b.MinerID == mc.ID {
		return nil, nil
	}
	if b.Round < mc.LatestFinalizedBlock.Round {
		Logger.Debug("verify block handler", zap.Int64("round", b.Round), zap.Int64("lf_round", mc.LatestFinalizedBlock.Round))
		return true, nil
	}
	msg := NewBlockMessage(MessageVerify, node.GetSender(ctx), nil, b)
	mc.GetBlockMessageChannel() <- msg
	return true, nil
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
	return true, nil
}

/*NotarizationReceiptHandler - handles the receipt of a notarization for a block */
func NotarizationReceiptHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	notarization, ok := entity.(*Notarization)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	msg := NewBlockMessage(MessageNotarization, node.GetSender(ctx), nil, nil)
	msg.Notarization = notarization
	GetMinerChain().GetBlockMessageChannel() <- msg
	return true, nil
}

/*NotarizedBlockHandler - handles a notarized block*/
func NotarizedBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	mc := GetMinerChain()
	Logger.Info("notarized block handler", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("current_round", mc.CurrentRound))
	if b.Round < mc.CurrentRound-1 {
		Logger.Debug("notarized block handler (round older than the current round)", zap.String("block", b.Hash), zap.Any("round", b.Round))
		return true, nil
	}
	if err := mc.VerifyNotarization(ctx, b.Hash, b.VerificationTickets); err != nil {
		return nil, err
	}
	msg := &BlockMessage{Sender: node.GetSender(ctx), Type: MessageNotarizedBlock, Block: b}
	mc.GetBlockMessageChannel() <- msg
	return true, nil
}

//NotarizedBlockSendHandler - handles a request for a notarized block
func NotarizedBlockSendHandler(ctx context.Context, r *http.Request) (interface{}, error) {
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
			b := r.GetBestNotarizedBlock()
			if b != nil {
				return b, nil
			}
		}
	} else if hash != "" {
		b, err := mc.GetBlock(ctx, hash)
		if err != nil {
			return nil, err
		}
		if mc.IsBlockNotarized(ctx, b) {
			return b, nil
		}
	} else {
		for r := mc.GetRound(mc.CurrentRound); r != nil; r = mc.GetRound(r.Number - 1) {
			b := r.GetBestNotarizedBlock()
			if b != nil {
				return b, nil
			}
		}
	}
	return nil, common.NewError("block_not_available", "Requested block is not available")
}
