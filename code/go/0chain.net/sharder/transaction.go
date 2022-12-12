package sharder

import (
	"context"
	"fmt"
	"math"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/persistencestore"
	"github.com/0chain/common/core/logging"
)

var txnSaveTimer metrics.Timer

func init() {
	txnSaveTimer = metrics.GetOrRegisterTimer("txn_save_time", nil)
}

/*GetTransactionSummary - given a transaction hash, get the transaction summary */
func (sc *Chain) GetTransactionSummary(ctx context.Context, hash string) (*transaction.TransactionSummary, error) {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	txnSummary := txnSummaryEntityMetadata.Instance().(*transaction.TransactionSummary)
	err := txnSummaryEntityMetadata.GetStore().Read(ctx, datastore.ToKey(hash), txnSummary)
	if err != nil {
		return nil, err
	}
	return txnSummary, nil
}

/*GetTransactionConfirmation - given a transaction return the confirmation of it's presence in the block chain */
func (sc *Chain) GetTransactionConfirmation(ctx context.Context, hash string) (*transaction.Confirmation, error) {
	var ts *transaction.TransactionSummary
	t, err := sc.BlockTxnCache.Get(hash)
	if err != nil {
		ts, err = sc.GetTransactionSummary(ctx, hash)
		if err != nil {
			return nil, err
		}
	} else {
		ts = t.(*transaction.TransactionSummary)
	}
	confirmation := datastore.GetEntityMetadata("txn_confirmation").Instance().(*transaction.Confirmation)
	confirmation.Hash = hash
	bhash, err := sc.GetBlockHash(ctx, ts.Round)
	if err != nil {
		return nil, err
	}
	confirmation.BlockHash = bhash

	var b *block.Block
	bc, err := sc.BlockCache.Get(bhash)
	if err != nil {
		bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
		bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
		defer ememorystore.Close(bctx)
		bs, err := sc.GetBlockSummary(bctx, bhash)
		if err != nil {
			return nil, err
		}
		confirmation.Round = bs.Round
		confirmation.MinerID = bs.MinerID
		confirmation.RoundRandomSeed = bs.RoundRandomSeed
		confirmation.StateChangesCount = bs.StateChangesCount
		confirmation.CreationDate = bs.CreationDate
		confirmation.MerkleTreeRoot = bs.MerkleTreeRoot
		confirmation.ReceiptMerkleTreeRoot = bs.ReceiptMerkleTreeRoot
		b, err = sc.GetBlockBySummary(ctx, bs)
		if err != nil {
			return confirmation, nil
		}
	} else {
		b = bc.(*block.Block)
		confirmation.Round = b.Round
		confirmation.MinerID = b.MinerID
		confirmation.RoundRandomSeed = b.GetRoundRandomSeed()
		confirmation.StateChangesCount = b.StateChangesCount
		confirmation.CreationDate = b.CreationDate
	}
	txn := b.GetTransaction(hash)
	confirmation.Status = txn.Status
	confirmation.Transaction = txn
	mt := b.GetMerkleTree()
	confirmation.MerkleTreeRoot = mt.GetRoot()
	confirmation.MerkleTreePath = mt.GetPath(confirmation)
	rmt := b.GetReceiptsMerkleTree()
	confirmation.ReceiptMerkleTreeRoot = rmt.GetRoot()
	confirmation.ReceiptMerkleTreePath = rmt.GetPath(transaction.NewTransactionReceipt(txn))
	confirmation.PreviousBlockHash = b.PrevHash
	return confirmation, nil
}

/*StoreTransactions - persists given list of transactions*/
func (sc *Chain) StoreTransactions(b *block.Block) error {
	var sTxns = make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		txnSummary := txn.GetSummary()
		txnSummary.Round = b.Round
		sTxns[idx] = txnSummary
		if err := sc.BlockTxnCache.Add(txn.Hash, txnSummary); err != nil {
			logging.Logger.Warn("save transaction to cache failed",
				zap.String("txn", txn.Hash),
				zap.Error(err))
		}
	}

	delay := time.Millisecond
	ts := time.Now()
	for tries := 1; tries <= 9; tries++ {
		err := sc.storeTransactions(sTxns)
		if err != nil {
			delay = 2 * delay
			logging.Logger.Error("save transactions error", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Int("retry", tries), zap.Duration("delay", delay), zap.Error(err))
			time.Sleep(delay)
			continue
		}

		logging.Logger.Debug("transactions saved successfully", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Int("block_size", len(b.Txns)))
		break
	}
	duration := time.Since(ts)
	txnSaveTimer.UpdateSince(ts)
	p95 := txnSaveTimer.Percentile(.95)
	if txnSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		logging.Logger.Info("save transactions - slow", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	}
	return nil
}

func (sc *Chain) storeTransactions(sTxns []datastore.Entity) error {
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(common.GetRootContext(), txnSummaryMetadata)
	defer persistencestore.Close(tctx)
	return txnSummaryMetadata.GetStore().MultiWrite(tctx, txnSummaryMetadata, sTxns)
}

var txnSummaryMV = false
var roundToHashMVTable = "round_to_hash"

func txnSummaryCreateMV(targetTable string, srcTable string) string {
	return fmt.Sprintf(
		"CREATE MATERIALIZED VIEW IF NOT EXISTS %v AS SELECT ROUND, HASH FROM %v WHERE ROUND IS NOT NULL PRIMARY KEY (ROUND, HASH)",
		targetTable, srcTable)
}

func getSelectCountTxn(table string, column string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM %v where %v=?", table, column)
}

func (sc *Chain) getTxnCountForRound(ctx context.Context, r int64) (int, error) {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryEntityMetadata)
	defer persistencestore.Close(tctx)
	c := persistencestore.GetCon(tctx)
	if !txnSummaryMV {
		err := c.Query(txnSummaryCreateMV(roundToHashMVTable, txnSummaryEntityMetadata.GetName())).Exec()
		if err == nil {
			txnSummaryMV = true
		} else {
			logging.Logger.Info("create mv", zap.Error(err))
			txnSummaryMV = true
			return 0, err
		}
	}
	// Get the query to get the select count transactions.
	var count int
	if err := c.Query(getSelectCountTxn(roundToHashMVTable, "round"), r).Scan(&count); err != nil {
		return 0, common.NewError("txns_count_failed", fmt.Sprintf("round: %v, err: %v", r, err))
	}
	return count, nil
}
