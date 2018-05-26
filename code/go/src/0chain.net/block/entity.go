package block

import (
	"context"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/transaction"
)

/*GenesisBlockHash - block of 0chain.net main chain */
var GenesisBlockHash = "ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae4" //TODO

/*VerificationTicket - verification ticket for the block */
type VerificationTicket struct {
	VerifierID string `json:"verifier_id"`
	Signature  string `json:"signature"`
}

/*BlockBody - used to compute the signature
* This is what is used to verify the correctness of the block & the associated signature
 */
type BlockBody struct {
	PrevHash                    string                `json:"prev_hash"`
	PrevBlockVerficationTickets []*VerificationTicket `json:"prev_verification_tickets"`

	MinerID string  `json:"miner_id"` // TODO: Is miner_id & node_id same?
	Round   int64   `json:"round"`
	ChainID string  `json:"chain_id"`
	Weight  float64 `json:"weight"`
	Txns    []*transaction.Transaction
}

/*VerifiedBlockBody - block body with verification tickets attached to it
*This is what goes to the sharder once the block reached consensus
 */
type VerifiedBlockBody struct {
	BlockBody
	Hash                string                `json:"hash"`
	Signature           string                `json:"signature"`
	VerificationTickets []*VerificationTicket `json:"verification_tickets"`
}

/*Block - data structure that holds the block data*/
type Block struct {
	VerifiedBlockBody
	datastore.CollectionIDField
	datastore.CreationDateField
	PrevBlock *Block `json:"-"`
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

var blockEntityCollection *datastore.EntityCollection

/*GetCollectionName - override GetCollectionName to provide queues partitioned by ChainID */
func (b *Block) GetCollectionName() string {
	return blockEntityCollection.GetCollectionName(b.ChainID)
}

/*Provider - entity provider for block object */
func Provider() interface{} {
	b := &Block{}
	b.PrevBlockVerficationTickets = make([]*VerificationTicket, 0, 1)
	b.VerificationTickets = make([]*VerificationTicket, 0, 1)

	b.EntityCollection = blockEntityCollection
	b.InitializeCreationDate()
	return b
}

/*SetupEntity - setup the entity */
func SetupEntity() {
	datastore.RegisterEntityProvider("block", Provider)
	blockEntityCollection = &datastore.EntityCollection{CollectionName: "collection.block", CollectionSize: 1000, CollectionDuration: time.Hour}
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
	b.Weight += t.GetWeight()
}
