package sharder

import (
	"context"
	"sort"
	"time"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/logging"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

/*UpdatePendingBlock - update the pending block */
func (sc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {

}

/*UpdateFinalizedBlock - updates the finalized block */
func (sc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("lf_round", sc.LatestFinalizedBlock.Round), zap.Any("current_round", sc.CurrentRound), zap.Any("blocks_size", len(sc.Blocks)), zap.Any("rounds_size", len(sc.rounds)))
	if b.Round%100 == 0 {
		if config.Development() || b.Round%1000 == 0 {
			common.LogRuntime(logging.Logger, zap.Int64("round", b.Round))
		}
	}
	// Sort transactions by their hash - useful for quick search
	sort.SliceStable(b.Txns, func(i, j int) bool { return b.Txns[i].Hash < b.Txns[j].Hash })
	sc.StoreBlock(ctx, b)
	fr := sc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
		sc.DeleteRoundsBelow(ctx, fr.Number)
	}
}

/*StoreBlock - store the block to persistence storage */
func (sc *Chain) StoreBlock(ctx context.Context, b *block.Block) error {
	ts := time.Now()
	err := blockstore.GetStore().Write(b)
	if err != nil {
		Logger.Error("block save", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	} else {
		Logger.Info("saved block", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Any("prev_hash", b.PrevHash), zap.Duration("duration", time.Since(ts)))
	}

	// TODO: Store the block summary and transaction summary information
	bs := datastore.GetEntityMetadata("block_summary").Instance().(*block.BlockSummary)
	bs.Hash = b.Hash
	bs.RoundRandomSeed = b.RoundRandomSeed
	bs.PrevHash = b.PrevHash
	bs.Round = b.Round
	/*
		ctx = persistencestore.WithEntityConnection(ctx, bs.GetEntityMetadata())
		store := persistencestore.GetStorageProvider()
			err = store.Write(ctx, bs)
			if err != nil {
				Logger.Error("db save error", zap.Error(err))
			}*/
	return err
}
