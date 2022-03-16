package miner

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
)

//InsufficientTxns - to indicate an error when the transactions are not sufficient to make a block
const InsufficientTxns = "insufficient_txns"

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

func (mc *Chain) storageScCommitSettingChangesTx(b *block.Block) *transaction.Transaction {
	scTxn := transaction.Provider().(*transaction.Transaction)
	scTxn.ClientID = b.MinerID
	scTxn.ToClientID = storagesc.ADDRESS
	scTxn.CreationDate = b.CreationDate
	scTxn.TransactionType = transaction.TxnTypeSmartContract
	scTxn.TransactionData = fmt.Sprintf(`{"name":"commit_settings_changes","input":{"round":%v}}`, b.Round)
	scTxn.Fee = 0
	scTxn.Sign(node.Self.GetSignatureScheme())
	return scTxn
}

func (mc *Chain) createBlockRewardTxn(b *block.Block) *transaction.Transaction {
	brTxn := transaction.Provider().(*transaction.Transaction)
	brTxn.ClientID = b.MinerID
	brTxn.ToClientID = storagesc.ADDRESS
	brTxn.CreationDate = b.CreationDate
	brTxn.TransactionType = transaction.TxnTypeSmartContract
	brTxn.TransactionData = `{"name":"blobber_block_rewards","input":{}}`
	brTxn.Fee = 0
	brTxn.Sign(node.Self.GetSignatureScheme())
	return brTxn
}

func (mc *Chain) txnToReuse(txn *transaction.Transaction) *transaction.Transaction {
	ctxn := txn.Clone()
	ctxn.OutputHash = ""
	return ctxn
}

