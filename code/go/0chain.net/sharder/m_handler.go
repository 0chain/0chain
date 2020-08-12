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
	println("KICK HANDLER")
	sc := GetSharderChain()
	b, ok := entity.(*block.Block)
	if !ok {
		println("KICK HANDLER: INVALID ENTITY")
		return nil, common.InvalidRequest("Invalid Entity")
	}
	var lfb = sc.GetLatestFinalizedBlock()
	if b.Round <= lfb.Round {
		println("KICK HANDLER: ROUND < LFB ROUND")
		return true, nil // doesn't need a not. block for the round
	}
	println("KICK HANDLER: SEND TO BLOCK CHANNEL")
	sc.GetBlockChannel() <- b // even if we have the block
	return true, nil
}

/*LatestFinalizedBlockHandler - handle latest finalized block*/
func LatestFinalizedBlockHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return GetSharderChain().GetLatestFinalizedBlock(), nil
}
