package sharder

import (
	"context"
	"net/http"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

/*SetupM2SReceivers - setup handlers for all the messages received from the miner */
func SetupM2SReceivers() {
	sc := GetSharderChain()
	options := &node.ReceiveOptions{}
	options.MessageFilter = sc
	http.HandleFunc("/v1/_m2s/block/finalized", common.N2NRateLimit(node.StopOnBlockSyncingHandler(sc,
		node.ToN2NReceiveEntityHandler(FinalizedBlockHandler(sc), options))))
	http.HandleFunc("/v1/_m2s/block/notarized", common.N2NRateLimit(node.StopOnBlockSyncingHandler(sc,
		node.RejectDuplicateNotarizedBlockHandler(
			sc, node.ToN2NReceiveEntityHandler(NotarizedBlockHandler(sc), options)))))
	http.HandleFunc("/v1/_m2s/block/notarized/kick", common.N2NRateLimit(node.StopOnBlockSyncingHandler(sc,
		node.ToN2NReceiveEntityHandler(NotarizedBlockKickHandler(sc), nil))))
}

//go:generate mockery --inpackage --testonly --name=Chainer --case=underscore
type Chainer interface {
	GetCurrentRound() int64
	GetLatestFinalizedBlock() *block.Block
	GetBlock(ctx context.Context, hash datastore.Key) (*block.Block, error)
	PushToBlockProcessor(b *block.Block) error
	ForceFinalizeRound()
}

// AcceptMessage - implement the node.MessageFilterI interface
func (sc *Chain) AcceptMessage(entityName string, entityID string) bool {
	switch entityName {
	case "block":
		_, err := sc.GetBlock(common.GetRootContext(), entityID)
		if err != nil {
			return true
		}
		return false
	default:
		return true
	}
}

/*FinalizedBlockHandler - handle the finalized block */
func FinalizedBlockHandler(sc Chainer) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		return NotarizedBlockHandler(sc)(ctx, entity)
	}
}

/*NotarizedBlockHandler - handle the notarized block */
func NotarizedBlockHandler(sc Chainer) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		b, ok := entity.(*block.Block)
		if !ok {
			return nil, common.InvalidRequest("Invalid Entity")
		}

		var lfb = sc.GetLatestFinalizedBlock()
		if b.Round <= lfb.Round {
			Logger.Debug("NotarizedBlockHandler block.Round <= lfb.Round",
				zap.Int64("block round", b.Round),
				zap.Int64("lfb round", lfb.Round))
			return true, nil // doesn't need a not. block for the round
		}

		_, err := sc.GetBlock(ctx, b.Hash)
		if err == nil {
			Logger.Debug("NotarizedBlockHandler block exist", zap.Int64("round", b.Round))
			return true, nil
		}

		if err = node.ValidateSenderSignature(ctx); err != nil {
			return false, err
		}

		if err := sc.PushToBlockProcessor(b); err != nil {
			Logger.Error("NotarizedBlockHandler, push notarized block to channel failed",
				zap.Int64("round", b.Round), zap.Error(err))
			return false, nil
		}

		return true, nil
	}
}

// NotarizedBlockKickHandler - handle the notarized block where the sharder is
// behind miners and don't finalized latest few rounds for a reason.
func NotarizedBlockKickHandler(sc Chainer) datastore.JSONEntityReqResponderF {
	return func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		b, ok := entity.(*block.Block)
		if !ok {
			return nil, common.InvalidRequest("Invalid Entity")
		}
		var lfb = sc.GetLatestFinalizedBlock()
		if b.Round <= lfb.Round {
			return true, nil // doesn't need a not. block for the round
		}

		if err := node.ValidateSenderSignature(ctx); err != nil {
			return false, err
		}

		if err := sc.PushToBlockProcessor(b); err != nil {
			Logger.Debug("Notarized block kick, push block to process channel failed",
				zap.Int64("round", b.Round), zap.Error(err))
		}

		return true, nil
	}
}

// RejectNotarizedBlock returns true if the sharder is being processed or
// the block is already notarized
func (sc *Chain) RejectNotarizedBlock(hash string) bool {
	_, err := sc.GetProcessingBlock(hash)
	switch err {
	case cache.ErrKeyNotFound:
		_, err := sc.GetBlock(context.Background(), hash)
		if err == nil {
			// block is already notarized, reject
			N2n.Debug("reject notarized block", zap.String("hash", hash))
			return true
		}

		return false
	case nil:
		// find the key in the cache
		N2n.Debug("reject notarized block", zap.String("hash", hash))
		return true
	default:
		// should not happen here
		return false
	}
}

// LatestFinalizedBlockHandler - handle latest finalized block
func LatestFinalizedBlockHandler(c Chainer) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		return c.GetLatestFinalizedBlock(), nil
	}
}

/*SetupM2SResponders - setup handlers for all the requests from the miner */
func SetupM2SResponders(sc Chainer) {
	http.HandleFunc("/v1/_m2s/block/latest_finalized/get", common.N2NRateLimit(
		node.ToS2MSendEntityHandler(LatestFinalizedBlockHandler(sc))))
}
