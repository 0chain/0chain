package sharder

import (
	"context"
	"fmt"
	"math"
	"time"

	"0chain.net/chaincore/block"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	. "0chain.net/core/logging"
	"0chain.net/core/persistencestore"
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
func (sc *Chain) StoreTransactions(ctx context.Context, b *block.Block) error {
	var sTxns = make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		txnSummary := txn.GetSummary()
		txnSummary.Round = b.Round
		sTxns[idx] = txnSummary
		sc.BlockTxnCache.Add(txn.Hash, txnSummary)
	}

	delay := time.Millisecond
	ts := time.Now()
	for tries := 1; tries <= 9; tries++ {
		err := sc.storeTransactions(ctx, sTxns)
		if err != nil {
			delay = 2 * delay
			Logger.Error("save transactions error", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Int("retry", tries), zap.Duration("delay", delay), zap.Error(err))
			time.Sleep(delay)

		} else {
			Logger.Debug("transactions saved successfully", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Int("block_size", len(b.Txns)))
			break
		}
	}
	duration := time.Since(ts)
	txnSaveTimer.UpdateSince(ts)
	p95 := txnSaveTimer.Percentile(.95)
	if txnSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		Logger.Info("save transactions - slow", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	}
	return nil
}

func (sc *Chain) storeTransactions(ctx context.Context, sTxns []datastore.Entity) error {
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryMetadata)
	defer persistencestore.Close(tctx)
	return txnSummaryMetadata.GetStore().MultiWrite(tctx, txnSummaryMetadata, sTxns)
}

var txnTableIndexed = false
var txnSummaryMV = false
var roundToHashMVTable = "round_to_hash"

func txnSummaryCreateMV(targetTable string, srcTable string) string {
	return fmt.Sprintf(
		"CREATE MATERIALIZED VIEW IF NOT EXISTS %v AS SELECT ROUND, HASH FROM %v WHERE ROUND IS NOT NULL PRIMARY KEY (ROUND, HASH)",
		targetTable, srcTable)
}
func getCreateIndex(table string, column string) string {
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS ON %v(%v)", table, column)
}

func getSelectCountTxn(table string, column string) string {
	return fmt.Sprintf("SELECT COUNT(*) FROM %v where %v=?", table, column)
}

func getSelectTxn(table string, column string) string {
	return fmt.Sprintf("SELECT round FROM %v where %v=?", table, column)
}
func (sc *Chain) getTxnCountForRound(ctx context.Context, r int64) (int, error) {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryEntityMetadata)
	defer persistencestore.Close(tctx)
	c := persistencestore.GetCon(tctx)
	if txnSummaryMV == false {
		err := c.Query(txnSummaryCreateMV(roundToHashMVTable, txnSummaryEntityMetadata.GetName())).Exec()
		if err == nil {
			txnSummaryMV = true
		} else {
			Logger.Info("create mv", zap.Error(err))
			txnSummaryMV = true
			return 0, err
		}
	}
	// Get the query to get the select count transactions.
	q := c.Query(getSelectCountTxn(roundToHashMVTable, "round"))
	q.Bind(r)
	iter := q.Iter()
	var count int
	valid := iter.Scan(&count)
	if !valid {
		return 0, common.NewError("txns_count_failed", fmt.Sprintf("txn count retrieval for round = %v failed", r))
	}
	if err := iter.Close(); err != nil {
		return 0, err
	}
	return count, nil
}
func (sc *Chain) getTxnAndCountForRound(ctx context.Context, r int64) (int, error) {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryEntityMetadata)
	defer persistencestore.Close(tctx)
	c := persistencestore.GetCon(tctx)
	if !txnTableIndexed {
		err := c.Query(getCreateIndex(txnSummaryEntityMetadata.GetName(), "round")).Exec()
		if err == nil {
			txnTableIndexed = true
		} else {
			return 0, err
		}
	}
	// Get the
	q := c.Query(getSelectTxn(txnSummaryEntityMetadata.GetName(), "round"))
	q.Bind(r)
	// Now iterate
	iter := q.Iter()
	var round int
	var count int
	for iter.Scan(&round) {
		count++
	}
	if err := iter.Close(); err != nil {
		return 0, err
	}
	return count, nil
}
