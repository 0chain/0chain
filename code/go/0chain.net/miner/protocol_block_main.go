//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/core/util"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var self = node.Self
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)
	return
}

func (mc *Chain) GenerateBlock(ctx context.Context, b *block.Block,
	bsh chain.BlockStateHandler, waitOver bool) error {

	return mc.generateBlockWorker.Run(ctx, func() error {
		return mc.generateBlock(ctx, b, minerChain, waitOver)
	})
}

type txnProcessorHandler func(context.Context, util.MerklePatriciaTrieI, *transaction.Transaction, *TxnIterInfo) bool

func txnProcessorHandlerFunc(mc *Chain, b *block.Block) txnProcessorHandler {
	return func(ctx context.Context, bState util.MerklePatriciaTrieI, txn *transaction.Transaction, tii *TxnIterInfo) bool {
		if _, ok := tii.txnMap[txn.GetKey()]; ok {
			return false
		}
		var debugTxn = txn.DebugTxn()

		err := mc.validateTransaction(b, bState, txn)
		switch err {
		case PastTransaction:
			tii.invalidTxns = append(tii.invalidTxns, txn)
			if debugTxn {
				logging.Logger.Info("generate block (debug transaction) error, transaction hash old nonce",
					zap.String("txn", txn.Hash), zap.Int32("idx", tii.idx),
					zap.Any("now", common.Now()), zap.Int64("nonce", txn.Nonce))
			}
			return false
		case FutureTransaction:
			list := tii.futureTxns[txn.ClientID]
			list = append(list, txn)
			sort.SliceStable(list, func(i, j int) bool {
				if list[i].Nonce == list[i].Nonce {
					//if the same nonce order by fee
					return list[i].Fee > list[i].Fee
				}
				return list[i].Nonce < list[i].Nonce
			})
			tii.futureTxns[txn.ClientID] = list
			return false
		case ErrNotTimeTolerant:
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
		//txn.SetCollectionScore(txn.GetCollectionScore() - 10*60)
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

		return true
	}
}

type TxnIterInfo struct {
	clients     map[string]*client.Client
	eTxns       []datastore.Entity
	invalidTxns []datastore.Entity
	futureTxns  map[datastore.Key][]*transaction.Transaction
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
}

func (tii *TxnIterInfo) checkForCurrent(txn *transaction.Transaction) {
	if tii.futureTxns == nil {
		return
	}
	//check whether we can execute future transactions
	futures := tii.futureTxns[txn.ClientID]
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
			tii.invalidTxns = append(tii.invalidTxns, futures[i])
			continue
		}

		currentNonce = futures[i].Nonce
		tii.currentTxns = append(tii.currentTxns, futures[i])
		//will not sorted by fee here but at least will be sorted by nonce correctly, can improve it
		sort.SliceStable(tii.currentTxns, func(i, j int) bool { return tii.currentTxns[i].Nonce < tii.currentTxns[i].Nonce })
	}

	if i > -1 {
		tii.futureTxns[txn.ClientID] = futures[i:]
	}
}

func newTxnIterInfo(blockSize int32) *TxnIterInfo {
	return &TxnIterInfo{
		clients:    make(map[string]*client.Client),
		eTxns:      make([]datastore.Entity, 0, blockSize),
		futureTxns: make(map[datastore.Key][]*transaction.Transaction),
		txnMap:     make(map[datastore.Key]struct{}, blockSize),
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
		logging.Logger.Info("generate block (found invalid transactions)", zap.Any("round", b.Round), zap.Int("num_invalid_txns", len(iterInfo.invalidTxns)))
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

	rcount := 0
	if blockSize < mc.BlockSize() && iterInfo.byteSize < mc.MaxByteSize() && len(iterInfo.currentTxns) > 0 && err != context.DeadlineExceeded {
		for _, txn := range iterInfo.currentTxns {
			if txnProcessor(ctx, blockState, txn, iterInfo) {
				rcount++
				if iterInfo.idx == mc.BlockSize() || iterInfo.byteSize >= mc.MaxByteSize() {
					break
				}
			}
		}
		logging.Logger.Debug("Processed current transactions", zap.Int("count", rcount))
	}
	//reuse current transactions here
	if blockSize < mc.BlockSize() && iterInfo.byteSize < mc.MaxByteSize() && mc.ReuseTransactions() && err != context.DeadlineExceeded {
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
		err = mc.processTxn(ctx, mc.createFeeTxn(b, blockState), b, blockState, iterInfo.clients)
		if err != nil {
			logging.Logger.Error("generate block (payFees)", zap.Int64("round", b.Round), zap.Error(err))
		}
	}

	if config.DevConfiguration.IsBlockRewards {
		err = mc.processTxn(ctx, mc.createBlockRewardTxn(b, blockState), b, blockState, iterInfo.clients)
		if err != nil {
			logging.Logger.Error("generate block (blockRewards)", zap.Int64("round", b.Round), zap.Error(err))
		}
	}

	if mc.SmartContractSettingUpdatePeriod() != 0 &&
		b.Round%mc.SmartContractSettingUpdatePeriod() == 0 {
		err = mc.processTxn(ctx, mc.storageScCommitSettingChangesTx(b, blockState), b, blockState, iterInfo.clients)
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
