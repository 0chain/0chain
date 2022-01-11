package miner

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"0chain.net/smartcontract/storagesc"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"0chain.net/smartcontract/minersc"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	metrics "github.com/rcrowley/go-metrics"
)

//InsufficientTxns - to indicate an error when the transactions are not sufficient to make a block
const InsufficientTxns = "insufficient_txns"

var (
	ErrNotTimeTolerant = common.NewError("not_time_tolerant", "Transaction is behind time tolerance")
	FutureTransaction  = common.NewError("future_transaction", "Transaction has future nonce")
	PastTransaction    = common.NewError("past_transaction", "Transaction has past nonce")
)
var (
	bgTimer     metrics.Timer // block generation timer
	bpTimer     metrics.Timer // block processing timer (includes block verification)
	btvTimer    metrics.Timer // block verification timer
	bsHistogram metrics.Histogram
)

func init() {
	bgTimer = metrics.GetOrRegisterTimer("bg_time", nil)
	bpTimer = metrics.GetOrRegisterTimer("bv_time", nil)
	btvTimer = metrics.GetOrRegisterTimer("btv_time", nil)
	bsHistogram = metrics.GetOrRegisterHistogram("bs_histogram", nil, metrics.NewUniformSample(1024))
}

func (mc *Chain) processTxn(ctx context.Context, txn *transaction.Transaction, b *block.Block, bState util.MerklePatriciaTrieI, clients map[string]*client.Client) error {
	clients[txn.ClientID] = nil
	if ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, txn); ok || err != nil {
		if err != nil {
			return err
		}
		return common.NewError("process fee transaction", "transaction already exists")
	}
	events, err := mc.UpdateState(ctx, b, bState, txn)
	b.Events = append(b.Events, events...)
	if err != nil {
		logging.Logger.Error("processTxn", zap.String("txn", txn.Hash),
			zap.String("txn_object", datastore.ToJSON(txn).String()),
			zap.Error(err))
		return err
	}
	b.Txns = append(b.Txns, txn)
	b.AddTransaction(txn)
	return nil
}

func (mc *Chain) createFeeTxn(b *block.Block, bState util.MerklePatriciaTrieI) *transaction.Transaction {
	feeTxn := transaction.Provider().(*transaction.Transaction)
	feeTxn.ClientID = b.MinerID
	feeTxn.Nonce = mc.getCurrentSelfNonce(b.MinerID, bState)
	feeTxn.ToClientID = minersc.ADDRESS
	feeTxn.CreationDate = b.CreationDate
	feeTxn.TransactionType = transaction.TxnTypeSmartContract
	feeTxn.TransactionData = fmt.Sprintf(`{"name":"payFees","input":{"round":%v}}`, b.Round)
	feeTxn.Fee = 0 //TODO: fee needs to be set to governance minimum fee
	feeTxn.Sign(node.Self.GetSignatureScheme())
	return feeTxn
}

func (mc *Chain) getCurrentSelfNonce(minerId datastore.Key, bState util.MerklePatriciaTrieI) int64 {
	s, err := mc.GetStateById(bState, minerId)
	if err != nil {
		logging.Logger.Error("can't get nonce", zap.Error(err))
		return 1
	}
	node.Self.SetNonce(s.Nonce)
	return node.Self.GetNextNonce()
}

func (mc *Chain) storageScCommitSettingChangesTx(b *block.Block, bState util.MerklePatriciaTrieI) *transaction.Transaction {
	scTxn := transaction.Provider().(*transaction.Transaction)
	scTxn.ClientID = b.MinerID
	scTxn.Nonce = mc.getCurrentSelfNonce(b.MinerID, bState)
	scTxn.ToClientID = storagesc.ADDRESS
	scTxn.CreationDate = b.CreationDate
	scTxn.TransactionType = transaction.TxnTypeSmartContract
	scTxn.TransactionData = fmt.Sprintf(`{"name":"commit_settings_changes","input":{"round":%v}}`, b.Round)
	scTxn.Fee = 0
	scTxn.Sign(node.Self.GetSignatureScheme())
	return scTxn
}

