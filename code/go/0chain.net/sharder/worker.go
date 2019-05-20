package sharder

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/sharder/blockstore"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	. "0chain.net/core/logging"
	"0chain.net/core/persistencestore"
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

/*HealthCheckWorker - checks the health for each round*/
func (sc *Chain) HealthCheckWorker(ctx context.Context) {
	hr := sc.HealthyRoundNumber
	hRound, err := sc.ReadHealthyRound(ctx)
	Logger.Info("health round from file", zap.Int64("healthy round", hRound.Number))
	if err == nil && hRound.Number > hr {
		hr = hRound.Number
	}
	sc.BSyncStats.SyncBeginR = hr + 1
	for true {
		select {
		case <-ctx.Done():
			return
		default:
			sc.SharderStats.HealthyRoundNum = hr
			hr = hr + 1
			t := time.Now()
			sc.healthCheck(ctx, hr)
			duration := time.Since(t)
			hRound.Number = hr
			err = sc.WriteHealthyRound(ctx, hRound)
			if err != nil {
				Logger.Error("failed to write healthy round", zap.Error(err))
			}
			sc.updateSyncStats(hr, duration)
		}
	}
}

/*QOSWorker - gets most recent K rounds and stores them*/
func (sc *Chain) QOSWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		default:
			lr := sc.LatestFinalizedBlock.Round
			sc.processLastNBlocks(ctx, lr, sc.BatchSyncSize)
		}
	}
}

func (sc *Chain) updateSyncStats(rNum int64, duration time.Duration) {
	var diff int64
	if sc.BSyncStats.CurrSyncR > 0 {
		diff = sc.BSyncStats.SyncUntilR - sc.BSyncStats.CurrSyncR
	} else {
		diff = sc.BSyncStats.SyncUntilR - sc.BSyncStats.SyncBeginR
	}
	if diff <= 0 {
		sc.BSyncStats.Status = SyncDone
	} else {
		sc.BSyncStats.Status = Sync
		BlockSyncTimer.Update(duration)
	}

	if sc.BSyncStats.Status == Sync {
		sc.BSyncStats.CurrSyncR = rNum
		sc.BSyncStats.SyncBlocksCount++
	} else {
		sc.BSyncStats.CurrSyncR = 0
	}
}

func (sc *Chain) healthCheck(ctx context.Context, rNum int64) {
	var r *round.Round
	var bs *block.BlockSummary
	var b *block.Block
	var hasEntity bool

	self := node.GetSelfNode(ctx)

	r, hasEntity = sc.hasRoundSummary(ctx, rNum)
	if !hasEntity {
		r = sc.syncRoundSummary(ctx, rNum, sc.BatchSyncSize)
	}
	bs, hasEntity = sc.hasBlockSummary(ctx, r.BlockHash)
	if !hasEntity {
		bs = sc.syncBlockSummary(ctx, r, sc.BatchSyncSize)
	}
	canShard := sc.IsBlockSharderFromHash(bs.Hash, self.Node)
	if canShard {
		b, hasEntity = sc.hasBlock(bs.Hash, r.Number)
		if !hasEntity {
			b = sc.syncBlock(ctx, r, canShard)
		}
	}
	hasTxns := sc.hasTransactions(ctx, bs)
	if !hasTxns {
		if b == nil {
			b = sc.syncBlock(ctx, r, canShard)
		}
		sc.storeBlockTransactions(ctx, b)
	}
}

func (sc *Chain) processLastNBlocks(ctx context.Context, lr int64, n int) {
	self := node.GetSelfNode(ctx)
	var r *round.Round
	var bs *block.BlockSummary
	var hasEntity bool

	for i := 0; i < n; i++ {
		currR := lr - int64(i)
		sc.SharderStats.QOSRound = currR
		if currR < 1 {
			return
		}
		r, hasEntity = sc.hasRoundSummary(ctx, currR)
		if !hasEntity {
			params := &url.Values{}
			params.Add("round", strconv.FormatInt(currR, 10))
			params.Add("range", strconv.Itoa(-n)) // we go backwards so it is a minus
			rs := sc.requestForRoundSummaries(ctx, params)
			if rs != nil {
				sc.storeRoundSummaries(ctx, rs)
				r, _ = sc.hasRoundSummary(ctx, lr)
			}
		}
		if r == nil || r.BlockHash == "" { // if we do not have the round or blockhash then continue
			continue
		}
		bs, hasEntity = sc.hasBlockSummary(ctx, r.BlockHash)
		if !hasEntity {
			params := &url.Values{}
			params.Add("round", strconv.FormatInt(currR, 10))
			params.Add("range", strconv.Itoa(-n))
			bs := sc.requestForBlockSummaries(ctx, params)
			if bs != nil {
				sc.storeBlockSummaries(ctx, bs)
			}
		}
		var b *block.Block
		canShard := sc.IsBlockSharderFromHash(bs.Hash, self.Node)
		if canShard {
			b, hasEntity = sc.hasBlock(bs.Hash, r.Number)
			if !hasEntity {
				params := &url.Values{}
				params.Add("round", strconv.FormatInt(r.Number, 10))
				params.Add("hash", r.BlockHash)
				b = sc.requestForBlock(ctx, params, r)
				if b != nil {
					blockstore.GetStore().Write(b)
				}
			}
		}
		hasTxns := sc.hasTransactions(ctx, bs)
		if !hasTxns {
			params := &url.Values{}
			params.Add("round", strconv.FormatInt(r.Number, 10))
			params.Add("hash", r.BlockHash)
			if b == nil {
				b = sc.requestForBlock(ctx, params, r)
			} else {
				sc.storeBlockTransactions(ctx, b)
			}
		}
	}
}

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
