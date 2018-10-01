package sharder

import (
	"context"

	"0chain.net/round"
	"0chain.net/transaction"

	"0chain.net/blockstore"
	"0chain.net/config"

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
	fr := sc.GetRound(b.Round)
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("lf_round", sc.LatestFinalizedBlock.Round), zap.Any("current_round", sc.CurrentRound))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	sc.BlockCache.Add(b.Hash, b)
	sc.cacheBlockTxns(b.Hash, b.Txns)
	err := blockstore.GetStore().Write(b)
	if err != nil {
		Logger.Error("block save", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	}
	if fr != nil {
		fr.Finalize(b)
		frImpl, _ := fr.(*round.Round)
		err := sc.StoreRound(ctx, frImpl)
		if err != nil {
			Logger.Error("db error (save round)", zap.Int64("round", fr.GetRoundNumber()), zap.Error(err))
		}
		sc.GetRoundChannel() <- frImpl
	} else {
		Logger.Debug("round - missed", zap.Int64("round", b.Round))
	}
}

func (sc *Chain) cacheBlockTxns(hash string, txns []*transaction.Transaction) {
	for _, txn := range txns {
		txnSummary := txn.GetSummary()
		txnSummary.BlockHash = hash
		sc.BlockTxnCache.Add(txn.Hash, txnSummary)
	}
}
