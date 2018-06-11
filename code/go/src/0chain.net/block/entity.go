package block

import (
	"context"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/transaction"
	"0chain.net/util"
)

/*UnverifiedBlockBody - used to compute the signature
* This is what is used to verify the correctness of the block & the associated signature
 */
type UnverifiedBlockBody struct {
	datastore.VersionField
	datastore.CreationDateField

	PrevHash                    string                `json:"prev_hash"`
	PrevBlockVerficationTickets []*VerificationTicket `json:"prev_verification_tickets,omitempty"`

	MinerID datastore.Key `json:"miner_id"`
	Round   int64         `json:"round"`
	ChainID datastore.Key `json:"chain_id"`

	// The entire transaction payload to represent full block
	Txns []*transaction.Transaction `json:"transactions,omitempty"`

	MerkleTree []string `json:"merkle_tree,omitempty"`
}

/*Block - data structure that holds the block data */
type Block struct {
	datastore.CollectionIDField
	UnverifiedBlockBody
	VerificationTickets []*VerificationTicket `json:"verification_tickets,omitempty"`

	RoundRank int    `json:"-"` // rank of the block in the round it belongs to
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
	if datastore.IsEmpty(b.MinerID) {
		return common.InvalidRequest("miner id is required")
	}
	hash := b.ComputeHash()
	if b.Hash != hash {
		return common.NewError("incorrect_block_hash", "Block hash doesn't match the merkle tree root")
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
	blockEntityMetadata = datastore.MetadataProvider()
	blockEntityMetadata.Name = "block"
	blockEntityMetadata.Provider = Provider
	blockEntityMetadata.Store = store
	blockEntityMetadata.IDColumnName = "hash"
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

/*AddVerificationTicket - Add a verification ticket to a block
*Assuming this is done single-threaded at least per block
*It's the callers responsibility to decide what to do if this operation is successful
*  - the miner of the block for example will decide if the notarization is received and send it off to others
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

/*GetMerkleTree - return the merkle tree of this block using the transactions as leaf nodes */
func (b *Block) GetMerkleTree() *util.MerkleTree {
	if len(b.MerkleTree) > 0 {
		var mt util.MerkleTree
		mt.SetTree(len(b.Txns), b.MerkleTree)
		return &mt
	}
	var hashables = make([]util.Hashable, len(b.Txns))
	for idx, txn := range b.Txns {
		hashables[idx] = txn
	}
	var mt util.MerkleTree
	mt.ComputeTree(hashables)
	return &mt
}

/*ComputeHash - compute the hash of the block */
func (b *Block) ComputeHash() string {
	mt := b.GetMerkleTree()
	b.MerkleTree = mt.GetTree()
	return mt.GetRoot()
}

/*HashBlock - compute and set the hash of the block */
func (b *Block) HashBlock() {
	b.Hash = b.ComputeHash()
	b.ID = datastore.ToKey(b.Hash)
}
