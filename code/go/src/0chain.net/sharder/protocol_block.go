package sharder

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/transaction"
	"0chain.net/util"
	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/blockstore"
	"0chain.net/config"

	"0chain.net/block"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

var blockSaveTimer metrics.Timer

func init() {
	blockSaveTimer = metrics.GetOrRegisterTimer("block_save_time", nil)
}

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
	sc.StoreTransactions(ctx, b)
	err := sc.StoreBlockSummary(ctx, b)
	if err != nil {
		Logger.Error("db error (save block)", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
	}
	self := node.GetSelfNode(ctx)
	if sc.IsBlockSharder(b, self.Node) {
		sc.storeBlock(b)
	}
	if fr != nil {
		sc.storeRound(ctx, fr, b)
	}
	sc.DeleteRoundsBelow(ctx, b.Round)
}

func (sc *Chain) cacheBlockTxns(hash string, txns []*transaction.Transaction) {
	for _, txn := range txns {
		txnSummary := txn.GetSummary()
		txnSummary.BlockHash = hash
		sc.BlockTxnCache.Add(txn.Hash, txnSummary)
	}
}

func (sc *Chain) processBlock(ctx context.Context, b *block.Block) {
	if err := sc.VerifyNotarization(ctx, b.Hash, b.VerificationTickets); err != nil {
		Logger.Error("notarization verification failed", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		return
	}
	if err := b.Validate(ctx); err != nil {
		Logger.Error("block validation", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
		return
	}
	er := sc.GetRound(b.Round)
	if er == nil {
		var r = round.NewRound(b.Round)
		er, _ = sc.AddRound(r).(*round.Round)
		sc.SetRandomSeed(er, b.RoundRandomSeed)
	}
	if sc.AddRoundBlock(er, b) != b {
		return
	}
	sc.SetRoundRank(er, b)
	Logger.Info("received block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)))
	sc.AddNotarizedBlock(ctx, er, b)
}

func (sc *Chain) GetLatestRoundFromSharders(ctx context.Context, currRound int64) *round.Round {
	latestRounds := make([]*round.Round, 0, 1)

	latestRoundHandler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		r, ok := entity.(*round.Round)
		if !ok {
			return nil, nil
		}
		Logger.Info("bc-27 - received round", zap.Int64("round", r.Number))
		latestRounds = append(latestRounds, r)
		return r, nil
	}

	Logger.Info("bc-27 - requesting all the sharders for their latest rounds")
	sc.Sharders.RequestEntityFromAll(ctx, LatestRoundRequestor, nil, latestRoundHandler)
	Logger.Info("bc-27 - Back from requesting all")

	if len(latestRounds) > 0 {
		sort.Slice(latestRounds, func(i int, j int) bool { return latestRounds[i].Number >= latestRounds[j].Number })
		Logger.Info("bc-27 - rounds found the latest round", zap.Int64("round#", latestRounds[0].Number))
		return latestRounds[0]
	}

	Logger.Info("bc-27 - no rounds rreceived from any of the sharders")
	return nil
}

func (sc *Chain) GetMissingRounds(ctx context.Context, targetR int64, dbR int64) {
	Logger.Info("bc-27 get missing rounds", zap.Int64("target round", targetR), zap.Int64("round from db", dbR))
	//get missing rounds starting from the next round of the current round
	dbR++

	rounds := targetR - dbR

	Logger.Info("bc-27 Synching up ", zap.Int64("rounds", rounds))

	for i := int64(0); i < rounds; i++ {
		loopR := dbR + i
		params := map[string]string{"round": strconv.FormatInt(loopR, 10)}
		var r *round.Round
		Logger.Info("bc-27 requesting all sharders for the round", zap.Int64("round", loopR))
		sc.Sharders.RequestEntityFromAll(ctx, RoundRequestor, params, func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
			roundEntity, ok := entity.(*round.Round)
			if !ok {
				Logger.Info("bc-27 Could not get the round info from others", zap.Int64("round#", loopR))
				return nil, nil
			}
			r = roundEntity
			Logger.Info("bc-27 received the round entity from others", zap.Int64("round", roundEntity.Number))
			return r, nil
		})
		if r == nil {
			Logger.Info("bc-27 round is nil")
			return
		}
		if r.BlockHash == "" {
			Logger.Info("bc-27 round block hash is empty", zap.Int64("round", r.Number))
			return
		}
		Logger.Info("bc-27 check to see if block needed to be stored")
		sc.storeMissingRoundBlock(ctx, r)
	}
}

func (sc *Chain) storeMissingRoundBlock(ctx context.Context, r *round.Round) {
	params := map[string]string{"block": r.BlockHash}
	self := node.GetSelfNode(ctx)
	canStore, nodes := sc.IsBlockSharderWithNodes(r.BlockHash, self.Node)
	Logger.Info(fmt.Sprintf("can Store - %b, nodes length - %d", canStore, len(nodes)))
	var requestNode *node.Node
	for _, n := range nodes {
		if n != self.Node {
			requestNode = n
			break
		}
	}
	if requestNode == nil {
		Logger.Info("bc-27 request node is nil")
		return
	}
	var b *block.Block
	Logger.Info("bc-27 requesting sharder for the block", zap.Int64("round", r.Number), zap.Int("sharder-index", requestNode.SetIndex))
	requestNode.RequestEntityFromNode(ctx, BlockRequestor, params, func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		blockEntity, ok := entity.(*block.Block)
		if !ok {
			return nil, nil
		}
		err := blockEntity.Validate(ctx)
		if err == nil {
			b = blockEntity
			return blockEntity, nil
		}
		return nil, err
	})
	if b == nil {
		Logger.Info("bc-27 round block is nil", zap.Int64("round", r.Number))
		return
	}
	sc.StoreTransactions(ctx, b)
	err := sc.StoreBlockSummary(ctx, b)
	if err != nil {
		Logger.Error("db error (save block)", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
	} else {
		Logger.Info("bc-27 missed block stored in db", zap.String("block-hash", b.Hash))
	}
	if canStore {
		sc.storeBlock(b)
		Logger.Info("bc-27 missed block stored in file", zap.String("block-hash", b.Hash))
	}
	sc.storeRound(ctx, r, b)
	Logger.Info("bc-27 missed round stored in db", zap.Int64("round", r.GetRoundNumber()))
}

func (sc *Chain) storeRound(ctx context.Context, r round.RoundI, b *block.Block) {
	r.Finalize(b)
	rImpl, _ := r.(*round.Round)
	err := sc.StoreRound(ctx, rImpl)
	if err != nil {
		Logger.Error("db error (save round)", zap.Int64("round", r.GetRoundNumber()), zap.Error(err))
	}
}

func (sc *Chain) storeBlock(b *block.Block) {
	sc.SharderStats.ShardedBlocksCount++
	ts := time.Now()
	err := blockstore.GetStore().Write(b)
	duration := time.Since(ts)
	blockSaveTimer.UpdateSince(ts)
	p95 := blockSaveTimer.Percentile(.95)
	if blockSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		Logger.Error("block save - slow", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	}
	if err != nil {
		Logger.Error("block save", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	}
}
