package miner

import (
	"context"
	"fmt"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chain"
	"0chain.net/config"

	"0chain.net/block"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/transaction"
	"0chain.net/util"
	"go.uber.org/zap"
)

const InsufficientTxns = "insufficient_txns"

var bgTimer metrics.Timer
var bvTimer metrics.Timer

func init() {
	bgTimer = metrics.GetOrRegisterTimer("bg_time", nil)
	bvTimer = metrics.GetOrRegisterTimer("bv_time", nil)
}

/*StartRound - start a new round */
func (mc *Chain) StartRound(ctx context.Context, r *Round) {
	mc.AddRound(r)
}

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (mc *Chain) GenerateBlock(ctx context.Context, b *block.Block, bsh chain.BlockStateHandler) error {
	clients := make(map[string]*client.Client)
	pndb := b.PrevBlock.ClientStateMT.GetNodeDB()
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, pndb, false)
	b.ClientStateMT = util.NewMerklePatriciaTrie(ndb)

	b.Txns = make([]*transaction.Transaction, mc.BlockSize)
	//wasting this because []interface{} != []*transaction.Transaction in Go
	etxns := make([]datastore.Entity, mc.BlockSize)
	var invalidTxns []datastore.Entity
	var idx int32
	var ierr error
	var count int32
	var roundMismatch bool
	var hasOwnerTxn bool
	var txnIterHandler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		if mc.CurrentRound > b.Round {
			roundMismatch = true
			return false
		}
		count++
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			Logger.Error("generate block (invalid entity)", zap.Any("entity", qe))
			return true
		}
		var debugTxn = txn.DebugTxn()

		if debugTxn {
			Logger.Info("generate block (debug transaction)", zap.String("txn", txn.Hash), zap.String("txn_object", datastore.ToJSON(txn).String()))
		}
		if !mc.validateTransaction(txn) {
			invalidTxns = append(invalidTxns, qe)
			if debugTxn {
				Logger.Info("generate block (debug transaction) error - txn creation not within tolerance", zap.String("txn", txn.Hash), zap.Any("now", common.Now()))
			}
			return true
		}
		if ok, err := b.PrevBlock.ChainHasTransaction(txn); ok || err != nil {
			if err != nil {
				ierr = err
			}
			return true
		}
		if !mc.UpdateState(txn, b) {
			return true
		}
		if txn.ClientID == mc.OwnerID {
			hasOwnerTxn = true
		}
		//Setting the score lower so the next time blocks are generated these transactions don't show up at the top
		txn.SetCollectionScore(txn.GetCollectionScore() - 10*60)
		b.Txns[idx] = txn
		etxns[idx] = txn
		b.AddTransaction(txn)
		clients[txn.ClientID] = nil
		idx++

		childTxns := txn.GenerateChildTransactions(ctx)
		if childTxns != nil {
			for _, ctxn := range childTxns {
				b.Txns[idx] = ctxn
				etxns[idx] = ctxn
				b.AddTransaction(ctxn)
				clients[ctxn.ClientID] = nil
				idx++
			}
		}

		if idx >= mc.BlockSize {
			return false
		}
		return true
	}

	start := time.Now()
	b.CreationDate = common.Now()
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*transaction.Transaction)
	collectionName := txn.GetCollectionName()
	err := transactionEntityMetadata.GetStore().IterateCollection(ctx, transactionEntityMetadata, collectionName, txnIterHandler)
	if roundMismatch {
		Logger.Debug("generate block (round mismatch)", zap.Any("round", b.Round), zap.Any("current_round", mc.CurrentRound))
		return common.NewError(RoundMismatch, "current round different from generation round")
	}
	if ierr != nil {
		Logger.Error("generate block (txn reinclusion check)", zap.Any("round", b.Round), zap.Error(ierr))
	}
	if len(invalidTxns) > 0 {
		Logger.Info("generate block (found txns very old)", zap.Any("round", b.Round), zap.Int("num_invalid_txns", len(invalidTxns)))
		go mc.deleteTxns(invalidTxns) // OK to do in background
	}
	if err != nil {
		return err
	}
	if idx != mc.BlockSize {
		if !hasOwnerTxn {
			b.Txns = nil
			Logger.Debug("generate block (insufficient txns)", zap.Int64("round", b.Round), zap.Int32("iteration_count", count), zap.Int32("block_size", mc.BlockSize), zap.Int32("num_txns", idx))
			return common.NewError(InsufficientTxns, fmt.Sprintf("not sufficient txns to make a block yet for round %v (iterated %v, invalid %v)", b.Round, count, len(invalidTxns)))
		}
		b.Txns = b.Txns[:idx]
		etxns = etxns[:idx]
	}
	if count > 10*mc.BlockSize {
		Logger.Info("generate block (too much iteration)", zap.Int64("round", b.Round), zap.Int32("iteration_count", count))
	}
	client.GetClients(ctx, clients)
	Logger.Debug("generate block (assemble)", zap.Int64("round", b.Round), zap.Duration("time", time.Since(start)))

	bsh.UpdatePendingBlock(ctx, b, etxns)
	for _, txn := range b.Txns {
		client := clients[txn.ClientID]
		if client == nil || client.PublicKey == "" {
			Logger.Error("generate block (invalid client)", zap.String("client_id", txn.ClientID))
			return common.NewError("invalid_client", "client not available")
		}
		txn.PublicKey = client.PublicKey
		txn.ClientID = datastore.EmptyKey
	}
	bgTimer.UpdateSince(start)
	Logger.Debug("generate block (assemble+update)", zap.Int64("round", b.Round), zap.Duration("time", time.Since(start)))

	self := node.GetSelfNode(ctx)
	b.MinerID = self.ID
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)
	if err != nil {
		return err
	}
	Logger.Info("generate block (assemble+update+sign)", zap.Int64("round", b.Round), zap.Duration("time", time.Since(start)), zap.String("block", b.Hash), zap.String("prev_block", b.PrevBlock.Hash), zap.Int32("iteration_count", count), zap.Float64("p_chain_weight", b.PrevBlock.ChainWeight))
	go b.ComputeTxnMap()
	return nil
}

