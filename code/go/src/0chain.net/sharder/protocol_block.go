package sharder

import (
	"context"
	"sort"

	"0chain.net/blockstore"

	"0chain.net/block"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

/*UpdatePendingBlock - update the pending block */
func (sc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {

}

/*UpdateFinalizedBlock - updates the finalized block */
func (sc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("lf_round", sc.LatestFinalizedBlock.Round), zap.Any("current_round", sc.CurrentRound), zap.Any("blocks_size", len(sc.Blocks)), zap.Any("rounds_size", len(sc.rounds)))
	// Sort transactions by their hash - useful for quick search
	sort.SliceStable(b.Txns, func(i, j int) bool { return b.Txns[i].Hash < b.Txns[j].Hash })
	err := blockstore.GetStore().Write(b)
	if err != nil {
		Logger.Error("block save", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	}
	fr := sc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
		err := sc.StoreRound(ctx, fr)
		if err != nil {
			Logger.Error("db error (save round)", zap.Int64("round", fr.Number), zap.Error(err))
		}
		sc.GetRoundChannel() <- fr
		sc.DeleteRoundsBelow(ctx, fr.Number)
	}
}