func (mc *Chain) createBlockRewardTxn(b *block.Block, bState util.MerklePatriciaTrieI) *transaction.Transaction {
	brTxn := transaction.Provider().(*transaction.Transaction)
	brTxn.ClientID = b.MinerID
	brTxn.Nonce = mc.getCurrentSelfNonce(b.MinerID, bState)
	brTxn.ToClientID = storagesc.ADDRESS
	brTxn.CreationDate = b.CreationDate
	brTxn.TransactionType = transaction.TxnTypeSmartContract
	brTxn.TransactionData = `{"name":"pay_blobber_block_rewards","input":{}}`
	brTxn.Fee = 0
	brTxn.Sign(node.Self.GetSignatureScheme())
	return brTxn
}

func (mc *Chain) txnToReuse(txn *transaction.Transaction) *transaction.Transaction {
	ctxn := txn.Clone()
	ctxn.OutputHash = ""
	return ctxn
}

func (mc *Chain) validateTransaction(b *block.Block, bState util.MerklePatriciaTrieI, txn *transaction.Transaction) error {
	if !common.WithinTime(int64(b.CreationDate), int64(txn.CreationDate), transaction.TXN_TIME_TOLERANCE) {
		return ErrNotTimeTolerant
	}
	state, err := mc.GetStateById(bState, txn.ClientID)
	if err != nil {
		return err
	}

	if txn.Nonce-state.Nonce > 1 {
		return FutureTransaction
	}

	if txn.Nonce-state.Nonce < 1 {
		return PastTransaction
	}

	return nil
}

// UpdatePendingBlock - updates the block that is generated and pending
// rest of the process.
func (mc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")

	// NOTE: Since we are not explicitly maintaining state in the db, we just
	//       need to adjust the collection score and don't need to write the
	//       entities themselves
	//
	//     transactionMetadataProvider.GetStore().MultiWrite(ctx, transactionMetadataProvider, txns)
	//
	transactionMetadataProvider.GetStore().MultiAddToCollection(ctx, transactionMetadataProvider, txns)
}

func (mc *Chain) verifySmartContracts(ctx context.Context, b *block.Block) error {
	for _, txn := range b.Txns {
		if txn.TransactionType == transaction.TxnTypeSmartContract {
			err := txn.VerifyOutputHash(ctx)
			if err != nil {
				logging.Logger.Error("Smart contract output verification failed", zap.Any("error", err), zap.Any("output", txn.TransactionOutput))
				return common.NewError("txn_output_verification_failed", "Transaction output hash verification failed")
			}
		}
	}
	return nil
}

// VerifyBlockMagicBlockReference verifies LatestFinalizedMagicBlockHash and
// LatestFinalizedMagicBlockRound fields of the block.
func (mc *Chain) VerifyBlockMagicBlockReference(b *block.Block) (err error) {

	var (
		round = b.Round
		lfmbr = mc.GetLatestFinalizedMagicBlockRound(round)

		offsetRound = mbRoundOffset(round)
		nextVCRound = mc.NextViewChange()
	)

	if lfmbr == nil {
		return common.NewError("verify_block_mb_reference", "can't get lfmbr")
	}

	if nextVCRound > 0 && offsetRound >= nextVCRound && lfmbr.StartingRound < nextVCRound {
		// TODO: offsetRound could >= nextVCRound on start when the nextVCRound was not updated correctly.
		logging.Logger.Warn("verify_block_mb_reference - required MB missing or still not finalized")
		return common.NewError("verify_block_mb_reference",
			"required MB missing or still not finalized")
	}

	if b.LatestFinalizedMagicBlockHash != lfmbr.Hash {
		return common.NewError("verify_block_mb_reference",
			"unexpected latest_finalized_mb_hash")
	}

	if b.LatestFinalizedMagicBlockRound != lfmbr.Round {
		return common.NewError("verify_block_mb_reference",
			"unexpected latest_finalized_mb_round")
	}

	return
}