func (mc *Chain) validateTransaction(b *block.Block, txn *transaction.Transaction) bool {
	return common.WithinTime(int64(b.CreationDate), int64(txn.CreationDate), transaction.TXN_TIME_TOLERANCE)
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
	cur := time.Now()
	logging.Logger.Debug("Validating", zap.String("block", b.Hash))
	if err = b.Validate(ctx); err != nil {
		return
	}
	logging.Logger.Debug("Validating finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	cur = time.Now()
	logging.Logger.Debug("VerifyBlockMagicBlockReference", zap.String("block", b.Hash))
	if err = mc.VerifyBlockMagicBlockReference(b); err != nil {
		return
	}
	logging.Logger.Debug("VerifyBlockMagicBlockReference finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	var pb *block.Block
	cur = time.Now()
	logging.Logger.Debug("GetPreviousBlock", zap.String("block", b.Hash))
	if pb = mc.GetPreviousBlock(ctx, b); pb == nil {
		return nil, block.ErrPreviousBlockUnavailable
	}
	logging.Logger.Debug("GetPreviousBlock finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	logging.Logger.Debug("ValidateTransactions", zap.String("block", b.Hash))
	cur = time.Now()
	if err = mc.ValidateTransactions(ctx, b); err != nil {
		return
	}
	logging.Logger.Debug("ValidateTransactions finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	logging.Logger.Debug("ValidateBlockCost", zap.String("block", b.Hash))
	cost := 0
	for _, txn := range b.Txns {
		c, err := mc.EstimateTransactionCost(ctx, b, mc.GetLatestFinalizedBlock().ClientState, txn)
		if err != nil {
			return nil, err
		}
		cost += c
		if cost > mc.Config.MaxBlockCost() {
			return nil, block.ErrCostTooBig
		}
	}
	logging.Logger.Debug("ValidateBlockCost", zap.Int("calculated cost", cost))

	logging.Logger.Debug("ComputeState", zap.String("block", b.Hash))
	cur = time.Now()
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
	logging.Logger.Debug("ComputeState finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	logging.Logger.Debug("verifySmartContracts", zap.String("block", b.Hash))
	cur = time.Now()
	if err = mc.verifySmartContracts(ctx, b); err != nil {
		return
	}
	logging.Logger.Debug("verifySmartContracts finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	logging.Logger.Debug("VerifyBlockMagicBlock", zap.String("block", b.Hash))
	cur = time.Now()
	if err = mc.VerifyBlockMagicBlock(ctx, b); err != nil {
		return
	}
	logging.Logger.Debug("VerifyBlockMagicBlock finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	logging.Logger.Debug("SignBlock", zap.String("block", b.Hash))
	cur = time.Now()
	if bvt, err = mc.SignBlock(ctx, b); err != nil {
		return nil, err
	}
	bpTimer.UpdateSince(start)
	logging.Logger.Debug("SignBlock finished", zap.String("block", b.Hash), zap.Duration("spent", time.Since(cur)))

	logging.Logger.Info("verify block successful", zap.Any("round", b.Round),
		zap.Int("block_size", len(b.Txns)), zap.Any("time", time.Since(start)),
		zap.Any("block", b.Hash), zap.String("prev_block", b.PrevHash),
		zap.String("state_hash", util.ToHex(b.ClientStateHash)),
		zap.Int8("state_status", b.GetStateStatus()))

	return
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
					logging.Logger.Error("validate transactions - no output hash", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()))
					return
				}
				err := txn.ValidateWrtTimeForBlock(ctx, b.CreationDate, !aggregate)
				if err != nil {
					cancel = true
					logging.Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()), zap.Error(err))
					return
				}
				ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, txn)
				if ok || err != nil {
					if err != nil {
						logging.Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Error(err))
					}
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
					logging.Logger.Info("validate transactions (round mismatch)", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("current_round", mc.GetCurrentRound()))
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
func (mc *Chain) updateFinalizedBlock(ctx context.Context, b *block.Block) {
	logging.Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round), zap.Int64("current_round", mc.GetCurrentRound()), zap.Float64("weight", b.Weight()))
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

type txnProcessorHandler func(context.Context, util.MerklePatriciaTrieI, *transaction.Transaction, *TxnIterInfo) bool

func txnProcessorHandlerFunc(mc *Chain, b *block.Block) txnProcessorHandler {
	return func(ctx context.Context, bState util.MerklePatriciaTrieI, txn *transaction.Transaction, tii *TxnIterInfo) bool {
		if _, ok := tii.txnMap[txn.GetKey()]; ok {
			return false
		}
		var debugTxn = txn.DebugTxn()
		if !mc.validateTransaction(b, txn) {
			tii.invalidTxns = append(tii.invalidTxns, txn)
			if debugTxn {
				logging.Logger.Info("generate block (debug transaction) error - "+
					"txn creation not within tolerance",
					zap.String("txn", txn.Hash), zap.Int32("idx", tii.idx),
					zap.Any("now", common.Now()))
			}
			return false
		}
		if debugTxn {
			logging.Logger.Info("generate block (debug transaction)",
				zap.String("txn", txn.Hash), zap.Int32("idx", tii.idx),
				zap.String("txn_object", datastore.ToJSON(txn).String()))
		}
		if ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, txn); ok || err != nil {
			if err != nil {
				tii.reInclusionErr = err
			}
			return false
		}
		events, err := mc.UpdateState(ctx, b, bState, txn)
		b.Events = append(b.Events, events...)
		if err != nil {
			if debugTxn {
				logging.Logger.Error("generate block (debug transaction) update state",
					zap.String("txn", txn.Hash), zap.Int32("idx", tii.idx),
					zap.String("txn_object", datastore.ToJSON(txn).String()),
					zap.Error(err))
			}
			tii.failedStateCount++
			return false
		}

		// Setting the score lower so the next time blocks are generated
		// these transactions don't show up at the top.
		txn.SetCollectionScore(txn.GetCollectionScore() - 10*60)
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
		return true
	}
}

type TxnIterInfo struct {
	clients     map[string]*client.Client
	eTxns       []datastore.Entity
	invalidTxns []datastore.Entity
	txnMap      map[datastore.Key]struct{}

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
	//accumulated transaction cost
	cost int
}

func newTxnIterInfo(blockSize int32) *TxnIterInfo {
	return &TxnIterInfo{
		clients: make(map[string]*client.Client),
		eTxns:   make([]datastore.Entity, 0, blockSize),
		txnMap:  make(map[datastore.Key]struct{}, blockSize),
	}
}

func txnIterHandlerFunc(mc *Chain,
	b *block.Block,
	bState util.MerklePatriciaTrieI,
	txnProcessor txnProcessorHandler,
	tii *TxnIterInfo) func(context.Context, datastore.CollectionEntity) bool {
	return func(ctx context.Context, qe datastore.CollectionEntity) bool {
		tii.count++
		if mc.GetCurrentRound() > b.Round {
			tii.roundMismatch = true
			return false
		}
		if tii.roundTimeoutCount != mc.GetRoundTimeoutCount() {
			tii.roundTimeout = true
			return false
		}
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			logging.Logger.Error("generate block (invalid entity)", zap.Any("entity", qe))
			return true
		}
		cost, err := mc.EstimateTransactionCost(ctx, mc.GetLatestFinalizedBlock(), mc.GetLatestFinalizedBlock().ClientState, txn)
		if err != nil {
			logging.Logger.Debug("Bad transaction cost", zap.Error(err))
			return true
		}
		if tii.cost+cost >= mc.Config.MaxBlockCost() {
			logging.Logger.Debug("generate block (too big cost, skipping)")
			return true
		}

		if txnProcessor(ctx, bState, txn, tii) {
			if tii.idx >= mc.BlockSize() || tii.byteSize >= mc.MaxByteSize() {
				logging.Logger.Debug("generate block (too big block size)",
					zap.Bool("idx >= block size", tii.idx >= mc.BlockSize()),
					zap.Bool("byteSize >= mc.NMaxByteSize", tii.byteSize >= mc.MaxByteSize()),
					zap.Int32("idx", tii.idx),
					zap.Int32("block size", mc.BlockSize()),
					zap.Int64("byte size", tii.byteSize),
					zap.Int64("max byte size", mc.MaxByteSize()),
					zap.Int32("count", tii.count),
					zap.Int("txns", len(b.Txns)))
				return false
			}
		}
		return true
	}
}

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (mc *Chain) generateBlock(ctx context.Context, b *block.Block,
	bsh chain.BlockStateHandler, waitOver bool) error {

	b.Txns = make([]*transaction.Transaction, 0, mc.BlockSize())

	var (
		iterInfo       = newTxnIterInfo(mc.BlockSize())
		txnProcessor   = txnProcessorHandlerFunc(mc, b)
		blockState     = block.CreateStateWithPreviousBlock(b.PrevBlock, mc.GetStateDB(), b.Round)
		beginState     = blockState.GetRoot()
		txnIterHandler = txnIterHandlerFunc(mc, b, blockState, txnProcessor, iterInfo)
	)

	iterInfo.roundTimeoutCount = mc.GetRoundTimeoutCount()

	start := time.Now()
	b.CreationDate = common.Now()
	if b.CreationDate < b.PrevBlock.CreationDate {
		b.CreationDate = b.PrevBlock.CreationDate
	}

	//we use this context for transaction aggregation phase only
	cctx, _ := context.WithTimeout(ctx, mc.Config.BlockProposalMaxWaitTime())

	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*transaction.Transaction)
	collectionName := txn.GetCollectionName()
	logging.Logger.Info("generate block starting iteration", zap.Int64("round", b.Round), zap.String("prev_block", b.PrevHash), zap.String("prev_state_hash", util.ToHex(b.PrevBlock.ClientStateHash)))
	err := transactionEntityMetadata.GetStore().IterateCollection(cctx, transactionEntityMetadata, collectionName, txnIterHandler)
	if len(iterInfo.invalidTxns) > 0 {
		logging.Logger.Info("generate block (found txns very old)", zap.Any("round", b.Round), zap.Int("num_invalid_txns", len(iterInfo.invalidTxns)))
		go mc.deleteTxns(iterInfo.invalidTxns) // OK to do in background
	}
	if iterInfo.roundMismatch {
		logging.Logger.Debug("generate block (round mismatch)", zap.Any("round", b.Round), zap.Any("current_round", mc.GetCurrentRound()))
		return ErrRoundMismatch
	}
	if iterInfo.roundTimeout {
		logging.Logger.Debug("generate block (round timeout)", zap.Any("round", b.Round), zap.Any("current_round", mc.GetCurrentRound()))
		return ErrRoundTimeout
	}
	if iterInfo.reInclusionErr != nil {
		logging.Logger.Error("generate block (txn reinclusion check)",
			zap.Any("round", b.Round), zap.Error(iterInfo.reInclusionErr))
	}

	switch err {
	case context.DeadlineExceeded:
		logging.Logger.Debug("Slow block generation, stopping transaction collection and finishing the block")
	case context.Canceled:
		logging.Logger.Debug("Context cancelled, rejecting current block")
		return err
	default:
		if err != nil {
			return err
		}
	}

	blockSize := iterInfo.idx
	var reusedTxns int32
	if blockSize < mc.BlockSize() && iterInfo.byteSize < mc.MaxByteSize() && mc.ReuseTransactions() &&
		err != context.DeadlineExceeded && iterInfo.cost < mc.Config.MaxBlockCost() {
		blocks := mc.GetUnrelatedBlocks(10, b)
		rcount := 0
		for _, ub := range blocks {
			for _, txn := range ub.Txns {
				rcount++
				rtxn := mc.txnToReuse(txn)
				needsVerification := (ub.MinerID != node.Self.Underlying().GetKey() || ub.GetVerificationStatus() != block.VerificationSuccessful)
				if needsVerification {
					//TODO remove context, since it is not used here
					if err := rtxn.ValidateWrtTime(cctx, ub.CreationDate); err != nil {
						continue
					}
				}
				if txnProcessor(cctx, blockState, rtxn, iterInfo) {
					if iterInfo.idx == mc.BlockSize() || iterInfo.byteSize >= mc.MaxByteSize() {
						break
					}
				}
			}
			if iterInfo.idx == mc.BlockSize() || iterInfo.byteSize >= mc.MaxByteSize() {
				break
			}
		}
		reusedTxns = iterInfo.idx - blockSize
		blockSize = iterInfo.idx
		logging.Logger.Error("generate block (reused txns)",
			zap.Int64("round", b.Round), zap.Int("ub", len(blocks)),
			zap.Int32("reused", reusedTxns), zap.Int("rcount", rcount),
			zap.Int32("blockSize", iterInfo.idx))
	}
	if blockSize != mc.BlockSize() && iterInfo.byteSize < mc.MaxByteSize() {
		if !waitOver && blockSize < mc.MinBlockSize() {
			b.Txns = nil
			logging.Logger.Debug("generate block (insufficient txns)",
				zap.Int64("round", b.Round),
				zap.Int32("iteration_count", iterInfo.count),
				zap.Int32("block_size", blockSize))
			return common.NewError(InsufficientTxns,
				fmt.Sprintf("not sufficient txns to make a block yet for round %v (iterated %v,block_size %v,state failure %v, invalid %v,reused %v)",
					b.Round, iterInfo.count, blockSize, iterInfo.failedStateCount, len(iterInfo.invalidTxns), 0))
		}
		b.Txns = b.Txns[:blockSize]
		iterInfo.eTxns = iterInfo.eTxns[:blockSize]
	}

	if config.DevConfiguration.IsFeeEnabled {
		err = mc.processTxn(ctx, mc.createFeeTxn(b), b, blockState, iterInfo.clients)
		if err != nil {
			logging.Logger.Error("generate block (payFees)", zap.Int64("round", b.Round), zap.Error(err))
		}
	}

	if config.DevConfiguration.IsBlockRewards &&
		b.Round%config.SmartContractConfig.GetInt64("smart_contracts.storagesc.block_reward.trigger_period") == 0 {
		logging.Logger.Info("start_block_rewards", zap.Int64("round", b.Round))
		err = mc.processTxn(ctx, mc.createBlockRewardTxn(b), b, blockState, iterInfo.clients)
		if err != nil {
			logging.Logger.Error("generate block (blockRewards)", zap.Int64("round", b.Round), zap.Error(err))
		}
	}

	if mc.SmartContractSettingUpdatePeriod() != 0 &&
		b.Round%mc.SmartContractSettingUpdatePeriod() == 0 {
		err = mc.processTxn(ctx, mc.storageScCommitSettingChangesTx(b), b, blockState, iterInfo.clients)
		if err != nil {
			logging.Logger.Error("generate block (commit settings)", zap.Int64("round", b.Round), zap.Error(err))
		}
	}

	b.RunningTxnCount = b.PrevBlock.RunningTxnCount + int64(len(b.Txns))
	if iterInfo.count > 10*mc.BlockSize() {
		logging.Logger.Info("generate block (too much iteration)", zap.Int64("round", b.Round), zap.Int32("iteration_count", iterInfo.count))
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
	return nil
}
