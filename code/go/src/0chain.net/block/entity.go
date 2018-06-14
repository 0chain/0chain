package block

import (
	"context"
	"time"

	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/transaction"
	"0chain.net/util"
	"go.uber.org/zap"
)

/*UnverifiedBlockBody - used to compute the signature
* This is what is used to verify the correctness of the block & the associated signature
 */
type UnverifiedBlockBody struct {
	datastore.VersionField
	datastore.CreationDateField

	MagicBlockHash              string                `json:"magic_block_hash"`
	PrevHash                    string                `json:"prev_hash"`
	PrevBlockVerficationTickets []*VerificationTicket `json:"prev_verification_tickets,omitempty"`

	MinerID         datastore.Key `json:"miner_id"`
	Round           int64         `json:"round"`
	RoundRandomSeed int64         `json:"round_random_seed"`

	// The entire transaction payload to represent full block
	Txns []*transaction.Transaction `json:"transactions,omitempty"`
}

/*Block - data structure that holds the block data */
type Block struct {
	datastore.CollectionIDField
	UnverifiedBlockBody
	VerificationTickets []*VerificationTicket `json:"verification_tickets,omitempty"`

	Hash      string `json:"hash"`
	Signature string `json:"signature"`

	ChainID   datastore.Key `json:"chain_id"` // TODO: Do we need chain id at all?
	RoundRank int           `json:"-"`        // rank of the block in the round it belongs to
	PrevBlock *Block        `json:"-"`

	//TODO: May be this should be replaced with a bloom filter & check against sorted txns
	TxnsMap map[string]bool `json:"-"`
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
	// For now this does nothing. May be we don't need. Txn can't influence the weight of the block,
	// or else, everyone will try to maximize the block which is not good
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
	return mt.GetRoot()
}

/*HashBlock - compute and set the hash of the block */
func (b *Block) HashBlock() {
	b.Hash = b.ComputeHash()
	b.ID = datastore.ToKey(b.Hash)
}

/*ComputeTxnMap - organize the transactions into a hashmap for check if the txn exists*/
func (b *Block) ComputeTxnMap() {
	b.TxnsMap = make(map[string]bool, len(b.Txns))
	for _, txn := range b.Txns {
		b.TxnsMap[txn.Hash] = true
	}
}

/*HasTransaction - check if the transaction exists in this block */
func (b *Block) HasTransaction(hash string) bool {
	_, ok := b.TxnsMap[hash]
	return ok
}

/*ChainHasTransaction - indicates if this chain has the transaction */
func (b *Block) ChainHasTransaction(txn *transaction.Transaction) (bool, error) {
	for blk := b; blk != nil; blk = blk.PrevBlock {
		if blk.Round == 0 {
			return false, nil
		}
		if blk.HasTransaction(txn.Hash) {
			return true, nil
		}
		if blk.CreationDate < txn.CreationDate {
			return false, nil
		}
	}
	return false, common.NewError("insufficient_chain", "Chain length not sufficient to confirm the presence of this transaction")
}

/*ValidateTransactions - validate the transactions in the block */
func (b *Block) ValidateTransactions(ctx context.Context) error {
	validate := func(ctx context.Context, txns []*transaction.Transaction, cancel *bool, validChannel chan<- bool) {
		for _, txn := range txns {
			err := txn.Validate(ctx)
			if err != nil {
				*cancel = true
				validChannel <- false
			}
			ok, err := b.PrevBlock.ChainHasTransaction(txn)
			if ok || err != nil {
				if err != nil {
					Logger.Error("validation transactions: chain has transactions", zap.Any("round", b.Round), zap.Any("block", b), zap.Error(err))
				}
				return
			}
			if *cancel {
				return
			}
		}
		validChannel <- true
	}

	size := 2000
	numWorkers := len(b.Txns) / size
	if numWorkers*size < len(b.Txns) {
		numWorkers++
	}
	validChannel := make(chan bool, len(b.Txns)/size+1)
	var cancel bool
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
			return common.NewError("txn_validation_failed", "Transaction validation failed")
		}
		count++
		if count == numWorkers {
			break
		}
	}
	return nil
}
