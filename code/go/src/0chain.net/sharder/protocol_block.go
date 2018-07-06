package sharder

import (
	"context"
	"sort"
	"time"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/datastore"
	"0chain.net/ememorystore"
	. "0chain.net/logging"
	"0chain.net/persistencestore"
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
	sc.StoreBlock(ctx, b)
	fr := sc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
		sc.DeleteRoundsBelow(ctx, fr.Number)
	}
}

/*StoreBlock - store the block to persistence storage */
func (sc *Chain) StoreBlock(ctx context.Context, b *block.Block) {
	ts := time.Now()
	err := blockstore.GetStore().Write(b)
	if err != nil {
		Logger.Error("block save", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	} else {
		Logger.Info("saved block", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Any("prev_hash", b.PrevHash), zap.Duration("duration", time.Since(ts)))
	}
	bs := b.GetSummary()
	bSummaryEntityMetadata := bs.GetEntityMetadata()
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	err = bs.Write(bctx)
	if err != nil {
		Logger.Error("db error (save block)", zap.String("block", b.Hash), zap.Error(err))
	} else {
		con := ememorystore.GetEntityCon(bctx, bSummaryEntityMetadata)
		err := con.Commit()
		if err != nil {
			Logger.Error("db error (save block)", zap.String("block", b.Hash), zap.Error(err))
		}
	}
	var sTxns = make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		txnSummary := txn.GetSummary()
		txnSummary.BlockHash = b.Hash
		sTxns[idx] = txnSummary
	}
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryMetadata)
	err = txnSummaryMetadata.GetStore().MultiWrite(tctx, txnSummaryMetadata, sTxns)
	if err != nil {
		Logger.Error("db error (save transaction)", zap.String("block", b.Hash), zap.Error(err))
	}
}