// VerifyBlockMagicBlock verifies MagicBlock of the block. If this miner is
// member of miners of the MagicBlock it can do the verification. Otherwise,
// this method does nothing.
func (mc *Chain) VerifyBlockMagicBlock(ctx context.Context, b *block.Block) (
	err error) {

	var (
		mb          = b.MagicBlock
		selfNodeKey = node.Self.Underlying().GetKey()
		nvc         int64
	)

	if mb == nil || !mb.Miners.HasNode(selfNodeKey) {
		return // ok
	}

	if !b.IsStateComputed() {
		return common.NewErrorf("verify_block_mb",
			"block state is not computed or synced %d", b.Round)
	}

	// the block state required for the NextViewChangeOfBlock to
	// get fresh NVC value
	if b.ClientState == nil {
		if err = mc.InitBlockState(b); err != nil {
			return common.NewErrorf("verify_block_mb",
				"can't initialize block state %d: %v", b.Round, err)
		}
	}

	if nvc, err = mc.NextViewChangeOfBlock(b); err != nil {
		return common.NewErrorf("verify_block_mb",
			"can't get NVC of the block %d: %v", b.Round, err)
	}

	logging.Logger.Debug("verify_block_mb", zap.Int64("round", b.Round),
		zap.Int64("mb_sr", mb.StartingRound), zap.Int64("nvc", nvc))

	if mb.StartingRound != b.Round {
		return common.NewErrorf("verify_block_mb", "got block with invalid "+
			"MB, MB starting round not equal to the block round: R: %d, SR: %d",
			b.Round, mb.StartingRound)
	}

	// check out next view change (miner SC MB rejection)
	if mb.StartingRound != nvc {
		return common.NewErrorf("verify_block_mb",
			"got block with MB rejected by miner SC: %d, %d",
			mb.StartingRound, nvc)
	}

	// check out the MB if this miner is member of it
	var (
		id  = strconv.FormatInt(mb.MagicBlockNumber, 10)
		lmb *block.MagicBlock
	)

	// get stored MB
	if lmb, err = LoadMagicBlock(ctx, id); err != nil {
		return common.NewErrorf("verify_block_mb",
			"can't load related MB from store: %v", err)
	}

	// compare given MB and the stored one (should be equal)
	if !bytes.Equal(mb.Encode(), lmb.Encode()) {
		return common.NewError("verify_block_mb",
			"MB given doesn't match the stored one")
	}

	return
}

