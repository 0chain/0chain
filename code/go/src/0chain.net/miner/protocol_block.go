package miner

import (
	"context"
	"fmt"
	"sort"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chain"
	"0chain.net/config"
	"0chain.net/util"

	"0chain.net/block"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/transaction"

	"go.uber.org/zap"
)

//InsufficientTxns - to indicate an error when the transactions are not sufficient to make a block
const InsufficientTxns = "insufficient_txns"

var bgTimer metrics.Timer
var bvTimer metrics.Timer

func init() {
	bgTimer = metrics.GetOrRegisterTimer("bg_time", nil)
	bvTimer = metrics.GetOrRegisterTimer("bv_time", nil)
}

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (mc *Chain) GenerateBlock(ctx context.Context, b *block.Block, bsh chain.BlockStateHandler, waitOver bool) error {
	clients := make(map[string]*client.Client)
	b.Txns = make([]*transaction.Transaction, mc.BlockSize)
	//wasting this because []interface{} != []*transaction.Transaction in Go
	etxns := make([]datastore.Entity, mc.BlockSize)
	var invalidTxns []datastore.Entity
	var idx int32
	var ierr error
	var count int32
	var roundMismatch bool
	var hasOwnerTxn bool
	var failedStateCount int32
	var byteSize int64
	txnMap := make(map[datastore.Key]bool, mc.BlockSize)
	var txnProcessor = func(ctx context.Context, txn *transaction.Transaction) bool {
		if _, ok := txnMap[txn.GetKey()]; ok {
			return false
		}
		var debugTxn = txn.DebugTxn()
		if debugTxn {
			Logger.Info("generate block (debug transaction)", zap.String("txn", txn.Hash), zap.String("txn_object", datastore.ToJSON(txn).String()))
		}
		if !mc.validateTransaction(b, txn) {
			invalidTxns = append(invalidTxns, txn)
			if debugTxn {
				Logger.Info("generate block (debug transaction) error - txn creation not within tolerance", zap.String("txn", txn.Hash), zap.Any("now", common.Now()))
			}
			return false
		}
		if ok, err := mc.ChainHasTransaction(ctx, b.PrevBlock, txn); ok || err != nil {
			if err != nil {
				ierr = err
			}
			return false
		}
		if !mc.UpdateState(b, txn) {
			failedStateCount++
			if config.DevConfiguration.State {
				return false
			}
		}
		if txn.ClientID == mc.OwnerID {
			hasOwnerTxn = true
		}
		//Setting the score lower so the next time blocks are generated these transactions don't show up at the top
		txn.SetCollectionScore(txn.GetCollectionScore() - 10*60)
		txnMap[txn.GetKey()] = true
		b.Txns[idx] = txn
		etxns[idx] = txn
		b.AddTransaction(txn)
		byteSize += int64(len(txn.TransactionData)) + int64(len(txn.TransactionOutput))
		if txn.PublicKey == "" {
			clients[txn.ClientID] = nil
		}
		idx++
		return true
	}
	var txnIterHandler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		count++
		if mc.CurrentRound > b.Round {
			roundMismatch = true
			return false
		}
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			Logger.Error("generate block (invalid entity)", zap.Any("entity", qe))
			return true
		}
		if txnProcessor(ctx, txn) {
			if idx >= mc.BlockSize || byteSize >= mc.MaxByteSize {
				return false
			}
		}
		return true
	}
	start := time.Now()
	b.CreationDate = common.Now()
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*transaction.Transaction)
	collectionName := txn.GetCollectionName()
	Logger.Info("generate block starting iteration", zap.Int64("round", b.Round), zap.String("prev_block", b.PrevHash), zap.String("prev_state_hash", util.ToHex(b.PrevBlock.ClientStateHash)))
	err := transactionEntityMetadata.GetStore().IterateCollection(ctx, transactionEntityMetadata, collectionName, txnIterHandler)
	if len(invalidTxns) > 0 {
		Logger.Info("generate block (found txns very old)", zap.Any("round", b.Round), zap.Int("num_invalid_txns", len(invalidTxns)))
		go mc.deleteTxns(invalidTxns) // OK to do in background
	}
	if roundMismatch {
		Logger.Debug("generate block (round mismatch)", zap.Any("round", b.Round), zap.Any("current_round", mc.CurrentRound))
		return common.NewError(RoundMismatch, "current round different from generation round")
	}
	if ierr != nil {
		Logger.Error("generate block (txn reinclusion check)", zap.Any("round", b.Round), zap.Error(ierr))
	}
	if err != nil {
		return err
	}
	blockSize := idx
	var reusedTxns int32
	if blockSize < mc.BlockSize && byteSize < mc.MaxByteSize && mc.ReuseTransactions {
		blocks := mc.GetUnrelatedBlocks(10, b)
		rcount := 0
		for _, ub := range blocks {
			for _, txn := range ub.Txns {
				rcount++
				if txnProcessor(ctx, mc.txnToReuse(txn)) {
					if idx == mc.BlockSize || byteSize >= mc.MaxByteSize {
						break
					}
				}
			}
			if idx == mc.BlockSize || byteSize >= mc.MaxByteSize {
				break
			}
		}
		reusedTxns = idx - blockSize
		blockSize = idx
		Logger.Info("generate block (reused txns)", zap.Int64("round", b.Round), zap.Int("ub", len(blocks)), zap.Int32("reused", reusedTxns), zap.Int("rcount", rcount), zap.Int32("blockSize", idx))
	}
	if blockSize != mc.BlockSize && byteSize < mc.MaxByteSize {
		if !waitOver || blockSize < mc.MinBlockSize {
			b.Txns = nil
			Logger.Debug("generate block (insufficient txns)", zap.Int64("round", b.Round), zap.Int32("iteration_count", count), zap.Int32("block_size", blockSize))
			return common.NewError(InsufficientTxns, fmt.Sprintf("not sufficient txns to make a block yet for round %v (iterated %v,block_size %v,state failure %v, invalid %v,reused %v)", b.Round, count, blockSize, failedStateCount, len(invalidTxns), reusedTxns))
		}
		b.Txns = b.Txns[:blockSize]
		etxns = etxns[:blockSize]
	}
	if count > 10*mc.BlockSize {
		Logger.Info("generate block (too much iteration)", zap.Int64("round", b.Round), zap.Int32("iteration_count", count))
	}
	client.GetClients(ctx, clients)
	Logger.Debug("generate block (assemble)", zap.Int64("round", b.Round), zap.Duration("time", time.Since(start)))

	bsh.UpdatePendingBlock(ctx, b, etxns)
	for _, txn := range b.Txns {
		if txn.PublicKey != "" {
			txn.ClientID = datastore.EmptyKey
			continue
		}
		client := clients[txn.ClientID]
		if client == nil || client.PublicKey == "" {
			Logger.Error("generate block (invalid client)", zap.String("client_id", txn.ClientID))
			return common.NewError("invalid_client", "client not available")
		}
		txn.PublicKey = client.PublicKey
		txn.ClientID = datastore.EmptyKey
	}
	b.ClientStateHash = b.ClientState.GetRoot()
	bgTimer.UpdateSince(start)
	Logger.Debug("generate block (assemble+update)", zap.Int64("round", b.Round), zap.Duration("time", time.Since(start)))

	self := node.GetSelfNode(ctx)
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)
	if err != nil {
		return err
	}
	b.SetBlockState(block.StateGenerated)
	b.SetStateStatus(block.StateSuccessful)
	Logger.Info("generate block (assemble+update+sign)", zap.Int64("round", b.Round), zap.Int32("block_size", blockSize), zap.Int32("reused_txns", reusedTxns), zap.Duration("time", time.Since(start)),
		zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Int8("state_status", b.GetStateStatus()),
		zap.Float64("p_chain_weight", b.PrevBlock.ChainWeight), zap.Int32("iteration_count", count))
	mc.StateSanityCheck(ctx, b)
	go b.ComputeTxnMap()
	return nil
}

