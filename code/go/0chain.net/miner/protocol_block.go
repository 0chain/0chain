package miner

import (
	"context"
	"fmt"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/smartcontract/minersc"

	"go.uber.org/zap"
)

//InsufficientTxns - to indicate an error when the transactions are not sufficient to make a block
const InsufficientTxns = "insufficient_txns"

var bgTimer metrics.Timer  // block generation timer
var bpTimer metrics.Timer  // block processing timer (includes block verification)
var btvTimer metrics.Timer // block verification timer
var bsHistogram metrics.Histogram

func init() {
	bgTimer = metrics.GetOrRegisterTimer("bg_time", nil)
	bpTimer = metrics.GetOrRegisterTimer("bv_time", nil)
	btvTimer = metrics.GetOrRegisterTimer("btv_time", nil)
	bsHistogram = metrics.GetOrRegisterHistogram("bs_histogram", nil, metrics.NewUniformSample(1024))
}

func (mc *Chain) processFeeTxn(ctx context.Context, b *block.Block, clients map[string]*client.Client) error {
	feeTxn := mc.createFeeTxn(b)
	clients[feeTxn.ClientID] = nil
	if ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, feeTxn); ok || err != nil {
		if err != nil {
			return err
		}
		return common.NewError("process fee transaction", "transaction already exists")
	}
	if err := mc.UpdateState(b, feeTxn); err != nil {
		Logger.Error("processFeeTxn", zap.String("txn", feeTxn.Hash), zap.String("txn_object", datastore.ToJSON(feeTxn).String()), zap.Error(err))
		return err
	}
	b.Txns = append(b.Txns, feeTxn)
	b.AddTransaction(feeTxn)
	return nil
}

func (mc *Chain) createFeeTxn(b *block.Block) *transaction.Transaction {
	feeTxn := transaction.Provider().(*transaction.Transaction)
	feeTxn.ClientID = b.MinerID
	feeTxn.ToClientID = minersc.ADDRESS
	feeTxn.CreationDate = b.CreationDate
	feeTxn.TransactionType = transaction.TxnTypeSmartContract
	feeTxn.TransactionData = fmt.Sprintf(`{"name":"payFees","input":{"round":%v}}`, b.Round)
	feeTxn.Fee = 0 //TODO: fee needs to be set to governance minimum fee
	feeTxn.Sign(node.Self.GetSignatureScheme())
	return feeTxn
}

func (mc *Chain) txnToReuse(txn *transaction.Transaction) *transaction.Transaction {
	ctxn := *txn
	ctxn.OutputHash = ""
	return &ctxn
}

func (mc *Chain) validateTransaction(b *block.Block, txn *transaction.Transaction) bool {
	return common.WithinTime(int64(b.CreationDate), int64(txn.CreationDate), transaction.TXN_TIME_TOLERANCE)
}

/*UpdatePendingBlock - updates the block that is generated and pending rest of the process */
func (mc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")

	//NOTE: Since we are not explicitly maintaining state in the db, we just need to adjust the collection score and don't need to write the entities themselves
	//transactionMetadataProvider.GetStore().MultiWrite(ctx, transactionMetadataProvider, txns)
	transactionMetadataProvider.GetStore().MultiAddToCollection(ctx, transactionMetadataProvider, txns)
}

func (mc *Chain) verifySmartContracts(ctx context.Context, b *block.Block) error {
	for _, txn := range b.Txns {
		if txn.TransactionType == transaction.TxnTypeSmartContract {
			err := txn.VerifyOutputHash(ctx)
			if err != nil {
				Logger.Error("Smart contract output verification failed", zap.Any("error", err), zap.Any("output", txn.TransactionOutput))
				return common.NewError("txn_output_verification_failed", "Transaction output hash verification failed")
			}
		}
	}
	return nil
}

/*VerifyBlock - given a set of transaction ids within a block, validate the block */
func (mc *Chain) VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	start := time.Now()
	err := b.Validate(ctx)
	if err != nil {
		return nil, err
	}
	if err = mc.VerifyBlockLFMB(b); err != nil {
		return nil, err
	}
	pb := mc.GetPreviousBlock(ctx, b)
	if pb == nil {
		return nil, chain.ErrPreviousBlockUnavailable
	}
	err = mc.ValidateTransactions(ctx, b)
	if err != nil {
		return nil, err
	}
	serr := mc.ComputeState(ctx, b)
	if serr != nil {
		Logger.Error("verify block - error computing state",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("prev_block", b.PrevHash),
			zap.String("state_hash", util.ToHex(b.ClientStateHash)),
			zap.Error(serr))
		return nil, serr
	}
	err = mc.verifySmartContracts(ctx, b)
	if err != nil {
		return nil, err
	}
	bvt, err := mc.SignBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	bpTimer.UpdateSince(start)
	Logger.Info("verify block successful", zap.Any("round", b.Round), zap.Int("block_size", len(b.Txns)), zap.Any("time", time.Since(start)),
		zap.Any("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Int8("state_status", b.GetStateStatus()),
		zap.Float64("p_chain_weight", pb.ChainWeight), zap.Error(serr))
	return bvt, nil
}

