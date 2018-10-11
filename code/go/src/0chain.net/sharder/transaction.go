package sharder

import (
	"context"
	"math"
	"time"

	"0chain.net/block"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/datastore"
	"0chain.net/ememorystore"
	. "0chain.net/logging"
	"0chain.net/persistencestore"
	"0chain.net/transaction"
)

var txnSaveTimer metrics.Timer

func init() {
	txnSaveTimer = metrics.GetOrRegisterTimer("txn_save_time", nil)
}

/*GetTransactionSummary - given a transaction hash, get the transaction summary */
func GetTransactionSummary(ctx context.Context, hash string) (*transaction.TransactionSummary, error) {
	txnSummaryEntityMetadata := datastore.GetEntityMetadata("txn_summary")
	txnSummary := txnSummaryEntityMetadata.Instance().(*transaction.TransactionSummary)
	err := txnSummaryEntityMetadata.GetStore().Read(ctx, datastore.ToKey(hash), txnSummary)
	if err != nil {
		return nil, err
	}
	return txnSummary, nil
}

/*GetTransactionConfirmation - given a transaction return the confirmation of it's presence in the block chain */
func GetTransactionConfirmation(ctx context.Context, hash string) (*transaction.Confirmation, error) {
	var ts *transaction.TransactionSummary
	t, err := GetSharderChain().BlockTxnCache.Get(hash)
	if err != nil {
		ts, err = GetTransactionSummary(ctx, hash)
		if err != nil {
			return nil, err
		}
	} else {
		ts = t.(*transaction.TransactionSummary)
	}
	confirmation := datastore.GetEntityMetadata("txn_confirmation").Instance().(*transaction.Confirmation)
	confirmation.Hash = hash
	confirmation.BlockHash = ts.BlockHash

	var b *block.Block
	bc, err := GetSharderChain().BlockCache.Get(ts.BlockHash)
	if err != nil {
		bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
		bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
		defer ememorystore.Close(bctx)
		bs, err := GetBlockSummary(bctx, ts.BlockHash)
		if err != nil {
			return nil, err
		}
		confirmation.Round = bs.Round
		confirmation.RoundRandomSeed = bs.RoundRandomSeed
		confirmation.CreationDate = bs.CreationDate
		confirmation.MerkleTreeRoot = bs.MerkleTreeRoot
		confirmation.ReceiptMerkleTreeRoot = bs.ReceiptMerkleTreeRoot
		b, err = GetSharderChain().GetBlockBySummary(ctx, bs)
		if err != nil {
			return confirmation, nil
		}
	} else {
		b = bc.(*block.Block)
		confirmation.Round = b.Round
		confirmation.RoundRandomSeed = b.RoundRandomSeed
		confirmation.CreationDate = b.CreationDate
	}
	txn := b.GetTransaction(hash)
	confirmation.Transaction = txn
	mt := b.GetMerkleTree()
	confirmation.MerkleTreeRoot = mt.GetRoot()
	confirmation.MerkleTreePath = mt.GetPath(confirmation)
	rmt := b.GetReceiptsMerkleTree()
	confirmation.ReceiptMerkleTreeRoot = rmt.GetRoot()
	confirmation.ReceiptMerkleTreePath = rmt.GetPath(transaction.NewTransactionReceipt(txn))
	return confirmation, nil
}

/*StoreTransactions - persists given list of transactions*/
func (sc *Chain) StoreTransactions(ctx context.Context, b *block.Block) error {
	var sTxns = make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		txnSummary := txn.GetSummary()
		txnSummary.BlockHash = b.Hash
		sTxns[idx] = txnSummary
	}
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := persistencestore.WithEntityConnection(ctx, txnSummaryMetadata)
	defer persistencestore.Close(tctx)
	delay := time.Millisecond
	ts := time.Now()
	for tries := 1; tries <= 9; tries++ {
		err := txnSummaryMetadata.GetStore().MultiWrite(tctx, txnSummaryMetadata, sTxns)
		if err != nil {
			delay = 2 * delay
			Logger.Error("save transactions error", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Int("retry", tries), zap.Duration("delay", delay), zap.Error(err))
			time.Sleep(delay)
		} else {
			Logger.Info("transactions saved successfully", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Int("block_size", len(b.Txns)))
			break
		}
	}
	duration := time.Since(ts)
	p95 := txnSaveTimer.Percentile(.95)
	if txnSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		Logger.Error("save transactions - slow", zap.Any("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	}
	txnSaveTimer.UpdateSince(ts)
	return nil
}
