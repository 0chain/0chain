// +build integration_tests

package miner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"

	crpc "0chain.net/conductor/conductrpc"
	crpcutils "0chain.net/conductor/utils"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	var state = crpc.Client().State()

	if !state.SignOnlyCompetingBlocks.IsCompetingGroupMember(state, b.MinerID) {
		return nil, errors.New("skip block signing -- not competing block")
	}

	// regular or competing signing
	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var (
		self  = node.Self
		state = crpc.Client().State()
	)
	b.HashBlock()

	switch {
	case state.WrongBlockHash != nil:
		b.Hash = revertString(b.Hash) // just wrong block hash
		b.Signature, err = self.Sign(b.Hash)
	case state.WrongBlockSignHash != nil:
		b.Signature, err = self.Sign(revertString(b.Hash)) // sign another hash
	case state.WrongBlockSignKey != nil:
		b.Signature, err = crpcutils.Sign(b.Hash) // wrong secret key
	default:
		b.Signature, err = self.Sign(b.Hash)
	}

	return
}

// has double-spend transaction
func hasDST(pb, b []*transaction.Transaction) (has bool) {
	for _, bx := range b {
		if bx == nil {
			continue
		}
		for _, pbx := range pb {
			if pbx == nil {
				continue
			}
			if bx.Hash == pbx.Hash {
				return true // has
			}
		}
	}
	return false // has not
}

