package miner

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"0chain.net/block"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/transaction"
	"go.uber.org/zap"
)

/*StartRound - start a new round */
func (mc *Chain) StartRound(ctx context.Context, r *round.Round) {
	mc.AddRound(r)
}

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (mc *Chain) GenerateBlock(ctx context.Context, b *block.Block) error {
	clients := make(map[string]*client.Client)
	b.Txns = make([]*transaction.Transaction, mc.BlockSize)
	//TODO: wasting this because []interface{} != []*transaction.Transaction in Go
	etxns := make([]datastore.Entity, mc.BlockSize)
	var idx int32
	self := node.GetSelfNode(ctx)
	if self == nil {
		panic("Invalid setup, could not find the self node")
	}
	b.MinerID = self.ID
	var txnIterHandler = func(ctx context.Context, qe datastore.CollectionEntity) bool {
		txn, ok := qe.(*transaction.Transaction)
		if !ok {
			return true
		}

		if txn.Status != transaction.TXN_STATUS_FREE {
			return true
		}

		txn.Status = transaction.TXN_STATUS_PENDING
		//Setting the score lower so the next time blocks are generated these transactions don't show up at the top
		txn.SetCollectionScore(txn.GetCollectionScore() - 10*60)

		b.Txns[idx] = txn
		etxns[idx] = txn
		b.AddTransaction(txn)
		idx++

		clients[txn.ClientID] = nil

		if idx == mc.BlockSize {
			return false
		}
		return true
	}
	transactionEntityMetadata := datastore.GetEntityMetadata("txn")
	txn := transactionEntityMetadata.Instance().(*transaction.Transaction)
	txn.ChainID = b.ChainID
	collectionName := txn.GetCollectionName()
	//TODO: remove timing code later (or make it applicable to test mode)
	start := time.Now()
	err := transactionEntityMetadata.GetStore().IterateCollection(ctx, transactionEntityMetadata, collectionName, txnIterHandler)
	if err != nil {
		return err
	}
	if idx != mc.BlockSize {
		b.Txns = nil
		return common.NewError("insufficient_txns", "Not sufficient txns to make a block yet\n")
	}

	client.GetClients(ctx, clients)
	Logger.Info("time to assemble block", zap.Any("block", b.Hash), zap.Any("time", time.Since(start)))
	updateTxnsToPending(ctx, etxns)
	Logger.Info("time to assemble + write block", zap.Any("block", b.Hash), zap.Any("time", time.Since(start)))
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)

	//TODO: After the hashblock is done with the txn hashes, the publickey/clientid switch can move right after GetClients
	for _, txn := range b.Txns {
		client := clients[txn.ClientID]
		if client == nil {
			return common.NewError("invalid_client_id", "client id not available")
		}
		txn.PublicKey = client.PublicKey
		txn.ClientID = datastore.EmptyKey
	}
	if err != nil {
		return err
	}
	Logger.Info("time to assemble+write+sign block", zap.Any("block", b.Hash), zap.Any("time", time.Since(start)))
	mc.AddToVerification(ctx, b)
	return nil
}

func updateTxnsToPending(ctx context.Context, txns []datastore.Entity) {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")
	transactionMetadataProvider.GetStore().MultiWrite(ctx, transactionMetadataProvider, txns)
}

/*AddToVerification - Add a block to verify : WARNING: does not support concurrent access for a given round */
func (mc *Chain) AddToVerification(ctx context.Context, b *block.Block) {
	r := mc.GetRound(b.Round)
	if r != nil {
		if r.IsVerificationComplete() {
			return
		}
	} else {
		// TODO: This can happen because
		// 1) This is past round that is no longer applicable - reject it
		// 2) This is a future round we didn't know about yet as our network is slow or something
		// 3) The verify message received before the start round message
		r = datastore.GetEntityMetadata("round").Instance().(*round.Round)
		r.Number = b.Round
		mc.AddRound(r)
	}
	mc.AddBlock(b)
	r.StartVerificationBlockCollection(ctx, mc.CollectBlocksForVerification)
	r.AddBlock(b)
}

/*VerifyBlock - given a set of transaction ids within a block, validate the block */
func (mc *Chain) VerifyBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	err := b.Validate(ctx)
	if err != nil {
		return nil, err
	}
	hashCameWithBlock := b.Hash
	hash := b.ComputeHash()
	if hashCameWithBlock != hash {
		return nil, common.NewError("hash wrong", "The hash of the block is wrong")
	}
	miner := node.GetNode(b.MinerID)
	if miner == nil {
		return nil, common.NewError("unknown_miner", "Do not know this miner")
	}
	var ok bool
	ok, err = miner.Verify(b.Signature, b.Hash)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, common.NewError("signature invalid", "The block wasn't signed correctly")
	}
	size := 2000
	numWorkers := len(b.Txns) / size
	if numWorkers*size < len(b.Txns) {
		numWorkers++
	}
	validChannel := make(chan bool, len(b.Txns)/size+1)
	var cancel bool
	start := time.Now()
	for start := 0; start < len(b.Txns); start += size {
		end := start + size
		if end > len(b.Txns) {
			end = len(b.Txns)
		}
		go validate(ctx, b.Txns[start:end], &cancel, validChannel)
	}
	count := 0
	for result := range validChannel {
		if !result {
			fmt.Printf("Block verification time due to failure:%v\n", time.Since(start))
			return nil, common.NewError("txn_validation_failed", "Transaction validation failed")
		}
		count++
		if count == numWorkers {
			break
		}
	}
	bvt, err := mc.SignBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Block verification time(%v,%v):%v\n", len(b.Txns), numWorkers, time.Since(start))
	return bvt, nil
}

