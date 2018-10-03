package sharder

import (
	"context"

	"0chain.net/chain"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/transaction"
	"0chain.net/util"

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

func (sc *Chain) processBlock(ctx context.Context, b *block.Block) {
	eb, err := sc.GetBlock(ctx, b.Hash)
	if eb != nil {
		if err == nil {
			Logger.Debug("block already received", zap.Any("round", b.Round), zap.Any("block", b.Hash))
			return
		}
		Logger.Error("get block", zap.Any("block", b.Hash), zap.Error(err))
	}
	if err := sc.VerifyNotarization(ctx, b.Hash, b.VerificationTickets); err != nil {
		Logger.Error("notarization verification failed", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return
	}
	if err := b.Validate(ctx); err != nil {
		Logger.Error("block validation", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
		return
	}
	if sc.AddBlock(b) != b {
		return
	}
	er := sc.GetRound(b.Round)
	if er != nil {
		if sc.BlocksToSharder == chain.FINALIZED {
			nb := er.GetNotarizedBlocks()
			if len(nb) > 0 {
				Logger.Error("*** different blocks for the same round ***", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("existing_block", nb[0].Hash))
			}
		}
	} else {
		r := datastore.GetEntityMetadata("round").Instance().(*round.Round)
		r.Number = b.Round
		r.RandomSeed = b.RoundRandomSeed
		r.ComputeMinerRanks(sc.Miners.Size())
		er, _ = sc.AddRound(r).(*round.Round)
	}
	bNode := node.GetNode(b.MinerID)
	b.RoundRank = er.GetMinerRank(bNode)
	Logger.Info("received block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)))
	er.AddNotarizedBlock(b)
	pr := sc.GetRound(er.GetRoundNumber() - 1)
	if pr != nil {
		go sc.FinalizeRound(ctx, pr, sc)
	}
	err = sc.ComputeState(ctx, b)
	if err != nil {
		if config.DevConfiguration.State {
			Logger.Error("error computing the state (TODO sync state)", zap.Error(err))
		}
	}
}
