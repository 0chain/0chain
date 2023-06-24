package sharder

import (
	"context"
	"errors"
	"math"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
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
	key := transaction.BuildSummaryTransactionKey(hash)
	err := txnSummaryEntityMetadata.GetStore().Read(ctx, datastore.ToKey(key), txnSummary)
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
	confirmation := new(transaction.Confirmation)
	confirmation.Version = "1.0"
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
		defer ememorystore.CloseEntityConnection(bctx, bSummaryEntityMetadata)
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
	var success bool
	for tries := 1; tries <= 9; tries++ {
		err := sc.storeTransactions(sTxns, b.Round)
		if err != nil {
			var (
				txnNames      []string
				txnOutputSize []int
			)

			for _, txn := range b.Txns {
				txnNames = append(txnNames, txn.TransactionData)
				txnOutputSize = append(txnOutputSize, len(txn.TransactionOutput))
			}

			delay = 2 * delay
			logging.Logger.Error("save transactions error",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.Int("block_size", len(b.Txns)),
				zap.Strings("txns", txnNames),
				zap.Ints("txn_output_size", txnOutputSize),
				zap.Int("retry", tries),
				zap.Duration("delay", delay), zap.Error(err))
			time.Sleep(delay)
			continue
		}

		success = true
		logging.Logger.Debug("transactions saved successfully", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)))
		break
	}

	if !success {
		return errors.New("failed to save transactions")
	}

	duration := time.Since(ts)
	txnSaveTimer.UpdateSince(ts)
	p95 := txnSaveTimer.Percentile(.95)
	if txnSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		logging.Logger.Info("save transactions - slow", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	}
	return nil
}

/* storeTransactions - persists given list of transactions and increment TxnsCount of their round */
func (sc *Chain) storeTransactions(sTxns []datastore.Entity, roundNumber int64) error {
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := ememorystore.WithEntityConnection(common.GetRootContext(), txnSummaryMetadata)
	defer ememorystore.Close(tctx)

	rtcKey := transaction.BuildSummaryRoundKey(roundNumber)
	rtcDelta := transaction.RoundTxnsCount{
		HashIDField: datastore.HashIDField{
			Hash: rtcKey,
		},
		TxnsCount: int64(len(sTxns)),
	}
	err := txnSummaryMetadata.GetStore().Merge(tctx, &rtcDelta)
	if err != nil {
		return err
	}

	// Write the transactions, keyspace the hash
	for _, txn := range sTxns {
		txKey := transaction.BuildSummaryTransactionKey(txn.GetKey())
		txn.SetKey(txKey)
	}
	err = txnSummaryMetadata.GetStore().MultiWrite(tctx, txnSummaryMetadata, sTxns)
	if err != nil {
		return err
	}

	tCon := ememorystore.GetEntityCon(tctx, txnSummaryMetadata)
	err = tCon.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (sc *Chain) getTxnCountForRound(ctx context.Context, r int64) (int, error) {
	txnSummaryMetadata := datastore.GetEntityMetadata("txn_summary")
	tctx := ememorystore.WithEntityConnection(common.GetRootContext(), txnSummaryMetadata)
	defer ememorystore.Close(tctx)

	// Read the count of txns_per_round for this round
	rtcKey := transaction.BuildSummaryRoundKey(r)
	var rtc transaction.RoundTxnsCount
	err := txnSummaryMetadata.GetStore().Read(tctx, rtcKey, &rtc)
	if err != nil {
		return 0, err
	}
	return int(rtc.TxnsCount), nil
}
