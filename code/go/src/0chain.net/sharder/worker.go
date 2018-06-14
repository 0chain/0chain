package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers() {
	ClearWorkerState()
	ctx := common.GetRootContext()
	go GetSharderChain().BlockWorker(ctx)
}

//TODO: The blocks and rounds data structures are temporary for debugging.
var blocks map[string]*block.Block
var rounds map[int64]*round.Round

/*ClearWorkerState - clears the worker state */
func ClearWorkerState() {
	Logger.Info("clearing worker state")
	blocks = make(map[string]*block.Block)
	rounds = make(map[int64]*round.Round)
}

/*BlockWorker - stores the blocks */
func (sc *Chain) BlockWorker(ctx context.Context) {

	for true {
		select {
		case <-ctx.Done():
			return
		case b := <-sc.GetBlockChannel():
			_, ok := blocks[b.Hash]
			if ok {
				Logger.Info("block already received", zap.Any("round", b.Round), zap.Any("block", b.Hash))
				continue
			}
			blocks[b.Hash] = b
			er, ok := rounds[b.Round]
			if ok {
				nb := er.GetNotarizedBlocks()
				if len(nb) > 0 {
					Logger.Error("*** different blocks for the same round ***", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("existing_block", nb[0].Hash))
				}
			} else {
				er = datastore.GetEntityMetadata("round").Instance().(*round.Round)
				er.Number = b.Round
				er.RandomSeed = b.RoundRandomSeed
				rounds[er.Number] = er
			}
			er.AddNotarizedBlock(b)
			StoreBlock(ctx, b)
		}
	}
}
