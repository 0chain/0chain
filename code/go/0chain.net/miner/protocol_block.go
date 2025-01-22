package miner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/config"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
)

// InsufficientTxns - to indicate an error when the transactions are not sufficient to make a block
const InsufficientTxns = "insufficient_txns"

// ErrLFBClientStateNil is returned when client state of latest finalized block is nil
var ErrLFBClientStateNil = errors.New("client state of latest finalized block is empty")

var (
	ErrNotTimeTolerant = common.NewError("not_time_tolerant", "transaction is behind time tolerance")
	FutureTransaction  = common.NewError("future_transaction", "transaction has future nonce")
	PastTransaction    = common.NewError("past_transaction", "transaction has past nonce")
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

func (mc *Chain) processTxn(ctx context.Context,
	txn *transaction.Transaction,
	b *block.Block,
	bState util.MerklePatriciaTrieI,
	clients map[string]*client.Client,
	blockStateCache *statecache.BlockCache,
) error {
	clients[txn.ClientID] = nil
	events, err := mc.UpdateState(ctx, b, bState, txn, blockStateCache)
	if err != nil {
		logging.Logger.Error("processTxn", zap.String("txn", txn.Hash),
			zap.String("txn_object", datastore.ToJSON(txn).String()),
			zap.Error(err))
		return err
	}
	b.Events = append(b.Events, events...)
	b.Txns = append(b.Txns, txn)
	b.AddTransaction(txn)
	return nil
}

func (mc *Chain) createFeeTxn(b *block.Block) (*transaction.Transaction, error) {
	feeTxn := transaction.Provider().(*transaction.Transaction)
	feeTxn.ClientID = node.Self.ID
	feeTxn.PublicKey = node.Self.PublicKey
	feeTxn.ToClientID = minersc.ADDRESS
	feeTxn.CreationDate = b.CreationDate
	feeTxn.TransactionType = transaction.TxnTypeSmartContract
	feeTxn.TransactionData = fmt.Sprintf(`{"name":"payFees","input":{"round":%v}}`, b.Round)
	feeTxn.Fee = 0 //TODO: fee needs to be set to governance minimum fee
	if err := feeTxn.ComputeProperties(); err != nil {
		return nil, err
	}
	return feeTxn, nil
}

func (mc *Chain) storageScCommitSettingChangesTx(b *block.Block) (*transaction.Transaction, error) {
	scTxn := transaction.Provider().(*transaction.Transaction)
	scTxn.ClientID = node.Self.ID
	scTxn.PublicKey = node.Self.PublicKey
	scTxn.ToClientID = storagesc.ADDRESS
	scTxn.CreationDate = b.CreationDate
	scTxn.TransactionType = transaction.TxnTypeSmartContract
	scTxn.TransactionData = fmt.Sprintf(`{"name":"commit_settings_changes","input":{"round":%v}}`, b.Round)
	scTxn.Fee = 0
	if err := scTxn.ComputeProperties(); err != nil {
		return nil, err
	}
	return scTxn, nil
}

func (mc *Chain) createBlockRewardTxn(b *block.Block) (*transaction.Transaction, error) {
	brTxn := transaction.Provider().(*transaction.Transaction)
	brTxn.ClientID = node.Self.ID
	brTxn.PublicKey = node.Self.PublicKey
	brTxn.ToClientID = storagesc.ADDRESS
	brTxn.CreationDate = b.CreationDate
	brTxn.TransactionType = transaction.TxnTypeSmartContract
	brTxn.TransactionData = fmt.Sprintf(`{"name":"blobber_block_rewards","input":{"round":%v}}`, b.Round)
	brTxn.Fee = 0
	if err := brTxn.ComputeProperties(); err != nil {
		return nil, err
	}
	return brTxn, nil
}

func (mc *Chain) validateTransaction(b *block.Block,
	bState util.MerklePatriciaTrieI, txn *transaction.Transaction, waitC chan struct{}) (int64, error) {
	if !common.WithinTime(int64(b.CreationDate), int64(txn.CreationDate), transaction.TXN_TIME_TOLERANCE) {
		return 0, ErrNotTimeTolerant
	}
	state, err := chain.GetStateById(bState, txn.ClientID)
	if err != nil {
		if err == util.ErrValueNotPresent {
			if txn.Nonce > 1 {
				return 0, FutureTransaction
			}
			if txn.Nonce < 1 {
				return 0, PastTransaction
			}
			return 0, nil
		}
		if cstate.ErrInvalidState(err) {
			mc.SyncMissingNodes(b.Round, bState.GetMissingNodeKeys(), waitC)
		}
		return 0, err
	}

	if txn.Nonce-state.Nonce > 1 {
		return state.Nonce, FutureTransaction
	}

	if txn.Nonce-state.Nonce < 1 {
		return state.Nonce, PastTransaction
	}

	return state.Nonce, nil
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
	if err := transactionMetadataProvider.GetStore().MultiAddToCollection(ctx,
		transactionMetadataProvider, txns); err != nil {
		logging.Logger.Error("update pending block failed", zap.Error(err))
	}
}

func (mc *Chain) verifySmartContracts(ctx context.Context, b *block.Block) error {
	for _, txn := range b.Txns {
		if txn.TransactionType == transaction.TxnTypeSmartContract {
			err := txn.VerifyOutputHash(ctx)
			if err != nil {
				logging.Logger.Error("Smart contract output verification failed", zap.Error(err), zap.String("output", txn.TransactionOutput))
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

	return
}

// VerifyBlock - given a set of transaction ids within a block, validate the block.
func (mc *Chain) VerifyBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {
	//ctx = common.GetRootContext()

	var start = time.Now()
	cur := time.Now()
	if err = b.Validate(ctx); err != nil {
		return
	}
	logging.Logger.Debug("Validating finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	cur = time.Now()
	if err = mc.VerifyBlockMagicBlockReference(b); err != nil {
		return
	}
	logging.Logger.Debug("VerifyBlockMagicBlockReference finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	var pb *block.Block
	cur = time.Now()
	if pb = mc.GetPreviousBlock(ctx, b); pb == nil {
		return nil, block.ErrPreviousBlockUnavailable
	}
	logging.Logger.Debug("GetPreviousBlock finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	cur = time.Now()
	if err = mc.ValidateTransactions(ctx, b); err != nil {
		return
	}
	logging.Logger.Debug("ValidateTransactions finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	cost := 0

	lfb := mc.GetLatestFinalizedBlock()
	if lfb.ClientState == nil {
		logging.Logger.Warn("ValidateBlockCost, could not estimate txn cost",
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash),
			zap.Error(ErrLFBClientStateNil))
		return nil, ErrLFBClientStateNil
	}

	var costs []int
	for _, txn := range b.Txns {
		if err := mc.syncAndRetry(ctx, b, "estimate cost", func(ctx context.Context, waitC chan struct{}) error {
			c, err := mc.EstimateTransactionCost(ctx, lfb, txn, chain.WithSync(), chain.WithNotifyC(waitC))
			if err != nil {
				return err
			}

			cost += c
			costs = append(costs, c)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	if cost > mc.ChainConfig.MaxBlockCost() {
		logging.Logger.Error("cost limit exceeded", zap.Int("calculated_cost", cost),
			zap.Int("cost_limit", mc.ChainConfig.MaxBlockCost()), zap.String("block_hash", b.Hash),
			zap.Int("txn_amount", len(b.Txns)), zap.Ints("txn_costs", costs))
		return nil, block.ErrCostTooBig
	}
	logging.Logger.Debug("ValidateBlockCost",
		zap.Int64("round", b.Round),
		zap.String("hash", b.Hash),
		zap.Int("calculated cost", cost))

	cur = time.Now()
	if err := mc.syncAndRetry(ctx, b, "verify block", func(ctx context.Context, waitC chan struct{}) error {
		return mc.ComputeState(ctx, b, waitC)
	}); err != nil {
		return nil, err
	}

	logging.Logger.Debug("verify block - ComputeState finished",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.Duration("spent", time.Since(cur)))

	cur = time.Now()
	if err = mc.verifySmartContracts(ctx, b); err != nil {
		return
	}
	logging.Logger.Debug("verifySmartContracts finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	cur = time.Now()
	// TODO: verify magic block in MPT
	if err = mc.VerifyBlockMagicBlock(ctx, b); err != nil {
		return
	}
	logging.Logger.Debug("VerifyBlockMagicBlock finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	cur = time.Now()
	if bvt, err = mc.SignBlock(ctx, b); err != nil {
		return nil, err
	}
	bpTimer.UpdateSince(start)
	logging.Logger.Debug("SignBlock finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	logging.Logger.Info("verify block successful", zap.Int64("round", b.Round),
		zap.Int("block_size", len(b.Txns)), zap.Duration("time", time.Since(start)),
		zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash),
		zap.String("state_hash", util.ToHex(b.ClientStateHash)),
		zap.Int8("state_status", b.GetStateStatus()))

	return
}

func (mc *Chain) syncAndRetry(ctx context.Context, b *block.Block, desc string, f func(ctx context.Context, ch chan struct{}) error) error {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		retry, err := func() (retry bool, err error) {
			wc := make(chan struct{}, 1)
			err = f(cctx, wc)
			if err == nil {
				return false, nil
			}

			if !cstate.ErrInvalidState(err) {
				logging.Logger.Warn("sync and retry - none invalid state error",
					zap.String("desc", desc),
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash),
					zap.Error(err))
				return false, err
			}

			select {
			case <-cctx.Done():
				return false, cctx.Err()
			case _, ok := <-wc:
				if !ok {
					logging.Logger.Error("sync and retry - sync failed",
						zap.String("desc", desc),
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.Error(err))
					return false, err
				}

				logging.Logger.Debug("sync and retry - retry",
					zap.String("desc", desc),
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash),
					zap.String("prev_block", b.PrevHash),
					zap.String("state_hash", util.ToHex(b.ClientStateHash)))
				return true, nil
			}
		}()

		if err != nil {
			return err
		}

		if !retry {
			return nil
		}
	}
}

func (mc *Chain) ValidateTransactions(ctx context.Context, b *block.Block) error {
	return mc.validateTxnsWithContext.Run(ctx, func() error {
		if len(b.Txns) == 0 {
			logging.Logger.Warn("validating block with empty transactions")
			return nil
		}

		var roundMismatch bool
		var cancel bool
		numWorkers := len(b.Txns) / mc.ValidationBatchSize()
		if numWorkers*mc.ValidationBatchSize() < len(b.Txns) {
			numWorkers++
		}
		var aggregate bool
		aggregateSignatureScheme := encryption.GetAggregateSignatureScheme(mc.ClientSignatureScheme(), len(b.Txns), mc.ValidationBatchSize())
		if aggregateSignatureScheme != nil {
			aggregate = true
		}

		var (
			validChannel   = make(chan bool, numWorkers)
			buildInTxnsMap = make(map[string]struct{})
			bicLock        = &sync.Mutex{}
		)

		hasDuplicateBuildInTxns := func(txn *transaction.Transaction) bool {
			bicLock.Lock()
			defer bicLock.Unlock()
			if mc.isBuildInTxn(txn) {
				if _, ok := buildInTxnsMap[txn.FunctionName]; ok {
					return true
				}
				buildInTxnsMap[txn.FunctionName] = struct{}{}
			}
			return false
		}

		validate := func(ctx context.Context, txns []*transaction.Transaction, start int) {
			result := false
			defer func() {
				select {
				case validChannel <- result:
				case <-ctx.Done():
				}
			}()

			validTxns := make([]*transaction.Transaction, 0, len(txns))
			for _, txn := range txns {
				if cancel {
					return
				}
				if mc.GetCurrentRound() > b.Round {
					cancel = true
					roundMismatch = true
					return
				}
				if txn.OutputHash == "" {
					cancel = true
					logging.Logger.Error("validate transactions - no output hash", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()))
					return
				}
				err := txn.ValidateWrtTimeForBlock(ctx, b.CreationDate, !aggregate)
				if err != nil {
					cancel = true
					logging.Logger.Error("validate transactions", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()), zap.Error(err))
					return
				}

				if hasDuplicateBuildInTxns(txn) {
					logging.Logger.Error("validate transactions - duplicated build-in transaction",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.String("txn", txn.Hash),
						zap.String("function_name", txn.FunctionName))
					cancel = true
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
						return
					}
				}
			}
			result = true
		}

		ts := time.Now()
		for start := 0; start < len(b.Txns); start += mc.ValidationBatchSize() {
			end := start + mc.ValidationBatchSize()
			if end > len(b.Txns) {
				end = len(b.Txns)
			}
			go validate(ctx, b.Txns[start:end], start)
		}

		for count := 0; count < numWorkers; count++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case result := <-validChannel:
				if roundMismatch {
					logging.Logger.Info("validate transactions (round mismatch)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("current_round", mc.GetCurrentRound()))
					return ErrRoundMismatch
				}
				if !result {
					return common.NewError("txn_validation_failed", "Transaction validation failed")
				}
			}
		}

		if aggregate {
			if _, err := aggregateSignatureScheme.Verify(); err != nil {
				return err
			}
		}
		btvTimer.UpdateSince(ts)
		if mc.discoverClients {
			go func() {
				cs, err := b.GetClients()
				if err != nil {
					logging.Logger.Warn("validate transactions, get clients of block failed",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.Error(err))
					return
				}

				if err := mc.SaveClients(cs); err != nil {
					logging.Logger.Warn("validate transactions, save discovered clients failed",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.Error(err))
				}
			}()
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
func (mc *Chain) updateFinalizedBlock(ctx context.Context, b *block.Block) error {
	logging.Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round), zap.Int64("current_round", mc.GetCurrentRound()), zap.Float64("weight", b.Weight()))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			logging.Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	if err := mc.FinalizeBlock(ctx, b); err != nil {
		logging.Logger.Warn("finalize block failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Error(err))
	}

	go mc.SendFinalizedBlock(context.Background(), b)
	mc.DeleteRoundsBelow(b.Round)

	var txns []datastore.Entity
	for _, txn := range b.Txns {
		txns = append(txns, txn)
	}

	cleanPoolCtx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	transaction.RemoveFromPool(cleanPoolCtx, txns)

	selfID := node.Self.Underlying().GetKey()
	cs, err := chain.GetStateById(b.ClientState, selfID)
	if err != nil {
		logging.Logger.Error("[mvc] clean txns, could not find node state", zap.Error(err),
			zap.String("miner", selfID), zap.String("block", b.Hash))
	} else {
		otxns, err := transaction.GetOldNonceTxns(common.GetRootContext(), selfID, cs.Nonce)
		if err != nil {
			logging.Logger.Error("[mvc] clean txns, could not remove old nonce txns", zap.Error(err))
		} else {
			if len(otxns) > 0 {
				if err := mc.deleteTxns(otxns); err != nil {
					logging.Logger.Error("[mvc] clean txns, delete old nonce txns failed", zap.Error(err))
				}
			}
		}
	}

	br := mc.GetRound(b.Round)
	if br == nil {
		return nil
	}

	proposedBlocks := br.GetProposedBlocks()
	for _, sb := range proposedBlocks {
		if sb.MinerID == selfID && sb.Hash != b.Hash {
			_, err := transaction.CollectInvalidFutureTxns(common.GetRootContext(), sb.CreationDate, cs.Nonce, selfID)
			if err != nil {
				logging.Logger.Error("[mvc] clean txns, get invalid future txns failed", zap.Error(err),
					zap.String("miner", selfID), zap.String("block", b.Hash))
			}

			break
		}
	}

	return nil
}

/*FinalizeBlock - finalize the transactions in the block */
func (mc *Chain) FinalizeBlock(ctx context.Context, b *block.Block) error {
	modifiedTxns := make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		modifiedTxns[idx] = txn
	}
	return mc.deleteTxns(modifiedTxns)
}

// NotarizedBlockFetched - handler to process fetched notarized block
func (mc *Chain) NotarizedBlockFetched(ctx context.Context, b *block.Block) {
	// mc.SendNotarization(ctx, b)
}

// txnProcessorHandler process transaction and return bool and error to indicate
// whether processed successfully, or error if any
type txnProcessorHandler func(context.Context,
	util.MerklePatriciaTrieI,
	*transaction.Transaction,
	*TxnIterInfo,
	*statecache.BlockCache,
	chan struct{}) (bool, error)

func txnProcessorHandlerFunc(mc *Chain, b *block.Block) txnProcessorHandler {
	return func(ctx context.Context,
		bState util.MerklePatriciaTrieI,
		txn *transaction.Transaction,
		tii *TxnIterInfo,
		blockStateCache *statecache.BlockCache,
		waitC chan struct{}) (bool, error) {

		if _, ok := tii.txnMap[txn.GetKey()]; ok {
			return false, nil
		}
		var debugTxn = true

		nonce, err := mc.validateTransaction(b, bState, txn, waitC)
		switch err {
		case PastTransaction:
			tii.pastTxns = append(tii.pastTxns, txn)
			if debugTxn {
				logging.Logger.Debug("generate block (debug transaction) error, transaction hash old nonce",
					zap.Any("txn", txn),
					zap.Int32("iterate count", tii.count),
					zap.Any("now", common.Now()),
					zap.Int64("nonce", txn.Nonce))
			}
			return false, nil
		case FutureTransaction:
			list, ok := tii.futureTxns[txn.ClientID]
			if !ok {
				list = &clientNonceTxns{}
			}

			if list.nonce < nonce {
				list.nonce = nonce
			}
			list.txns = append(list.txns, txn)
			sort.SliceStable(list.txns, func(i, j int) bool {
				if list.txns[i].Nonce == list.txns[j].Nonce {
					//if the same nonce order by fee
					return list.txns[i].Fee > list.txns[j].Fee
				}
				return list.txns[i].Nonce < list.txns[j].Nonce
			})
			tii.futureTxns[txn.ClientID] = list
			if debugTxn {
				logging.Logger.Debug("generate block - future transaction",
					zap.String("txn", txn.Hash),
					zap.Int64("round", b.Round),
					zap.Int32("iterate count", tii.count))
			}
			return false, nil
		case ErrNotTimeTolerant:
			tii.invalidTxns = append(tii.invalidTxns, txn)
			if debugTxn {
				logging.Logger.Info("generate block (debug transaction) error - txn creation not within tolerance",
					zap.String("txn", txn.Hash), zap.Int32("idx", tii.idx),
					zap.Any("now", common.Now()))
			}
			return false, nil
		default:
			if err != nil && cstate.ErrInvalidState(err) {
				return false, err // return err to break the txns pool iteration
			}
		}

		if debugTxn {
			logging.Logger.Info("generate block (debug transaction)",
				zap.String("txn", txn.Hash), zap.Int32("idx", tii.idx),
				zap.String("txn_object", datastore.ToJSON(txn).String()))
		}

		events, err := mc.UpdateState(ctx, b, bState, txn, blockStateCache, waitC)
		if err != nil {
			if debugTxn {
				logging.Logger.Error("generate block (debug transaction) update state",
					zap.String("txn", txn.Hash), zap.Int32("idx", tii.idx),
					zap.String("txn_object", datastore.ToJSON(txn).String()),
					zap.Error(err))
			}
			tii.failedStateCount++
			if cstate.ErrInvalidState(err) {
				return false, err // return err to break the txns pool iteration
			}
			return false, nil
		}

		b.Events = append(b.Events, events...)

		// Setting the score lower so the next time blocks are generated
		// these transactions don't show up at the top.
		tii.txnMap[txn.GetKey()] = struct{}{}
		b.Txns = append(b.Txns, txn)
		if debugTxn {
			logging.Logger.Info("generate block (debug transaction) success in processing Txn hash: " + txn.Hash + " blockHash? = " + b.Hash)
		}
		tii.eTxns = append(tii.eTxns, txn)
		b.AddTransaction(txn)
		tii.byteSize += int64(len(txn.TransactionData)) + int64(len(txn.TransactionOutput))
		if txn.PublicKey == "" {
			tii.clients[txn.ClientID] = nil
		}
		tii.idx++
		tii.checkForCurrent(txn)

		return true, nil
	}
}

type clientNonceTxns struct {
	nonce int64
	txns  []*transaction.Transaction
}

type TxnIterInfo struct {
	clients     map[string]*client.Client
	eTxns       []datastore.Entity
	invalidTxns []datastore.Entity
	pastTxns    []datastore.Entity
	futureTxns  map[datastore.Key]*clientNonceTxns
	currentTxns []*transaction.Transaction

	txnMap map[datastore.Key]struct{}

	roundMismatch     bool
	roundTimeout      bool
	count             int32
	roundTimeoutCount int64

	// reInclusionErr is set if the transaction was found in previous block
	reInclusionErr error
	// state compute failed count
	failedStateCount int32
	// transaction index in a block
	idx int32
	// included transaction data size
	byteSize int64
	// accumulated transaction cost
	cost         int
	exemptTxnNum int
}

func (tii *TxnIterInfo) checkForCurrent(txn *transaction.Transaction) {
	if tii.futureTxns == nil {
		return
	}
	//check whether we can execute future transactions
	nonceTxns, ok := tii.futureTxns[txn.ClientID]
	if !ok {
		return
	}
	futures := nonceTxns.txns
	if len(futures) == 0 {
		return
	}
	currentNonce := txn.Nonce
	i := 0
	for ; i < len(futures); i++ {
		if futures[i].Nonce-currentNonce > 1 {
			break
		}
		//we can have several transactions with the same nonce execute first and skip others
		// included n=0 in the list 1, 1, 2. take first 1 and skip the second
		if futures[i].Nonce-currentNonce < 1 {
			tii.pastTxns = append(tii.pastTxns, futures[i])
			continue
		}

		currentNonce = futures[i].Nonce
		tii.currentTxns = append(tii.currentTxns, futures[i])
	}
	//will not sorted by fee here but at least will be sorted by nonce correctly, can improve it
	sort.SliceStable(tii.currentTxns, func(i, j int) bool { return tii.currentTxns[i].Nonce < tii.currentTxns[j].Nonce })

	if i > -1 {
		tii.futureTxns[txn.ClientID].txns = futures[i:]
		tii.futureTxns[txn.ClientID].nonce = currentNonce
		futureNonces := make([]int64, len(futures[i:]))
		for j, t := range futures[i:] {
			futureNonces[j] = t.Nonce
		}
		logging.Logger.Debug("generate block - debug future transactions",
			zap.Int64("current nonce", currentNonce),
			zap.Int64s("future nonces", futureNonces))
	}
}

func newTxnIterInfo(blockSize int32) *TxnIterInfo {
	return &TxnIterInfo{
		clients:    make(map[string]*client.Client),
		eTxns:      make([]datastore.Entity, 0, blockSize),
		futureTxns: make(map[datastore.Key]*clientNonceTxns),
		txnMap:     make(map[datastore.Key]struct{}, blockSize),
	}
}

// txns iterate handler, the function will return bool and error to indicate
// whether the iteration should continue or not, or error if any to stop the iteration
func txnIterHandlerFunc(
	mc *Chain,
	b *block.Block,
	lfb *block.Block,
	bState util.MerklePatriciaTrieI,
	txnProcessor txnProcessorHandler,
	tii *TxnIterInfo,
	blockStateCache *statecache.BlockCache,
	waitC chan struct{}) func(context.Context, datastore.CollectionEntity) (bool, error) {
	return func(ctx context.Context, qe datastore.CollectionEntity) (bool, error) {
		tii.count++
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if mc.GetCurrentRound() > b.Round {
			tii.roundMismatch = true
			return false, nil
		}
		if tii.roundTimeoutCount != mc.GetRoundTimeoutCount() {
			tii.roundTimeout = true
			return false, nil
		}
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			logging.Logger.Error("generate block (invalid entity)", zap.Any("entity", qe))
			// continue iteration to process next transaction
			return true, nil
		}

		if lfb.ClientState == nil {
			logging.Logger.Warn("generate block, chain is not ready yet",
				zap.Int64("round", b.Round),
				zap.String("hash", b.Hash),
				zap.Error(ErrLFBClientStateNil))
			return false, nil
		}

		if txn.Value > config.MaxTokenSupply {
			logging.Logger.Error("generate block, invalid transaction value",
				zap.String("hash", txn.Hash),
				zap.Uint64("value", uint64(txn.Value)))
			tii.invalidTxns = append(tii.invalidTxns, txn)
			return false, errors.New("invalid transaction value, exceeds max token supply")
		}

		cost, fee, err := mc.EstimateTransactionCostFee(ctx, lfb, txn, chain.WithSync(), chain.WithNotifyC(waitC))
		if err != nil {
			logging.Logger.Debug("generate block - bad transaction cost fee",
				zap.Error(err),
				zap.String("txn_hash", txn.Hash))

			// return error to break iteration due to the invalid state error
			if cstate.ErrInvalidState(err) {
				return false, err
			}

			// skipping and continue
			return true, nil
		}

		isExemptTxn := fee == 0
		if isExemptTxn && tii.exemptTxnNum > 0 {
			// only allow 1 exempt transaction per block
			return true, nil
		}

		if mc.IsFeeEnabled() {
			confMinFee := mc.ChainConfig.MinTxnFee()
			if confMinFee > fee {
				fee = confMinFee
			}

			if err := txn.ValidateFee(mc.ChainConfig.TxnExempt(), fee); err != nil {
				logging.Logger.Error("generate block - invalid transaction fee",
					zap.Any("txn", txn),
					zap.Any("estimated fee", fee),
					zap.Error(err))
				tii.invalidTxns = append(tii.invalidTxns, txn)
				return true, nil // skipping and continue
			}
		}

		if tii.cost+cost >= mc.ChainConfig.MaxBlockCost() {
			logging.Logger.Debug("generate block (too big cost, skipping)")
			return true, nil
		}

		success, err := txnProcessor(ctx, bState, txn, tii, blockStateCache, waitC)
		if err != nil {
			logging.Logger.Debug("generate block txn processor failed",
				zap.Error(err),
				zap.Int64("round", b.Round),
				zap.Int32("iterate count", tii.count),
				zap.Int("current block size", len(b.Txns)))
			return false, err
		}

		if !success {
			// skipping and continue to check the next transaction
			logging.Logger.Debug("generate block txn processor failed",
				zap.Int64("round", b.Round),
				zap.Int32("iterate count", tii.count),
				zap.Int("current block size", len(b.Txns)))
			return true, nil
		}

		logging.Logger.Debug("generate block - process txn success",
			zap.Int64("round", b.Round),
			zap.String("txn", txn.Hash))

		tii.cost += cost

		if isExemptTxn { // exempt transaction
			tii.exemptTxnNum++
		}

		if tii.byteSize >= mc.MaxByteSize() {
			logging.Logger.Debug("generate block (too big block size)",
				zap.Bool("byteSize >= mc.NMaxByteSize", tii.byteSize >= mc.ChainConfig.MaxByteSize()),
				zap.Int32("idx", tii.idx),
				zap.Int64("byte size", tii.byteSize),
				zap.Int64("max byte size", mc.ChainConfig.MaxByteSize()),
				zap.Int32("count", tii.count),
				zap.Int("txns", len(b.Txns)))
			return false, nil
		}
		return true, nil
	}
}

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (mc *Chain) generateBlock(ctx context.Context, b *block.Block,
	bsh chain.BlockStateHandler, waitOver bool, waitC chan struct{}) (err error) {

	lfb := mc.GetLatestFinalizedBlock()
	if lfb.ClientState == nil {
		logging.Logger.Error("generate block - chain is not ready yet",
			zap.Error(ErrLFBClientStateNil),
			zap.Int64("round", b.Round))
		return ErrLFBClientStateNil
	}

	b.Txns = make([]*transaction.Transaction, 0, 100)

	var (
		iterInfo        = newTxnIterInfo(int32(cap(b.Txns)))
		txnProcessor    = txnProcessorHandlerFunc(mc, b)
		blockState      = block.CreateStateWithPreviousBlock(b.PrevBlock, mc.GetStateDB(), b.Round)
		blockStateCache = statecache.NewBlockCache(mc.GetStateCache(), statecache.Block{Round: b.Round, Hash: b.Hash, PrevHash: b.PrevHash})
		beginState      = blockState.GetRoot()
		txnIterHandler  = txnIterHandlerFunc(mc, b, lfb, blockState, txnProcessor, iterInfo, blockStateCache, waitC)
	)

	iterInfo.roundTimeoutCount = mc.GetRoundTimeoutCount()

	start := time.Now()
	b.CreationDate = common.Now()
	if b.CreationDate < b.PrevBlock.CreationDate {
		b.CreationDate = b.PrevBlock.CreationDate
	}

	//we use this context for transaction aggregation phase only
	cctx, cancel := context.WithTimeout(ctx, mc.ChainConfig.BlockProposalMaxWaitTime())
	defer cancel()

	buildInTxns, cost, err := mc.buildInTxns(ctx, lfb, b)
	if err != nil {
		return fmt.Errorf("get build-in txns failed: %v", err)
	}

	iterInfo.cost += cost
	futureNonceAllowed := int64(mc.ChainConfig.TxnFutureNonce())

	defer func() {
		var (
			deleteTxns = make([]datastore.Entity, 0, len(iterInfo.futureTxns)+len(iterInfo.pastTxns))
			txnHashes  = make([]string, 0, len(iterInfo.futureTxns)+len(iterInfo.pastTxns))
		)

		// removes future transactions with nonce is too far away
		if len(iterInfo.futureTxns) > 0 {
			for _, nonceTxns := range iterInfo.futureTxns {
				txns := nonceTxns.txns
				if len(txns) > 0 {
					if txns[0].Nonce-nonceTxns.nonce > futureNonceAllowed {
						// remove all following future txns
						for _, ft := range txns {
							deleteTxns = append(deleteTxns, ft)
							txnHashes = append(txnHashes, ft.Hash)
						}
					}
				}
			}

			logging.Logger.Debug("remove future txns",
				zap.Int("count", len(deleteTxns)),
				zap.Strings("txns", txnHashes),
				zap.Int64("future transaction limit", futureNonceAllowed))
		}

		if len(deleteTxns) > 0 {
			if err := mc.deleteTxns(deleteTxns); err != nil {
				logging.Logger.Warn("generate block - remove future txns failed", zap.Error(err))
			}
		}
	}()

	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*transaction.Transaction)
	collectionName := txn.GetCollectionName()
	logging.Logger.Info("generate block starting iteration", zap.Int64("round", b.Round), zap.String("prev_block", b.PrevHash), zap.String("prev_state_hash", util.ToHex(b.PrevBlock.ClientStateHash)))
	err = transactionEntityMetadata.GetStore().IterateCollection(cctx, transactionEntityMetadata, collectionName, txnIterHandler)
	if cstate.ErrInvalidState(err) {
		logging.Logger.Error("generate block - process txn failed",
			zap.Error(err),
			zap.Int64("round", b.Round))
		return err
	}

	if len(iterInfo.invalidTxns) > 0 {
		var keys []string
		for _, txn := range iterInfo.invalidTxns {
			keys = append(keys, txn.GetKey())
		}
		logging.Logger.Info("generate block (found txns very old)", zap.Int64("round", b.Round),
			zap.Int("num_invalid_txns", len(iterInfo.invalidTxns)), zap.Strings("txn_hashes", keys))
		go func() {
			if err := mc.deleteTxns(iterInfo.invalidTxns); err != nil {
				logging.Logger.Warn("generate block - delete invalid txns failed", zap.Error(err))
			}
		}()
	}
	if len(iterInfo.pastTxns) > 0 {
		var keys []string
		for _, txn := range iterInfo.pastTxns {
			keys = append(keys, txn.GetKey())
		}
		logging.Logger.Info("generate block (found pastTxns transactions)", zap.Int64("round", b.Round), zap.Int("txn num", len(keys)))
		go func() {
			if err := mc.deleteTxns(iterInfo.pastTxns); err != nil {
				logging.Logger.Warn("generate block - delete past txns failed", zap.Error(err))
			}
		}()
	}
	if iterInfo.roundMismatch {
		logging.Logger.Debug("generate block (round mismatch)", zap.Int64("round", b.Round), zap.Int64("current_round", mc.GetCurrentRound()))
		return ErrRoundMismatch
	}
	if iterInfo.roundTimeout {
		logging.Logger.Debug("generate block (round timeout)", zap.Int64("round", b.Round), zap.Int64("current_round", mc.GetCurrentRound()))
		return ErrRoundTimeout
	}
	if iterInfo.reInclusionErr != nil {
		logging.Logger.Error("generate block (txn reinclusion check)",
			zap.Int64("round", b.Round), zap.Error(iterInfo.reInclusionErr))
	}

	switch err {
	case context.DeadlineExceeded:
		logging.Logger.Debug("generate block - slow block generation, stopping transaction collection and finishing the block")
	case context.Canceled:
		logging.Logger.Debug("generate block - context cancelled, rejecting current block")
		return err
	default:
		if err != nil {
			return err
		}
	}

	blockSize := iterInfo.idx
	var reusedTxns int32

	rcount := 0
	for i := 0; i < len(iterInfo.currentTxns) && iterInfo.cost < mc.ChainConfig.MaxBlockCost() &&
		iterInfo.byteSize < mc.MaxByteSize() && err != context.DeadlineExceeded; i++ {
		txn := iterInfo.currentTxns[i]
		cost, err := mc.EstimateTransactionCost(ctx, lfb, txn, chain.WithSync())
		if err != nil {
			// Note: optimistic block generation
			// we would just skip the error so that the work on txns collection and state computation above
			// would not be wasted. Therefore, we will pack the block anyway.
			logging.Logger.Debug("Bad transaction cost", zap.Error(err), zap.String("txn_hash", txn.Hash))
			break
		}
		if iterInfo.cost+cost >= mc.ChainConfig.MaxBlockCost() {
			logging.Logger.Debug("generate block (too big cost, skipping)")
			break
		}

		success, err := txnProcessor(ctx, blockState, txn, iterInfo, blockStateCache, waitC)
		if err != nil {
			// optimistic block generation. Same as EstimateTransactionCost above
			logging.Logger.Debug("generate block - process failed and ignored", zap.Error(err))
			break
		}

		if success {
			logging.Logger.Debug("txnProcessor not successful", zap.Any("txn", txn))
			rcount++
			iterInfo.cost += cost
			if iterInfo.byteSize >= mc.MaxByteSize() {
				break
			}
		}
	}
	if rcount > 0 {
		blockSize += int32(rcount)
		logging.Logger.Debug("Processed current transactions", zap.Int("count", rcount))
	}

	if iterInfo.byteSize < mc.MaxByteSize() {
		b.Txns = b.Txns[:blockSize]
		iterInfo.eTxns = iterInfo.eTxns[:blockSize]
	}

l:
	for _, biTxn := range buildInTxns {
		biTxn.Nonce, err = mc.GetCurrentSelfNonce(b.MinerID, blockState)
		if err != nil {
			logging.Logger.Error("generate block - could not get miner nonce",
				zap.Error(err),
				zap.String("miner", b.MinerID))
			return fmt.Errorf("could not get miner nonce of %v: %v", b.MinerID, err)
		}

		_, err := biTxn.Sign(node.Self.GetSignatureScheme())
		if err != nil {
			panic(err)
		}

		err = mc.processTxn(ctx, biTxn, b, blockState, iterInfo.clients, blockStateCache)
		if err != nil {
			logging.Logger.Warn("generate block - process build-in txn failed",
				zap.String("txn", txn.Hash),
				zap.String("SC", txn.TransactionData),
				zap.Int64("round", b.Round),
				zap.Error(err))
			if cstate.ErrInvalidState(err) {
				return err
			}

			switch err {
			case context.Canceled, context.DeadlineExceeded:
				break l
			}
		}
		blockSize++

		if !waitOver && blockSize < mc.MinBlockSize() {
			b.Txns = nil
			var futureTxnsCount int
			for _, ftxns := range iterInfo.futureTxns {
				futureTxnsCount += len(ftxns.txns)
			}
			logging.Logger.Debug("generate block (insufficient txns)",
				zap.Int64("round", b.Round),
				zap.Int32("iteration_count", iterInfo.count),
				zap.Int32("block_size", blockSize),
				zap.Int32("state failure", iterInfo.failedStateCount),
				zap.Int("invalid txns", len(iterInfo.invalidTxns)),
				zap.Int("future txns", futureTxnsCount))

			return common.NewError(InsufficientTxns,
				fmt.Sprintf("not sufficient txns to make a block yet for round %v (iterated %v,block_size %v,state failure %v, invalid %v, future %v, reused %v)",
					b.Round, iterInfo.count, blockSize, iterInfo.failedStateCount, len(iterInfo.invalidTxns), len(iterInfo.futureTxns), 0))
		}
	}

	b.RunningTxnCount = b.PrevBlock.RunningTxnCount + int64(len(b.Txns))
	if iterInfo.byteSize > 10*mc.MaxByteSize() {
		logging.Logger.Info("generate block (too much byte size)",
			zap.Int64("round", b.Round),
			zap.Int64("iteration byte size", iterInfo.byteSize))
	}

	if err = client.GetClients(ctx, iterInfo.clients); err != nil {
		logging.Logger.Error("generate block (get clients error)", zap.Error(err))
		return common.NewError("get_clients_error", err.Error())
	}

	logging.Logger.Debug("generate block (assemble)",
		zap.Int64("round", b.Round),
		zap.Int("txns", len(b.Txns)),
		zap.Duration("time", time.Since(start)))

	bsh.UpdatePendingBlock(ctx, b, iterInfo.eTxns)
	for _, txn := range b.Txns {
		if txn.PublicKey != "" {
			txn.ClientID = datastore.EmptyKey
			continue
		}
		cl := iterInfo.clients[txn.ClientID]
		if cl == nil || cl.PublicKey == "" {
			logging.Logger.Error("generate block (invalid client)", zap.String("client_id", txn.ClientID))
			return common.NewError("invalid_client", "client not available")
		}
		txn.PublicKey = cl.PublicKey
		txn.ClientID = datastore.EmptyKey
	}

	b.SetClientState(blockState)
	b.SetStateChangesCount(blockState)
	bgTimer.UpdateSince(start)
	logging.Logger.Debug("generate block (assemble+update)",
		zap.Int64("round", b.Round),
		zap.Int("txns", len(b.Txns)),
		zap.Duration("time", time.Since(start)))

	if err = mc.hashAndSignGeneratedBlock(ctx, b); err != nil {
		return err
	}

	b.SetBlockState(block.StateGenerated)
	b.SetStateStatus(block.StateSuccessful)
	logging.Logger.Info("generate block (assemble+update+sign)",
		zap.Int64("round", b.Round),
		zap.Int("block_size", len(b.Txns)),
		zap.Int32("reused_txns", 0),
		zap.Int32("reused_txns", reusedTxns),
		zap.Duration("time", time.Since(start)),
		zap.String("block", b.Hash),
		zap.String("prev_block", b.PrevHash),
		zap.String("begin_state_hash", util.ToHex(beginState)),
		zap.String("block_state_hash", util.ToHex(b.ClientStateHash)),
		zap.String("computed_state_hash", util.ToHex(blockState.GetRoot())),
		zap.Int("changes", blockState.GetChangeCount()),
		zap.Int8("state_status", b.GetStateStatus()),
		zap.Int32("iteration_count", iterInfo.count))
	block.StateSanityCheck(ctx, b)
	b.ComputeTxnMap()
	bsHistogram.Update(int64(len(b.Txns)))
	node.Self.Underlying().Info.AvgBlockTxns = int(math.Round(bsHistogram.Mean()))

	blockStateCache.SetBlockHash(b.Hash)
	blockStateCache.Commit()
	return nil
}

func (mc *Chain) buildInTxns(ctx context.Context, lfb, b *block.Block) ([]*transaction.Transaction, int, error) {
	txns := make([]*transaction.Transaction, 0, 4)

	if mc.ChainConfig.IsFeeEnabled() {
		feeTxn, err := mc.createFeeTxn(b)
		if err != nil {
			return nil, 0, err
		}
		txns = append(txns, feeTxn)
	}

	globalNode, err := storagesc.GetConfig(mc.GetQueryStateContext())
	if err != nil {
		return nil, 0, err
	}
	if globalNode.ChallengeEnabled && b.Round%globalNode.ChallengeGenerationGap == 0 {
		gcTxn, err := mc.createGenerateChallengeTxn(b)
		if err != nil {
			return nil, 0, err
		}
		if gcTxn != nil {
			txns = append(txns, gcTxn)
		}
	}
	if mc.ChainConfig.IsBlockRewardsEnabled() &&
		b.Round%globalNode.BlockReward.TriggerPeriod == 0 {
		logging.Logger.Info("start_block_rewards", zap.Int64("round", b.Round))
		brTxn, err := mc.createBlockRewardTxn(b)
		if err != nil {
			return nil, 0, err
		}
		txns = append(txns, brTxn)
	}

	if mc.SmartContractSettingUpdatePeriod() != 0 &&
		b.Round%mc.SmartContractSettingUpdatePeriod() == 0 {
		cscTxn, err := mc.storageScCommitSettingChangesTx(b)
		if err != nil {
			return nil, 0, err
		}
		txns = append(txns, cscTxn)
	}

	var cost int
	for _, txn := range txns {
		c, err := mc.EstimateTransactionCost(ctx, lfb, txn, chain.WithSync())
		if err != nil {
			logging.Logger.Debug("Bad transaction cost", zap.Error(err))
			return nil, 0, err
		}
		cost += c
	}

	return txns, cost, nil
}

func (mc *Chain) createGenChalTxn(b *block.Block) (*transaction.Transaction, error) {
	brTxn := transaction.Provider().(*transaction.Transaction)
	brTxn.ClientID = node.Self.ID
	brTxn.PublicKey = node.Self.PublicKey
	brTxn.ToClientID = storagesc.ADDRESS
	brTxn.CreationDate = b.CreationDate
	brTxn.TransactionType = transaction.TxnTypeSmartContract
	brTxn.TransactionData = fmt.Sprintf(`{"name":"generate_challenge","input":{"round":%d}}`, b.Round)
	brTxn.Fee = 0
	if err := brTxn.ComputeProperties(); err != nil {
		return nil, err
	}
	return brTxn, nil
}
