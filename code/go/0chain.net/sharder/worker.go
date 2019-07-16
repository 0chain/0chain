package sharder

import (


	"0chain.net/chaincore/block"

	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/persistencestore"
	"context"
)

/*SetupWorkers - setup the background workers */
func SetupWorkers(ctx context.Context) {
	sc := GetSharderChain()
	go sc.BlockWorker(ctx)              // 1) receives incoming blocks from the network
	go sc.FinalizeRoundWorker(ctx, sc)  // 2) sequentially finalize the rounds
	go sc.FinalizedBlockWorker(ctx, sc) // 3) sequentially processes finalized blocks
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


///*QOSWorker - gets most recent K rounds and stores them*/
//func (sc *Chain) QOSWorker(ctx context.Context) {
//	for true {
//		select {
//		case <-ctx.Done():
//			return
//		default:
//			lr := sc.LatestFinalizedBlock.Round
//			sc.processLastNBlocks(ctx, lr, sc.BatchSyncSize)
//		}
//	}
//}






//func (sc *Chain) processLastNBlocks(ctx context.Context, lr int64, n int) {
//	self := node.GetSelfNode(ctx)
//	var r *round.Round
//	var bs *block.BlockSummary
//	var hasEntity bool
//
//	for i := 0; i < n; i++ {
//		currR := lr - int64(i)
//		sc.SharderStats.QOSRound = currR
//		if currR < 1 {
//			return
//		}
//		r, hasEntity = sc.hasRoundSummary(ctx, currR)
//		if !hasEntity {
//			params := &url.Values{}
//			params.Add("round", strconv.FormatInt(currR, 10))
//			params.Add("range", strconv.Itoa(-n)) // we go backwards so it is a minus
//			rs := sc.requestForRoundSummaries(ctx, params)
//			if rs != nil {
//				sc.storeRoundSummaries(ctx, rs)
//				r, _ = sc.hasRoundSummary(ctx, lr)
//			}
//		}
//		if r == nil || r.BlockHash == "" { // if we do not have the round or blockhash then continue
//			continue
//		}
//		bs, hasEntity = sc.hasBlockSummary(ctx, r.BlockHash)
//		if !hasEntity {
//			params := &url.Values{}
//			params.Add("round", strconv.FormatInt(currR, 10))
//			params.Add("range", strconv.Itoa(-n))
//			bs := sc.requestForBlockSummaries(ctx, params)
//			if bs != nil {
//				sc.storeBlockSummaries(ctx, bs)
//			}
//		}
//		var b *block.Block
//		n := sc.GetActivesetSharder(self.GNode)
//		canShard := sc.IsBlockSharderFromHash(bs.Hash, n)
//		b, hasEntity = sc.hasBlock(bs.Hash, r.Number)
//		if hasEntity == false {
//			b = sc.requestBlock(ctx, r)
//			if b == nil {
//				Logger.Info("health-check: MissingObject",
//					zap.String("object", "Block"),
//					zap.Int64("cycle", bss.CycleCount),
//					zap.Int64("round", r.Number),
//					zap.String("hash", r.BlockHash))
//				return MissingBlock
//			}
//			if canShard {
//				// Save the block
//				err := sc.storeBlock(ctx, b)
//				if err != nil {
//					Logger.Error("health-check: DataStoreWriteFailure",
//						zap.String("object", "block"),
//						zap.Int64("cycle", bss.CycleCount),
//						zap.Int64("round", r.Number),
//						zap.Error(err))
//				}
//			}
//
//			b = sc.syncBlock(ctx, r, canShard)
//		}
//
//		// Check to store transaction summary.
//		if sc.hasBlockTransactions(ctx, b) == false {
//			// The block has transactions and may need to be stored.
//			err := sc.storeBlockTransactions(ctx, b)
//			if err != nil {
//				Logger.Error("health-check: DataStoreWriteFailure",
//					zap.String("object", "TransactionSummary"),
//					zap.Int64("cycle", bss.CycleCount),
//					zap.Int64("round", bs.Round),
//					zap.Int("txn-count", bs.NumTxns),
//					zap.String("block-hash", bs.Hash),
//					zap.Error(err))
//				return MissingTxnSummary
//			}
//		}
//		return BlockSuccess
//
//		hasTxns := sc.hasTransactions(ctx, bs)
//		if !hasTxns {
//			params := &url.Values{}
//			params.Add("round", strconv.FormatInt(r.Number, 10))
//			params.Add("hash", r.BlockHash)
//			if b == nil {
//				b = sc.requestForBlock(ctx, params, r)
//			} else {
//				sc.storeBlockTransactions(ctx, b)
//			}
//		}
//	}
//}

func (sc *Chain) hasRoundSummary(ctx context.Context, rNum int64) (*round.Round, bool) {
	r, err := sc.GetRoundFromStore(ctx, rNum)
	if err == nil {
		return r, true
	}
	return nil, false
}

func (sc *Chain) hasBlockSummary(ctx context.Context, bHash string) (*block.BlockSummary, bool) {
	bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	defer ememorystore.Close(bctx)
	bs, err := sc.GetBlockSummary(bctx, bHash)
	if err == nil {
		return bs, true
	}
	return nil, false
}

func (sc *Chain) hasBlock(bHash string, rNum int64) (*block.Block, bool) {
	b, err := sc.GetBlockFromStore(bHash, rNum)
	if err == nil {
		return b, true
	}
	return nil, false
}

func (sc *Chain) hasBlockTransactions(ctx context.Context, b *block.Block) bool {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryEntityMetadata)
	defer persistencestore.Close(tctx)
	for _, txn := range b.Txns {
		_, err := sc.GetTransactionSummary(tctx, txn.Hash)
		if err != nil {
			return false
		}
	}
	return true
}

func (sc *Chain) hasTransactions(ctx context.Context, bs *block.BlockSummary) bool {
	if bs == nil {
		return false
	}
	count, err := sc.getTxnCountForRound(ctx, bs.Round)
	if err != nil {
		return false
	}
	return count == bs.NumTxns
}
//func (sc *Chain) hasTransactions(ctx context.Context, bs *block.BlockSummary) bool {
//	if bs == nil || bs.NumTxns == 0 {
//		return false
//	} else {
//		return true
//	}
//}
func (sc *Chain) blockContainsTransactions(ctx context.Context, bs *block.BlockSummary) bool {
	if bs == nil || bs.NumTxns == 0 {
		return false
	} else {
		return true
	}
}
