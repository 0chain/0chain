package block

import (
	"context"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/transaction"
)

/*Block - data structure that holds the block data*/
type Block struct {
	datastore.CollectionIDField
	datastore.CreationDateField
	Hash      string  `json:"hash"`
	PrevHash  string  `json:"prev_hash"`
	Signature string  `json:"signature"`
	MinerID   string  `json:"miner_id"`
	Round     int64   `json:"round"`
	ChainID   string  `json:"chain_id"`
	Weight    float64 `json:"weight"`
	Txns      []interface{}
	PrevBlock *Block
}

/*GetEntityName - implementing the interface */
func (b *Block) GetEntityName() string {
	return "block"
}

/*Validate - implementing the interface */
func (b *Block) Validate(ctx context.Context) error {
	if b.ID == "" {
		if b.Hash == "" {
			return common.InvalidRequest("hash required for block")
		}
		b.ID = b.Hash
	}
	if b.ID != b.Hash {
		return common.NewError("id_hash_mismatch", "ID and Hash don't match")
	}
	if b.ID == "" {
		return common.InvalidRequest("block id is required")
	}
	if b.MinerID == "" {
		return common.InvalidRequest("miner id is required")
	}
	return nil
}

/*Read - datastore read */
func (b *Block) Read(ctx context.Context, key string) error {
	return datastore.Read(ctx, key, b)
}

/*Write - datastore read */
func (b *Block) Write(ctx context.Context) error {
	return datastore.Write(ctx, b)
}

/*Delete - datastore read */
func (b *Block) Delete(ctx context.Context) error {
	return datastore.Delete(ctx, b)
}

var blockEntityCollection = &datastore.EntityCollection{CollectionName: "collection.block", CollectionSize: 1000, CollectionDuration: time.Hour}

/*GetCollectionName - override GetCollectionName to provide queues partitioned by ChainID */
func (b *Block) GetCollectionName() string {
	return blockEntityCollection.GetCollectionName(b.ChainID)
}

/*Provider - entity provider for block object */
func Provider() interface{} {
	b := &Block{}
	b.EntityCollection = blockEntityCollection
	b.InitializeCreationDate()
	return b
}

/*GetPreviousBlock - returns the previous block */
func (b *Block) GetPreviousBlock() *Block {
	if b.PrevBlock != nil {
		return b.PrevBlock
	}
	// TODO: Query from the datastore and ensure the b.Txns array is populated
	return nil

}

/*GetWeight - Get the weight/score of this block */
func (b *Block) GetWeight() float64 {
	return b.Weight
}

/*AddTransaction - add a transaction to the block */
func (b *Block) AddTransaction(t *transaction.Transaction) {
	b.Txns = append(b.Txns, t)
	b.Weight += t.GetWeight()
}
