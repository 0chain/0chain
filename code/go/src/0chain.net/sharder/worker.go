package sharder

import (
	"context"

	"0chain.net/blockstore"

	"0chain.net/node"

	. "0chain.net/logging"
	"go.uber.org/zap"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers(ctx context.Context) {
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)                 // 1) receives incoming blocks from the network
	go sc.BlockFinalizationWorker(ctx, sc) // 2) sequentially runs finalization logic
	go sc.BlockStorageWorker(ctx)          // 3) persists the blocks
}

/*BlockWorker - stores the blocks */
func (sc *Chain) BlockWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case b := <-sc.GetBlockChannel():
			sc.processBlock(ctx, b)
		}
	}
}

/*BlockStorageWorker - a background worker that processes a block to store it in suitable formats */
func (sc *Chain) BlockStorageWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case r := <-sc.GetRoundChannel():
			b, err := sc.GetBlockFromHash(ctx, r.BlockHash, r.GetRoundNumber())
			if err != nil {
				Logger.Error("failed to get block", zap.String("blockhash", r.BlockHash), zap.Error(err))
			} else {
				sc.StoreTransactions(ctx, b)
				err = sc.StoreBlockSummary(ctx, b)
				if err != nil {
					Logger.Error("db error (save block)", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
				}
				self := node.GetSelfNode(ctx)
				if !sc.IsBlockSharder(r, b, self.Node) {
					err = blockstore.GetStore().DeleteBlock(b)
					if err != nil {
						Logger.Error("failed to delete block from file system", zap.Any("round", b.Round), zap.String("blockhash", b.Hash), zap.Error(err))
					}
				}
			}
			sc.DeleteRoundsBelow(ctx, r.GetRoundNumber()-10)
		}
	}
}