/*SignBlock - sign the block and provide the verification ticket */
func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (*block.BlockVerificationTicket, error) {
	var bvt = &block.BlockVerificationTicket{}
	bvt.BlockID = b.Hash
	self := node.GetSelfNode(ctx)
	if self == nil {
		panic("Invalid setup, could not find the self node")
	}
	var err error
	bvt.VerifierID = self.GetKey()
	bvt.Signature, err = self.Sign(b.Hash)
	if err != nil {
		return nil, err
	}
	return bvt, nil
}

func validate(ctx context.Context, txns []*transaction.Transaction, cancel *bool, validChannel chan<- bool) {
	for _, txn := range txns {
		err := txn.Validate(ctx)
		if err != nil {
			*cancel = true
			validChannel <- false
		}
		if *cancel {
			return
		}
	}
	validChannel <- true
}

/*ProcessVerifiedTicket - once a verified ticket is receiveid, do further processing with it */
func (mc *Chain) ProcessVerifiedTicket(ctx context.Context, r *round.Round, b *block.Block, vt *block.VerificationTicket) {
	if mc.AddVerificationTicket(ctx, b, vt) {
		if mc.IsBlockNotarized(ctx, b) {
			r.Block = b
			r.CancelVerification() // No need for further verification of any blocks
			notarization := datastore.GetEntityMetadata("block_notarization").Instance().(*Notarization)
			notarization.BlockID = b.Hash
			notarization.Round = b.Round
			notarization.VerificationTickets = b.VerificationTickets
			mc.SendNotarization(ctx, notarization)
			if mc.GetRound(r.Number+1) == nil {
				nr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
				nr.Number = r.Number + 1
				nr.RandomSeed = rand.New(rand.NewSource(r.RandomSeed)).Int63()
				go mc.startNewRound(ctx, nr)
				mc.Miners.SendAll(RoundStartSender(nr))
			}
			pr := mc.GetRound(r.Number - 1)
			if pr != nil && pr.Block != nil {
				mc.FinalizeBlock(ctx, pr.Block)
			}
		}
	}
}

/*IsBlockNotarized - Does the given number of signatures means eligible for notraization?
TODO: For now, we just assume more than 50% */
func (mc *Chain) IsBlockNotarized(ctx context.Context, b *block.Block) bool {
	numSignatures := b.GetVerificationTicketsCount()
	if 3*numSignatures >= 2*mc.Miners.Size() {
		return mc.IsCurrentlyWinningBlock(b)
	}
	return false
}

/*IsCurrentlyWinningBlock - Is this currently the winning block for it's round? */
func (mc *Chain) IsCurrentlyWinningBlock(b *block.Block) bool {
	//TODO: Ideally block's round's block should be the best if we are doing that book keeping
	return true
}

/*VerifyTicket - verify the ticket */
func (mc *Chain) VerifyTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) error {
	/* Seems like this is allowed per Dfiniity protocol
	if bvt.VerifierID == b.MinerID {
		return common.InvalidRequest("Self signing not allowed")
	} */
	sender := mc.Miners.GetNode(bvt.VerifierID)
	if sender == nil {
		return common.InvalidRequest("Verifier unknown or not authorized at this time")
	}

	if ok, _ := sender.Verify(bvt.Signature, b.Hash); !ok {
		return common.InvalidRequest("Couldn't verify the signature")
	}
	return nil
}

/*AddVerificationTicket - add a verified ticket to the list of verification tickets of the block */
func (mc *Chain) AddVerificationTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) bool {
	return b.AddVerificationTicket(bvt)
}

/*VerifyNotarization - verify that the notarization is correct */
func (mc *Chain) VerifyNotarization(ctx context.Context, b *block.Block, bvt []*block.VerificationTicket) error {
	if b.Round != 0 && bvt == nil {
		return common.NewError("no_verification_tickets", "No verification tickets for this block")
	}
	// TODO: Logic similar to ReachedNotarization to check the count satisfies (refactor)

	for _, vt := range bvt {
		if err := mc.VerifyTicket(ctx, b, vt); err != nil {
			return err
		}
	}
	return nil
}

/*Finalize - finalize the transactions in the block */
func (mc *Chain) Finalize(ctx context.Context, b *block.Block) error {
	modifiedTxns := make([]datastore.Entity, len(b.Txns))
	for idx, txn := range b.Txns {
		txn.BlockID = b.ID
		txn.Status = transaction.TXN_STATUS_FINALIZED
		modifiedTxns[idx] = txn
	}
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")
	err := transactionMetadataProvider.GetStore().MultiWrite(ctx, transactionMetadataProvider, modifiedTxns)
	if err != nil {
		return err
	}
	return nil
}
