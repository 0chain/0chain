package sharder

import (
	"context"
	"net/http"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/persistencestore"
	"go.uber.org/zap"
)

/*SetupM2SReceivers - setup handlers for all the messages received from the miner */
func SetupM2SReceivers() {
	http.HandleFunc("/v1/_m2s/block/finalized", node.ToN2NReceiveEntityHandler(persistencestore.WithConnectionEntityJSONHandler(FinalizedBlockHandler, datastore.GetEntityMetadata("block"))))
}

/*FinalizedBlockHandler - handle the finalized block */
func FinalizedBlockHandler(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	b, ok := entity.(*block.Block)
	if !ok {
		return nil, common.InvalidRequest("Invalid Entity")
	}
	err := StoreBlock(ctx, b)
	if err != nil {
		Logger.Info("save block failed", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
		return false, err
	}
	sender := node.GetSender(ctx)
	Logger.Info("saved block", zap.Any("sender_set_index", sender.SetIndex), zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Any("prev_hash", b.PrevHash))
	return true, nil
}