// VerifyBlock - given a set of transaction ids within a block, validate the block.
func (mc *Chain) VerifyBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {
	//ctx = common.GetRootContext()

	var start = time.Now()
	if err = b.Validate(ctx); err != nil {
		return
	}

	if err = mc.VerifyBlockMagicBlockReference(b); err != nil {
		return
	}

	var pb *block.Block
	if pb = mc.GetPreviousBlock(ctx, b); pb == nil {
		return nil, block.ErrPreviousBlockUnavailable
	}

	if err = mc.ValidateTransactions(ctx, b); err != nil {
		return
	}

	if err = mc.ComputeState(ctx, b); err != nil {
		if err == context.Canceled {
			logging.Logger.Warn("verify block - compute state canceled",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return
		}

		logging.Logger.Error("verify block - error computing state",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("prev_block", b.PrevHash),
			zap.String("state_hash", util.ToHex(b.ClientStateHash)),
			zap.Error(err))
		return // TODO (sfxdx): to return here or not to return (keep error)?
	}

	if err = mc.verifySmartContracts(ctx, b); err != nil {
		return
	}

	if err = mc.VerifyBlockMagicBlock(ctx, b); err != nil {
		return
	}

	if bvt, err = mc.SignBlock(ctx, b); err != nil {
		return nil, err
	}
	bpTimer.UpdateSince(start)

	logging.Logger.Info("verify block successful", zap.Any("round", b.Round),
		zap.Int("block_size", len(b.Txns)), zap.Any("time", time.Since(start)),
		zap.Any("block", b.Hash), zap.String("prev_block", b.PrevHash),
		zap.String("state_hash", util.ToHex(b.ClientStateHash)),
		zap.Int8("state_status", b.GetStateStatus()),
		zap.Float64("p_chain_weight", pb.ChainWeight))

	return
}

func (mc *Chain) ValidateTransactions(ctx context.Context, b *block.Block) error {
	return mc.validateTxnsWithContext.Run(ctx, func() error {
		var roundMismatch bool
		var cancel bool
		numWorkers := len(b.Txns) / mc.ValidationBatchSize()
		if numWorkers*mc.ValidationBatchSize() < len(b.Txns) {
			numWorkers++
		}
		aggregate := true
		var aggregateSignatureScheme encryption.AggregateSignatureScheme
		if aggregate {
			aggregateSignatureScheme = encryption.GetAggregateSignatureScheme(mc.ClientSignatureScheme(), len(b.Txns), mc.ValidationBatchSize())
		}
		if aggregateSignatureScheme == nil {
			aggregate = false
		}
		validChannel := make(chan bool, numWorkers)
		validate := func(ctx context.Context, txns []*transaction.Transaction, start int) {
			validTxns := make([]*transaction.Transaction, 0, len(txns))
			for _, txn := range txns {
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
					logging.Logger.Error("validate transactions - no output hash", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()))
					validChannel <- false
					return
				}
				err := txn.ValidateWrtTimeForBlock(ctx, b.CreationDate, !aggregate)
				if err != nil {
					cancel = true
					logging.Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()), zap.Error(err))
					validChannel <- false
					return
				}
				ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, txn)
				if ok || err != nil {
					if err != nil {
						logging.Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Error(err))
					}
					cancel = true
					validChannel <- false
					return
				}

				validTxns = append(validTxns, txn)
			}

			txnsNeedVerify := mc.FilterOutValidatedTxns(validTxns)

			if aggregate {
				for i, txn := range txnsNeedVerify {
					sigScheme, err := txn.GetSignatureScheme(ctx)
					if err != nil {
						panic(err)
					}
					if err := aggregateSignatureScheme.Aggregate(sigScheme, start+i, txn.Signature, txn.Hash); err != nil {
						logging.Logger.Error("validate transactions failed",
							zap.Int64("round", b.Round),
							zap.String("block", b.Hash),
							zap.Error(err))
						cancel = true
						validChannel <- false
						return
					}
				}
			}
			validChannel <- true
		}
		ts := time.Now()
		for start := 0; start < len(b.Txns); start += mc.ValidationBatchSize() {
			end := start + mc.ValidationBatchSize()
			if end > len(b.Txns) {
				end = len(b.Txns)
			}
			go validate(ctx, b.Txns[start:end], start)
		}
		count := 0
		for result := range validChannel {
			if roundMismatch {
				logging.Logger.Info("validate transactions (round mismatch)", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("current_round", mc.GetCurrentRound()))
				return ErrRoundMismatch
			}
			if !result {
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
			go mc.SaveClients(b.GetClients())
		}
		return nil
	})
}

/*SignBlock - sign the block and provide the verification ticket */
func (mc *Chain) signBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	var bvt = &block.BlockVerificationTicket{}
	bvt.BlockID = b.Hash
	bvt.Round = b.Round
	var (
		self = node.Self
		err  error
	)
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
	logging.Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round), zap.Int64("current_round", mc.GetCurrentRound()), zap.Float64("weight", b.Weight()), zap.Float64("chain_weight", b.ChainWeight))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			logging.Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	mc.FinalizeBlock(ctx, b)
	go mc.SendFinalizedBlock(context.Background(), b)
	fr := mc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
	}
	mc.DeleteRoundsBelow(b.Round)
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
	mb.Sharders.OneTimeStatusMonitor(ctx, mb.StartingRound)
	lfBlocks := mc.GetLatestFinalizedBlockFromSharder(ctx)
	if len(lfBlocks) > 0 {
		logging.Logger.Info("bc-1 latest finalized Block",
			zap.Int64("lfb_round", lfBlocks[0].Round))
		return lfBlocks[0].Block
	}
	logging.Logger.Info("bc-1 sharders returned no lfb.")
	return nil
}

//NotarizedBlockFetched - handler to process fetched notarized block
func (mc *Chain) NotarizedBlockFetched(ctx context.Context, b *block.Block) {
	// mc.SendNotarization(ctx, b)
}