func (mc *Chain) txnToReuse(txn *transaction.Transaction) *transaction.Transaction {
	ctxn := *txn
	ctxn.OutputHash = ""
	return &ctxn
}

func (mc *Chain) validateTransaction(b *block.Block, txn *transaction.Transaction) bool {
	if !common.WithinTime(int64(b.CreationDate), int64(txn.CreationDate), transaction.TXN_TIME_TOLERANCE) {
		return false
	}
	return true
}

/*UpdatePendingBlock - updates the block that is generated and pending rest of the process */
func (mc *Chain) UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity) {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")

	//NOTE: Since we are not explicitly maintaining state in the db, we just need to adjust the collection score and don't need to write the entities themselves
	//transactionMetadataProvider.GetStore().MultiWrite(ctx, transactionMetadataProvider, txns)
	transactionMetadataProvider.GetStore().MultiAddToCollection(ctx, transactionMetadataProvider, txns)
}

/*VerifyBlock - given a set of transaction ids within a block, validate the block */
func (mc *Chain) VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	start := time.Now()
	err := b.Validate(ctx)
	if err != nil {
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
		if config.DevConfiguration.State {
			Logger.Error("verify block - error computing state (TODO sync)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Error(serr))
			return nil, serr
		}
	}
	bvt, err := mc.SignBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	bvTimer.UpdateSince(start)
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
	validChannel := make(chan bool, len(b.Txns)/mc.ValidationBatchSize+1)
	validate := func(ctx context.Context, txns []*transaction.Transaction) {
		for _, txn := range txns {
			if cancel {
				validChannel <- false
				return
			}
			if mc.CurrentRound > b.Round {
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
			err := txn.ValidateWrtTime(ctx, b.CreationDate)
			if err != nil {
				cancel = true
				Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()), zap.Error(err))
				validChannel <- false
				return
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
	for start := 0; start < len(b.Txns); start += mc.ValidationBatchSize {
		end := start + mc.ValidationBatchSize
		if end > len(b.Txns) {
			end = len(b.Txns)
		}
		go validate(ctx, b.Txns[start:end])
	}
	count := 0
	for result := range validChannel {
		if roundMismatch {
			Logger.Info("validate transactions (round mismatch)", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("current_round", mc.CurrentRound))
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
	if mc.DiscoverClients {
		go mc.SaveClients(ctx, b.GetClients())
	}
	return nil
}

/*SignBlock - sign the block and provide the verification ticket */
func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	var bvt = &block.BlockVerificationTicket{}
	bvt.BlockID = b.Hash
	bvt.Round = b.Round
	self := node.GetSelfNode(ctx)
	var err error
	bvt.VerifierID = self.GetKey()
	bvt.Signature, err = self.Sign(b.Hash)
	b.SetVerificationStatus(block.VerificationSuccessful)
	if err != nil {
		return nil, err
	}
	return bvt, nil
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("lf_round", mc.LatestFinalizedBlock.Round), zap.Int64("current_round", mc.CurrentRound), zap.Float64("weight", b.Weight()), zap.Float64("chain_weight", b.ChainWeight))
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
	mc.Sharders.OneTimeStatusMonitor(ctx)
	lfBlocks := mc.GetLatestFinalizedBlockFromSharder(ctx)
	//Sorting as per the latest finalized blocks from all the sharders
	sort.Slice(lfBlocks, func(i int, j int) bool { return lfBlocks[i].Round >= lfBlocks[j].Round })
	if len(lfBlocks) > 0 {
		Logger.Info("bc-1 latest finalized Block", zap.Int64("lfb_round", lfBlocks[0].Round))
		return lfBlocks[0]
	}
	Logger.Info("bc-1 sharders returned no lfb.")
	return nil
}
