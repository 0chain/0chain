package block

import (
	"context"
	"fmt"
	"time"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/transaction"
)

var BLOCK_SIZE = 250000

/*GenerateBlock - This works on generating a block
* The context should be a background context which can be used to stop this logic if there is a new
* block published while working on this
 */
func (b *Block) GenerateBlock(ctx context.Context) error {
	clients := make(map[string]*client.Client)
	txns := make([]*transaction.Transaction, BLOCK_SIZE)
	b.Txns = &txns
	//TODO: wasting this because we []interface{} != []*transaction.Transaction in Go
	etxns := make([]memorystore.MemoryEntity, BLOCK_SIZE)
	idx := 0
	self := node.GetSelfNode(ctx)
	if self == nil {
		panic("Invalid setup, could not find the self node")
	}
	b.MinerID = self.ID
	b.Round = 0
	var txnIterHandler = func(ctx context.Context, qe memorystore.CollectionEntity) bool {
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

		txns[idx] = txn
		etxns[idx] = txn
		b.AddTransaction(txn)
		idx++

		clients[txn.ClientID] = nil

		if idx == BLOCK_SIZE {
			return false
		}
		return true
	}
	txn := transaction.Provider().(*transaction.Transaction)
	txn.ChainID = b.ChainID
	collectionName := txn.GetCollectionName()
	//TODO: remove timing code later (or make it applicable to test mode)
	start := time.Now()
	err := memorystore.IterateCollection(ctx, collectionName, txnIterHandler, datastore.GetEntityMetadata("txn"))
	if err != nil {
		return err
	}
	if idx != BLOCK_SIZE {
		b.Txns = nil
		return common.NewError("insufficient_txns", "Not sufficient txns to make a block yet\n")
	}

	client.GetClients(ctx, clients)
	fmt.Printf("time to assemble block: %v\n", time.Since(start))
	b.UpdateTxnsToPending(ctx, etxns)
	fmt.Printf("time to assemble + write block: %v\n", time.Since(start))
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)

	//TODO: After the hashblock is done with the txn hashes, the publickey/clientid switch can move right after GetClients
	for _, txn := range txns {
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
	fmt.Printf("time to assemble+write+sign block: %v\n", time.Since(start))

	return nil
}

/*UpdateTxnsToPending - marks all the given transactions to pending */
func (b *Block) UpdateTxnsToPending(ctx context.Context, txns []memorystore.MemoryEntity) {
	memorystore.MultiWrite(ctx, datastore.GetEntityMetadata("txn"), txns)
}

/*VerifyBlock - given a set of transaction ids within a block, validate the block */
func (b *Block) VerifyBlock(ctx context.Context) (bool, error) {
	start := time.Now()
	err := b.Validate(ctx)
	if err != nil {
		return false, err
	}
	hashCameWithBlock := b.Hash
	hash := b.ComputeHash()
	if hashCameWithBlock != hash {
		return false, common.NewError("hash wrong", "The hash of the block is wrong")
	}
	miner := node.GetNode(b.MinerID)
	if miner == nil {
		return false, common.NewError("unknown_miner", "Do not know this miner")
	}
	var ok bool
	ok, err = miner.Verify(b.Signature, b.Hash)
	if err != nil {
		return false, err
	} else if !ok {
		return false, common.NewError("signature invalid", "The block wasn't signed correctly")
	}
	for _, txn := range *b.Txns {
		err = txn.Validate(ctx)
		if err != nil {
			return false, err
		}
	}
	fmt.Printf("Block verification time:%v\n", time.Since(start))
	return true, nil
}

/*Finalize - finalize the transactions in the block */
func (b *Block) Finalize(ctx context.Context) error {
	modifiedTxns := make([]memorystore.MemoryEntity, 0, BLOCK_SIZE)
	for idx, txn := range *b.Txns {
		txn.BlockID = b.ID
		txn.Status = transaction.TXN_STATUS_FINALIZED
		modifiedTxns[idx] = txn
	}
	err := memorystore.MultiWrite(ctx, datastore.GetEntityMetadata("txn"), modifiedTxns)
	if err != nil {
		return err
	}
	return nil
}
