package block

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/transaction"
)

/*UnverifiedBlockBody - used to compute the signature
* This is what is used to verify the correctness of the block & the associated signature
 */
type UnverifiedBlockBody struct {
	datastore.VersionField
	datastore.CreationDateField

	PrevHash                    string                `json:"prev_hash"`
	PrevBlockVerficationTickets []*VerificationTicket `json:"prev_verification_tickets,omitempty"`

	MinerID datastore.Key `json:"miner_id"` // TODO: Is miner_id & node_id same?
	Round   int64         `json:"round"`
	ChainID datastore.Key `json:"chain_id"`

	// We only need either Txns or TxnHashes but not both
	// The entire transaction payload to represent full block
	Txns []*transaction.Transaction `json:"transactions,omitempty"`

	// Just the hashes of the entire transaction payload to repesent a compact block
	TxnHashes []string `json:"transaction_hashes,omitempty"`
}

/*Block - data structure that holds the block data */
type Block struct {
	datastore.CollectionIDField
	UnverifiedBlockBody
	VerificationTickets []*VerificationTicket `json:"verification_tickets,omitempty"`

	Hash      string `json:"hash"`
	Signature string `json:"signature"`

	PrevBlock *Block `json:"-"`
}

var blockEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (b *Block) GetEntityMetadata() datastore.EntityMetadata {
	return blockEntityMetadata
}

/*ComputeProperties - Entity implementation */
func (b *Block) ComputeProperties() {
	if b.Hash != "" {
		b.ID = datastore.ToKey(b.Hash)
	}
	if datastore.IsEmpty(b.ChainID) {
		b.ChainID = datastore.ToKey(config.GetMainChainID())
	}
	if b.Txns != nil {
		for _, txn := range b.Txns {
			txn.ComputeProperties()
		}
	}
}

/*Validate - implementing the interface */
func (b *Block) Validate(ctx context.Context) error {
	err := config.ValidChain(datastore.ToString(b.ChainID))
	if err != nil {
		return err
	}
	if datastore.IsEmpty(b.ID) {
		if b.Hash == "" {
			return common.InvalidRequest("hash required for block")
		}
	}
	if b.ID != datastore.ToKey(b.Hash) {
		return common.NewError("id_hash_mismatch", "ID and Hash don't match")
	}
	if datastore.IsEmpty(b.ID) {
		return common.InvalidRequest("block id is required")
	}
	if datastore.IsEmpty(b.MinerID) {
		return common.InvalidRequest("miner id is required")
	}
	return nil
}

/*Read - store read */
func (b *Block) Read(ctx context.Context, key datastore.Key) error {
	return b.GetEntityMetadata().GetStore().Read(ctx, key, b)
}

/*Write - store read */
func (b *Block) Write(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Write(ctx, b)
}

/*Delete - store read */
func (b *Block) Delete(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Delete(ctx, b)
}

var blockEntityCollection *datastore.EntityCollection

/*GetCollectionName - override GetCollectionName to provide queues partitioned by ChainID */
func (b *Block) GetCollectionName() string {
	return blockEntityCollection.GetCollectionName(b.ChainID)
}

/*Provider - entity provider for block object */
func Provider() datastore.Entity {
	b := &Block{}
	b.Version = "1.0"
	b.PrevBlockVerficationTickets = make([]*VerificationTicket, 0, 1)
	b.EntityCollection = blockEntityCollection
	b.InitializeCreationDate()
	return b
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	blockEntityMetadata = &datastore.EntityMetadataImpl{Name: "block", Provider: Provider, Store: store}
	datastore.RegisterEntityMetadata("block", blockEntityMetadata)
	blockEntityCollection = &datastore.EntityCollection{CollectionName: "collection.block", CollectionSize: 1000, CollectionDuration: time.Hour}
	SetupBVTEntity()
}

/*SetPreviousBlock - set the previous block of this block */
func (b *Block) SetPreviousBlock(prevBlock *Block) {
	b.PrevBlock = prevBlock
	b.PrevHash = prevBlock.Hash
	b.Round = prevBlock.Round + 1
	b.PrevBlockVerficationTickets = prevBlock.VerificationTickets
}

/*GetPreviousBlock - returns the previous block */
func (b *Block) GetPreviousBlock() *Block {
	if b.PrevBlock != nil {
		return b.PrevBlock
	}
	// TODO: Query from the store and ensure the b.Txns array is populated
	return nil

}

/*AddTransaction - add a transaction to the block */
func (b *Block) AddTransaction(t *transaction.Transaction) {
	// For now this does nothign. May be we don't need. Txn can't influence the weight of the block, or else,
	// everyone will try to maximize the block which is not good
}

/*CompactBlock - Get rid of transaction objects but ensure txn hashes are stored */
func (b *Block) CompactBlock() {
	if b.Txns == nil {
		return
	}
	if b.TxnHashes == nil {
		b.TxnHashes = make([]string, len(b.Txns))
		for idx, txn := range b.Txns {
			b.TxnHashes[idx] = txn.Hash
		}
	}
	b.Txns = nil
}

/*ExpandBlock - Given a block with txn hashes, load up all the txns
* This is a very expensive operation - use it wisely
 */
func (b *Block) ExpandBlock(ctx context.Context) {
	if b.TxnHashes == nil {
		return
	}
	if b.Txns == nil {
		b.Txns = make([]*transaction.Transaction, len(b.TxnHashes))
		// TODO: Block loading for miners has to happen from store
		// Block loading for sharders has to happen from persistence layer
	}
}

/*AddVerificationTicket - Add a verification ticket to a block
*Assuming this is done single-threaded at least per block
*It's the callers responsibility to decide what to do if this operation is successful
*  - the miner of the block for example will decide if the consensus is reached and send it off to others
 */
func (b *Block) AddVerificationTicket(vt *VerificationTicket) bool {
	if b.VerificationTickets != nil {
		for _, ivt := range b.VerificationTickets {
			if datastore.IsEqual(vt.VerifierID, ivt.VerifierID) {
				return false
			}
		}
	}
	if b.VerificationTickets == nil {
		b.VerificationTickets = make([]*VerificationTicket, 0, 1)
	}
	b.VerificationTickets = append(b.VerificationTickets, vt)
	return true
}

/*GetVerificationTicketsCount - get the number of verification tickets for the block */
func (b *Block) GetVerificationTicketsCount() int {
	if b.VerificationTickets == nil {
		return 0
	}
	return len(b.VerificationTickets)
}

/*ComputeHash - compute the hash of the block */
func (b *Block) ComputeHash() string {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(b.UnverifiedBlockBody)
	return encryption.Hash(buf.String())
}

/*HashBlock - compute and set the hash of the block */
func (b *Block) HashBlock() {
	b.Hash = b.ComputeHash()
	b.ID = datastore.ToKey(b.Hash)
}

/*GetHash - get the hash of the block */
func (b *Block) GetHash() string {
	return b.Hash
}
