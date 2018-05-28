package block

import (
	"context"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/node"
	"0chain.net/transaction"
)

/*GenesisBlockHash - block of 0chain.net main chain */
var GenesisBlockHash = "ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae4" //TODO

/*UnverifiedBlockBody - used to compute the signature
* This is what is used to verify the correctness of the block & the associated signature
 */
type UnverifiedBlockBody struct {
	PrevHash                    string                `json:"prev_hash"`
	PrevBlockVerficationTickets []*VerificationTicket `json:"prev_verification_tickets"`

	MinerID datastore.Key `json:"miner_id"` // TODO: Is miner_id & node_id same?
	Round   int64         `json:"round"`
	ChainID datastore.Key `json:"chain_id"`

	// TODO: Float can mess up signature due to precision
	// Also, do we need Weight as part of the Unverified Block Body? Who would care about this?
	Weight float64 `json:"weight"`

	// We only need either Txns or TxnHashes but not both
	// The entire transaction payload to represent full block
	Txns *[]*transaction.Transaction `json:"transactions,omitempty"`

	// Just the hashes of the entire transaction payload to repesent a compact block
	TxnHashes *[]string `json:"transaction_hashes,omitempty"`
}

/*VerifiedBlockBody - block body with verification tickets attached to it
*This is what goes to the sharder once the block reached consensus
 */
type VerifiedBlockBody struct {
	UnverifiedBlockBody
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

/*ComputeProperties - Entity implementation */
func (b *Block) ComputeProperties() {
	if b.Hash != "" {
		b.ID = datastore.ToKey(b.Hash)
	}
	if datastore.IsEmpty(b.ChainID) {
		b.ChainID = datastore.ToKey(config.GetMainChainID())
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

/*Read - datastore read */
func (b *Block) Read(ctx context.Context, key datastore.Key) error {
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

/*CompactBlock - Get rid of transaction objects but ensure txn hashes are stored */
func (b *Block) CompactBlock() {
	if b.Txns == nil {
		return
	}
	if b.TxnHashes == nil {
		hashes := make([]string, len(*b.Txns))
		for idx, txn := range *b.Txns {
			hashes[idx] = txn.Hash
		}
		b.TxnHashes = &hashes
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
		txns := make([]*transaction.Transaction, len(*b.TxnHashes))
		// TODO: Block loading for miners has to happen from datastore
		// Block loading for sharders has to happen from persistence layer
		b.Txns = &txns
	}
}

/*AddVerificationTicket - Add a verification ticket to a block
*Assuming this is done single-threaded at least per block
*It's the callers responsibility to decide what to do if this operation is successful
*  - the miner of the block for example will decide if the consensus is reached and send it off to others
 */
func (b *Block) AddVerificationTicket(vt *VerificationTicket) bool {
	for _, ivt := range b.VerificationTickets {
		if datastore.IsEqual(vt.VerifierID, ivt.VerifierID) {
			return false
		}
	}
	//TODO: Assuming verifier_id is same as the node_id
	nd := node.GetNode(datastore.ToString(vt.VerifierID))
	// We don't have the verifier information
	if nd == nil {
		// TODO: If I am the miner of this block, I better try to do some work and get this verifier data
		return false
	}
	if ok, _ := nd.Verify(vt.Signature, b.Signature); !ok {
		return false
	}
	b.VerificationTickets = append(b.VerificationTickets, vt)
	return true
}