/*ValidateTransactions - validate the transactions in the block */
func (mc *Chain) ValidateTransactions(ctx context.Context, b *block.Block) error {
	var roundMismatch bool
	var cancel bool
	numWorkers := len(b.Txns) / mc.ValidationBatchSize
	if numWorkers*mc.ValidationBatchSize < len(b.Txns) {
		numWorkers++
	}
	aggregate := true
	var aggregateSignatureScheme encryption.AggregateSignatureScheme
	if aggregate {
		aggregateSignatureScheme = encryption.GetAggregateSignatureScheme(mc.ClientSignatureScheme, len(b.Txns), mc.ValidationBatchSize)
	}
	if aggregateSignatureScheme == nil {
		aggregate = false
	}
	validChannel := make(chan bool, numWorkers)
	validate := func(ctx context.Context, txns []*transaction.Transaction, start int) {
		for idx, txn := range txns {
			if cancel {
				validChannel <- false
				return
			}
			if mc.GetCurrentRound() > b.Round {
				cancel = true
				roundMismatch = true
				validChannel <- false
				return
			}
			if txn.OutputHash == "" {
				cancel = true
				Logger.Error("validate transactions - no output hash", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()))
				validChannel <- false
				return
			}
			err := txn.ValidateWrtTimeForBlock(ctx, b.CreationDate, !aggregate)
			if err != nil {
				cancel = true
				Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()), zap.Error(err))
				validChannel <- false
				return
			}
			if aggregate {
				sigScheme, err := txn.GetSignatureScheme(ctx)
				if err != nil {
					panic(err)
				}
				aggregateSignatureScheme.Aggregate(sigScheme, start+idx, txn.Signature, txn.Hash)
			}
			ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, txn)
			if ok || err != nil {
				if err != nil {
					Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Error(err))
				}
				cancel = true
				validChannel <- false
				return
			}
		}
		validChannel <- true
	}
	ts := time.Now()
	for start := 0; start < len(b.Txns); start += mc.ValidationBatchSize {
		end := start + mc.ValidationBatchSize
		if end > len(b.Txns) {
			end = len(b.Txns)
		}
		go validate(ctx, b.Txns[start:end], start)
	}
	count := 0
	for result := range validChannel {
		if roundMismatch {
			Logger.Info("validate transactions (round mismatch)", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("current_round", mc.GetCurrentRound()))
			return common.NewError(RoundMismatch, "current round different from generation round")
		}
		if !result {
			//Logger.Debug("validate transactions failure", zap.String("block", datastore.ToJSON(b).String()))
			return common.NewError("txn_validation_failed", "Transaction validation failed")
		}
		count++
		if count == numWorkers {
			break
		}
	}
	if aggregate {
		if _, err := aggregateSignatureScheme.Verify(); err != nil {
			return err
		}
	}
	btvTimer.UpdateSince(ts)
	if mc.discoverClients {
		go mc.SaveClients(ctx, b.GetClients())
	}
	return nil
}

/*SignBlock - sign the block and provide the verification ticket */
func (mc *Chain) signBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	var bvt = &block.BlockVerificationTicket{}
	bvt.BlockID = b.Hash
	bvt.Round = b.Round
	self := node.GetSelfNode(ctx)
	var err error
	bvt.VerifierID = self.Underlying().GetKey()
	bvt.Signature, err = self.Sign(b.Hash)
	b.SetVerificationStatus(block.VerificationSuccessful)
	if err != nil {
		return nil, err
	}
	return bvt, nil
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round), zap.Int64("current_round", mc.GetCurrentRound()), zap.Float64("weight", b.Weight()), zap.Float64("chain_weight", b.ChainWeight))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	mc.FinalizeBlock(ctx, b)
	go mc.SendFinalizedBlock(ctx, b)
	fr := mc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
	}
	mc.DeleteRoundsBelow(ctx, b.Round)
}

/*FinalizeBlock - finalize the transactions in the block */
func (mc *Chain) FinalizeBlock(ctx context.Context, b *block.Block) error {
	modifiedTxns := make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		modifiedTxns[idx] = txn
	}
	return mc.deleteTxns(modifiedTxns)
}

func getLatestBlockFromSharders(ctx context.Context) *block.Block {
	mc := GetMinerChain()
	mb := mc.GetCurrentMagicBlock()
	mb.Sharders.OneTimeStatusMonitor(ctx)
	lfBlocks := mc.GetLatestFinalizedBlockFromSharder(ctx)
	if len(lfBlocks) > 0 {
		Logger.Info("bc-1 latest finalized Block",
			zap.Int64("lfb_round", lfBlocks[0].Round))
		return lfBlocks[0].Block
	}
	Logger.Info("bc-1 sharders returned no lfb.")
	return nil
}

//NotarizedBlockFetched - handler to process fetched notarized block
func (mc *Chain) NotarizedBlockFetched(ctx context.Context, b *block.Block) {
	mc.SendNotarization(ctx, b)
}