func (mc *Chain) GenerateBlock(ctx context.Context, b *block.Block,
	bsh chain.BlockStateHandler, waitOver bool) error {

	var clients = make(map[string]*client.Client)
	b.Txns = make([]*transaction.Transaction, mc.BlockSize())

	// wasting this because []interface{} != []*transaction.Transaction in Go
	var (
		etxns  = make([]datastore.Entity, mc.BlockSize())
		txnMap = make(map[datastore.Key]bool, mc.BlockSize())

		invalidTxns      []datastore.Entity
		idx              int32
		ierr             error
		count            int32
		roundMismatch    bool
		roundTimeout     bool
		failedStateCount int32
		byteSize         int64

		state         = crpc.Client().State()
		pb            = b.PrevBlock
		selfKey       = node.Self.GetKey()
		isDoubleSpend bool
		dstxn         *transaction.Transaction

		bState = block.CreateStateWithPreviousBlock(b.PrevBlock, mc.GetStateDB(), b.Round)
	)

	isDoubleSpend = state.DoubleSpendTransaction.IsBy(state, selfKey) &&
		pb != nil && len(pb.Txns) > 0 && len(pb.Txns) > 0 &&
		!hasDST(b.Txns, pb.Txns)

	if isDoubleSpend {
		dstxn = pb.Txns[rand.Intn(len(pb.Txns))] // a random one
	}

	var txnProcessor = func(ctx context.Context, bState util.MerklePatriciaTrieI, txn *transaction.Transaction) bool {
		if _, ok := txnMap[txn.GetKey()]; ok {
			return false
		}
		var debugTxn = txn.DebugTxn()
		if !mc.validateTransaction(b, txn) {
			invalidTxns = append(invalidTxns, txn)
			if debugTxn {
				logging.Logger.Info("generate block (debug transaction) error - txn creation not within tolerance", zap.String("txn", txn.Hash), zap.Int32("idx", idx), zap.Any("now", common.Now()))
			}
			return false
		}
		if debugTxn {
			logging.Logger.Info("generate block (debug transaction)", zap.String("txn", txn.Hash), zap.Int32("idx", idx), zap.String("txn_object", datastore.ToJSON(txn).String()))
		}
		if dstxn == nil || (dstxn != nil && txn.Hash != dstxn.Hash) {
			if ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, txn); ok || err != nil {
				if err != nil {
					ierr = err
				}
				return false
			}
		}
		events, err := mc.UpdateState(ctx, b, bState, txn)
		b.Events = append(b.Events, events...)
		if err != nil {
			if debugTxn {
				logging.Logger.Error("generate block (debug transaction) update state", zap.String("txn", txn.Hash), zap.Int32("idx", idx), zap.String("txn_object", datastore.ToJSON(txn).String()), zap.Error(err))
			}
			failedStateCount++
			return false
		}

		// Setting the score lower so the next time blocks are generated
		// these transactions don't show up at the top
		txn.SetCollectionScore(txn.GetCollectionScore() - 10*60)
		txnMap[txn.GetKey()] = true
		b.Txns[idx] = txn
		if debugTxn {
			logging.Logger.Info("generate block (debug transaction) success in processing Txn hash: " + txn.Hash + " blockHash? = " + b.Hash)
		}
		etxns[idx] = txn
		b.AddTransaction(txn)
		byteSize += int64(len(txn.TransactionData)) + int64(len(txn.TransactionOutput))
		if txn.PublicKey == "" {
			clients[txn.ClientID] = nil
		}
		idx++
		return true
	}
	var roundTimeoutCount = mc.GetRoundTimeoutCount()
	var txnIterHandler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		count++
		if mc.GetCurrentRound() > b.Round {
			roundMismatch = true
			return false
		}
		if roundTimeoutCount != mc.GetRoundTimeoutCount() {
			roundTimeout = true
			return false
		}
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			logging.Logger.Error("generate block (invalid entity)", zap.Any("entity", qe))
			return true
		}
		if txnProcessor(ctx, bState, txn) {
			if idx >= mc.BlockSize() || byteSize >= mc.MaxByteSize() {
				return false
			}
		}
		return true
	}
	start := time.Now()
	b.CreationDate = common.Now()
	if b.CreationDate < b.PrevBlock.CreationDate {
		b.CreationDate = b.PrevBlock.CreationDate
	}
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*transaction.Transaction)
	collectionName := txn.GetCollectionName()
	logging.Logger.Info("generate block starting iteration", zap.Int64("round", b.Round), zap.String("prev_block", b.PrevHash), zap.String("prev_state_hash", util.ToHex(b.PrevBlock.ClientStateHash)))
	if isDoubleSpend {
		txnIterHandler(ctx, dstxn) // inject double-spend transaction
	}
	err := transactionEntityMetadata.GetStore().IterateCollection(ctx, transactionEntityMetadata, collectionName, txnIterHandler)
	if len(invalidTxns) > 0 {
		logging.Logger.Info("generate block (found txns very old)", zap.Any("round", b.Round), zap.Int("num_invalid_txns", len(invalidTxns)))
		go mc.deleteTxns(invalidTxns) // OK to do in background
	}
	if roundMismatch {
		logging.Logger.Debug("generate block (round mismatch)", zap.Any("round", b.Round), zap.Any("current_round", mc.GetCurrentRound()))
		return ErrRoundMismatch
	}
	if roundTimeout {
		logging.Logger.Debug("generate block (round timeout)", zap.Any("round", b.Round), zap.Any("current_round", mc.GetCurrentRound()))
		return ErrRoundTimeout
	}
	if ierr != nil {
		logging.Logger.Error("generate block (txn reinclusion check)", zap.Any("round", b.Round), zap.Error(ierr))
	}
	if err != nil {
		return err
	}
	blockSize := idx
	var reusedTxns int32
	if blockSize < mc.BlockSize() && byteSize < mc.MaxByteSize() && mc.ReuseTransactions() {
		blocks := mc.GetUnrelatedBlocks(10, b)
		rcount := 0
		for _, ub := range blocks {
			for _, txn := range ub.Txns {
				rcount++
				rtxn := mc.txnToReuse(txn)
				needsVerification := (ub.MinerID != node.Self.Underlying().GetKey() || ub.GetVerificationStatus() != block.VerificationSuccessful)
				if needsVerification {
					if err := rtxn.ValidateWrtTime(ctx, ub.CreationDate); err != nil {
						continue
					}
				}
				if txnProcessor(ctx, bState, rtxn) {
					if idx == mc.BlockSize() || byteSize >= mc.MaxByteSize() {
						break
					}
				}
			}
			if idx == mc.BlockSize() || byteSize >= mc.MaxByteSize() {
				break
			}
		}
		reusedTxns = idx - blockSize
		blockSize = idx
		logging.Logger.Error("generate block (reused txns)",
			zap.Int64("round", b.Round), zap.Int("ub", len(blocks)),
			zap.Int32("reused", reusedTxns), zap.Int("rcount", rcount),
			zap.Int32("blockSize", idx))
	}
	if blockSize != mc.BlockSize() && byteSize < mc.MaxByteSize() {
		if !waitOver && blockSize < mc.MinBlockSize() {
			b.Txns = nil
			logging.Logger.Debug("generate block (insufficient txns)",
				zap.Int64("round", b.Round),
				zap.Int32("iteration_count", count),
				zap.Int32("block_size", blockSize))
			return common.NewError(InsufficientTxns, fmt.Sprintf("not sufficient txns to make a block yet for round %v (iterated %v,block_size %v,state failure %v, invalid %v,reused %v)", b.Round, count, blockSize, failedStateCount, len(invalidTxns), reusedTxns))
		}
		b.Txns = b.Txns[:blockSize]
		etxns = etxns[:blockSize]
	}
	if config.DevConfiguration.IsFeeEnabled {
		err = mc.processTxn(ctx, mc.createFeeTxn(b), b, bState, clients)
		if err != nil {
			return err
		}
	}
	if config.DevConfiguration.IsBlockRewards {
		err = mc.processTxn(ctx, mc.createBlockRewardTxn(b), b, bState, clients)
		if err != nil {
			return err
		}
	}
	b.RunningTxnCount = b.PrevBlock.RunningTxnCount + int64(len(b.Txns))
	if count > 10*mc.BlockSize() {
		logging.Logger.Info("generate block (too much iteration)", zap.Int64("round", b.Round), zap.Int32("iteration_count", count))
	}

	if err = client.GetClients(ctx, clients); err != nil {
		logging.Logger.Error("generate block (get clients error)", zap.Error(err))
		return common.NewError("get_clients_error", err.Error())
	}

	logging.Logger.Debug("generate block (assemble)", zap.Int64("round", b.Round), zap.Duration("time", time.Since(start)))

	bsh.UpdatePendingBlock(ctx, b, etxns)
	for _, txn := range b.Txns {
		if txn.PublicKey != "" {
			txn.ClientID = datastore.EmptyKey
			continue
		}
		cl := clients[txn.ClientID]
		if cl == nil || cl.PublicKey == "" {
			logging.Logger.Error("generate block (invalid client)", zap.String("client_id", txn.ClientID))
			return common.NewError("invalid_client", "client not available")
		}
		txn.PublicKey = cl.PublicKey
		txn.ClientID = datastore.EmptyKey
	}
	b.ClientStateHash = b.ClientState.GetRoot()
	bgTimer.UpdateSince(start)
	logging.Logger.Debug("generate block (assemble+update)", zap.Int64("round", b.Round), zap.Duration("time", time.Since(start)))

	if err = mc.hashAndSignGeneratedBlock(ctx, b); err != nil {
		return err
	}

	b.SetBlockState(block.StateGenerated)
	b.SetStateStatus(block.StateSuccessful)
	logging.Logger.Info("generate block (assemble+update+sign)", zap.Int64("round", b.Round), zap.Int32("block_size", blockSize), zap.Int32("reused_txns", reusedTxns), zap.Duration("time", time.Since(start)),
		zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Int8("state_status", b.GetStateStatus()),
		zap.Float64("p_chain_weight", b.PrevBlock.ChainWeight), zap.Int32("iteration_count", count))
	block.StateSanityCheck(ctx, b)
	b.ComputeTxnMap()
	bsHistogram.Update(int64(len(b.Txns)))
	node.Self.Underlying().Info.AvgBlockTxns = int(math.Round(bsHistogram.Mean()))
	return nil
}