func (mc *Chain) validateTransaction(txn *transaction.Transaction) bool {
	if !common.Within(int64(txn.CreationDate), transaction.TXN_TIME_TOLERANCE-1) {
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
	err = mc.ValidateTransactions(ctx, b)
	if err != nil {
		return nil, err
	}
	bvt, err := mc.SignBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	bvTimer.UpdateSince(start)
	Logger.Debug("block verification time", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("num_txns", len(b.Txns)), zap.Any("duration", time.Since(start)))
	return bvt, nil
}

/*ValidateTransactions - validate the transactions in the block */
func (mc *Chain) ValidateTransactions(ctx context.Context, b *block.Block) error {
	var roundMismatch bool
	var cancel bool
	size := 2000
	numWorkers := len(b.Txns) / size
	if numWorkers*size < len(b.Txns) {
		numWorkers++
	}
	validChannel := make(chan bool, len(b.Txns)/size+1)
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
			err := txn.Validate(ctx)
			if err != nil {
				cancel = true
				Logger.Error("validate transactions", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.String("txn", datastore.ToJSON(txn).String()), zap.Error(err))
				validChannel <- false
				return
			}
			ok, err := b.PrevBlock.ChainHasTransaction(txn)
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
	for start := 0; start < len(b.Txns); start += size {
		end := start + size
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
	return nil
}

/*SignBlock - sign the block and provide the verification ticket */
func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	var bvt = &block.BlockVerificationTicket{}
	bvt.BlockID = b.Hash
	self := node.GetSelfNode(ctx)
	var err error
	bvt.VerifierID = self.GetKey()
	bvt.Signature, err = self.Sign(b.Hash)
	if err != nil {
		return nil, err
	}
	return bvt, nil
}

/*AddVerificationTicket - add a verified ticket to the list of verification tickets of the block */
func (mc *Chain) AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool {
	return b.AddVerificationTicket(bvt)
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	Logger.Info("update finalized block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("lf_round", mc.LatestFinalizedBlock.Round), zap.Int64("current_round", mc.CurrentRound), zap.Float64("weight", b.Weight()), zap.Float64("chain_weight", b.ChainWeight), zap.Int("blocks_size", len(mc.Blocks)), zap.Int("rounds_size", len(mc.rounds)))
	if config.Development() {
		for _, t := range b.Txns {
			if !t.DebugTxn() {
				continue
			}
			Logger.Info("update finalized block (debug transaction)", zap.String("txn", t.Hash), zap.String("block", b.Hash))
		}
	}
	mc.FinalizeBlock(ctx, b)
	mc.SendFinalizedBlock(ctx, b)
	fr := mc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
		mc.DeleteRoundsBelow(ctx, fr.Number)
	}
}

/*FinalizeBlock - finalize the transactions in the block */
func (mc *Chain) FinalizeBlock(ctx context.Context, b *block.Block) error {
	modifiedTxns := make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		modifiedTxns[idx] = txn
	}
	return mc.deleteTxns(modifiedTxns)
}
