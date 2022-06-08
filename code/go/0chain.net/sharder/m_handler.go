package sharder

import (
	"context"
	"net/http"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*SetupM2SReceivers - setup handlers for all the messages received from the miner */
func SetupM2SReceivers() {
	sc := GetSharderChain()
	options := &node.ReceiveOptions{}
	options.MessageFilter = sc
	http.HandleFunc("/v1/_m2s/block/finalized", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(FinalizedBlockHandler(sc), options)))
	http.HandleFunc("/v1/_m2s/block/notarized", common.N2NRateLimit(node.RejectDuplicateNotarizedBlockHandler(
		sc, node.ToN2NReceiveEntityHandler(NotarizedBlockHandler(sc), options))))
	http.HandleFunc("/v1/_m2s/block/notarized/kick", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(NotarizedBlockKickHandler(sc), nil)))
}

//go:generate mockery --inpackage --testonly --name=Chainer --case=underscore
type Chainer interface {
	GetCurrentRound() int64
	GetLatestFinalizedBlock() *block.Block
	GetBlock(ctx context.Context, hash datastore.Key) (*block.Block, error)
	PushToBlockProcessor(b *block.Block) error
	ForceFinalizeRound()
}

//AcceptMessage - implement the node.MessageFilterI interface
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

/*SetupM2SResponders - setup handlers for all the requests from the miner */
func SetupM2SResponders(sc Chainer) {
	http.HandleFunc("/v1/_m2s/block/latest_finalized/get", common.N2NRateLimit(
		node.ToS2MSendEntityHandler(LatestFinalizedBlockHandler(sc))))
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
	sc.pbMutex.RLock()
	_, err := sc.processingBlocks.Get(hash)
	sc.pbMutex.RUnlock()
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

func (sc *Chain) cacheProcessingBlock(hash string) bool {
	sc.pbMutex.Lock()
	_, err := sc.processingBlocks.Get(hash)
	switch err {
	case cache.ErrKeyNotFound:
		if err := sc.processingBlocks.Add(hash, struct{}{}); err != nil {
			Logger.Warn("cache process block failed",
				zap.String("block", hash),
				zap.Error(err))
			sc.pbMutex.Unlock()
			return false
		}
		sc.pbMutex.Unlock()
		return true
	default:
	}
	sc.pbMutex.Unlock()
	return false
}

func (sc *Chain) removeProcessingBlock(hash string) {
	sc.pbMutex.Lock()
	sc.processingBlocks.Remove(hash)
	sc.pbMutex.Unlock()
}

/*

func (sc *Chain) getBlockStateChangeByBlock(b *block.Block) (
	bsc *block.StateChange, err error) {

	// we can't check the notarization

	if len(b.ClientStateHash) == 0 {
		return nil, common.NewError("handle_get_block_state", "DEBUG ERROR")
	}

	if b.ClientState == nil {
		if err = sc.InitBlockState(b); err != nil {
			return nil, common.NewErrorf("handle_get_block_state",
				"can't initialize block state %d (%s): %v", b.Round, b.Hash,
				err)
		}
	}

	return block.NewBlockStateChange(b), nil
}

// BlockStateChangeHandler requires both 'block' (hash) and 'round' query
// parameters. The round required to get block from store.
func BlockStateChangeHandler(ctx context.Context, r *http.Request) (
	resp interface{}, err error) {

	var sc = GetSharderChain()
	// 1. get block first
	// :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: :: //
	var (
		roundQuery = r.FormValue("round")
		hash       = r.FormValue("block")

		lfb = sc.GetLatestFinalizedBlock()

		rn int64 // round number
	)

	// check round query parameter

	if roundQuery == "" {
		return nil, common.NewError("handle_get_block_state",
			"missing 'round' query parameter")
	}

	// parse round query parameter

	if rn, err = strconv.ParseInt(roundQuery, 10, 64); err != nil {
		return nil, common.NewErrorf("handle_get_block_state",
			"can't parse 'round' query parameter: %v", err)
	}

	if rn > lfb.Round {
		return nil, common.NewErrorf("handle_get_block_state",
			"requested block is newer than lfb %d, want %d", lfb.Round, rn)
	}

	// check hash query parameter

	if hash == "" {
		if hash, err = sc.GetBlockHash(ctx, rn); err != nil {
			return nil, common.NewErrorf("handle_get_block_state",
				"can't get block hash by round number: %v", err)
		}
	}

	// try to get block by hash first (fresh blocks short circuit)

	var b *block.Block
	if b, err = sc.GetBlock(ctx, hash); err == nil {
		// block found, ignore round query parameter
		return sc.getBlockStateChangeByBlock(b)
	}

	// so, if we haven't block, then we should get it from store using
	// round number

	if b, err = sc.GetBlockFromStore(hash, rn); err != nil {
		return nil, common.NewErrorf("handle_get_block_state",
			"no such block (%s, %d): %v", hash, rn, err)
	}

	return sc.getBlockStateChangeByBlock(b)
}

*/

/*LatestFinalizedBlockHandler - handle latest finalized block*/
func LatestFinalizedBlockHandler(c Chainer) common.JSONResponderF {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		return c.GetLatestFinalizedBlock(), nil
	}
}
