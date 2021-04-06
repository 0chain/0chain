package sharder

import (
	"context"
	"net/http"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

/*SetupM2SReceivers - setup handlers for all the messages received from the miner */
func SetupM2SReceivers() {
	sc := GetSharderChain()
	options := &node.ReceiveOptions{}
	options.MessageFilter = sc
	http.HandleFunc("/v1/_m2s/block/finalized", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(FinalizedBlockHandler, options)))
	http.HandleFunc("/v1/_m2s/block/notarized", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(NotarizedBlockHandler, options)))
	http.HandleFunc("/v1/_m2s/block/notarized/kick", common.N2NRateLimit(node.ToN2NReceiveEntityHandler(NotarizedBlockKickHandler, nil)))
	// http.HandleFunc("/v1/_x2s/block/state_change/get", common.N2NRateLimit(node.ToN2NSendEntityHandler(BlockStateChangeHandler)))
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
func SetupM2SResponders() {
	http.HandleFunc("/v1/_m2s/block/latest_finalized/get", common.N2NRateLimit(node.ToS2MSendEntityHandler(LatestFinalizedBlockHandler)))
}

/*FinalizedBlockHandler - handle the finalized block */
func FinalizedBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	return NotarizedBlockHandler(ctx, entity)
}

/*NotarizedBlockHandler - handle the notarized block */
func NotarizedBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	sc := GetSharderChain()
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	var lfb = sc.GetLatestFinalizedBlock()
	if b.Round <= lfb.Round {
		return true, nil // doesn't need a not. block for the round
	}
	_, err := sc.GetBlock(ctx, b.Hash)
	if err == nil {
		return true, nil
	}
	sc.GetBlockChannel() <- b
	return true, nil
}

// NotarizedBlockKickHandler - handle the notarized block where the sharder is
// behind miners and don't finalized latest few rounds for a reason.
func NotarizedBlockKickHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	sc := GetSharderChain()
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	var lfb = sc.GetLatestFinalizedBlock()
	if b.Round <= lfb.Round {
		return true, nil // doesn't need a not. block for the round
	}
	sc.GetBlockChannel() <- b // even if we have the block
	return true, nil
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
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetSharderChain().GetLatestFinalizedBlock(), nil
}
