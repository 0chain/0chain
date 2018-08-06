package block

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/transaction"
	"0chain.net/util"
	"go.uber.org/zap"
)

const (
	StateGenerated              = 10
	StateVerificationPending    = 20
	StateVerificationAccepted   = 30
	StateVerificationRejected   = 40
	StateVerifying              = 50
	StateVerificationSuccessful = 60
	StateVerificationFailed     = 70
	StateNotarized              = 80
)

const (
	StatePending    = 0
	StateComputing  = 10
	StateFailed     = 20
	StateSuccessful = 30
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

	ClientStateHash util.Key `json:"state_hash"`

	// The entire transaction payload to represent full block
	Txns []*transaction.Transaction `json:"transactions,omitempty"`
}

/*Block - data structure that holds the block data */
type Block struct {
	datastore.CollectionMemberField
	UnverifiedBlockBody
	VerificationTickets []*VerificationTicket `json:"verification_tickets,omitempty"`

	datastore.HashIDField
	Signature string `json:"signature"`

	ChainID     datastore.Key `json:"chain_id"` // TODO: Do we need chain id at all?
	ChainWeight float64       `json:"chain_weight"`
	RoundRank   int           `json:"-"` // rank of the block in the round it belongs to
	PrevBlock   *Block        `json:"-"`

	//TODO: May be this should be replaced with a bloom filter & check against sorted txns
	TxnsMap map[string]bool `json:"-"`

	ClientState util.MerklePatriciaTrieI `json:"-"`
	StateStatus int8
	StateMutex  *sync.Mutex
	blockState  int8
}

var blockEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (b *Block) GetEntityMetadata() datastore.EntityMetadata {
	return blockEntityMetadata
}

/*ComputeProperties - Entity implementation */
func (b *Block) ComputeProperties() {
	if datastore.IsEmpty(b.ChainID) {
		b.ChainID = datastore.ToKey(config.GetServerChainID())
	}
	if b.Txns != nil {
		b.TxnsMap = make(map[string]bool, len(b.Txns))
		for _, txn := range b.Txns {
			txn.ComputeProperties()
			b.TxnsMap[txn.Hash] = true
		}
	}
}

/*Validate - implementing the interface */
func (b *Block) Validate(ctx context.Context) error {
	err := config.ValidChain(datastore.ToString(b.ChainID))
	if err != nil {
		return err
	}
	if b.Hash == "" {
		return common.InvalidRequest("hash required for block")
	}
	if datastore.IsEmpty(b.MinerID) {
		return common.InvalidRequest("miner id is required")
	}
	miner := node.GetNode(b.MinerID)
	if miner == nil {
		return common.NewError("unknown_miner", "Do not know this miner")
	}
	if b.ChainWeight > float64(b.Round) {
		return common.NewError("chain_weight_gt_round", "Chain weight can't be greater than the block round")
	}

	hash := b.ComputeHash()
	if b.Hash != hash {
		return common.NewError("incorrect_block_hash", fmt.Sprintf("computed block hash doesn't match with the hash of the block: %v: %v: %v", b.Hash, hash, b.getHashData()))
	}
	var ok bool
	ok, err = miner.Verify(b.Signature, b.Hash)
	if err != nil {
		return err
	} else if !ok {
		return common.NewError("signature invalid", "The block wasn't signed correctly")
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
	b.ChainID = datastore.ToKey(config.GetServerChainID())
	b.InitializeCreationDate()
	b.StateMutex = &sync.Mutex{}
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
	b.SetClientStateDB(prevBlock)
}

/*SetClientStateDB - set the client state from the previous block */
func (b *Block) SetClientStateDB(prevBlock *Block) {
	var pndb util.NodeDB
	var rootHash util.Key
	if prevBlock != nil && prevBlock.ClientState != nil {
		pndb = prevBlock.ClientState.GetNodeDB()
		if pndb == nil {
			Logger.Info("missing pndb")
		}
		rootHash = prevBlock.ClientStateHash
		Logger.Debug("prev state root", zap.Int64("round", b.Round), zap.String("prev_block", prevBlock.Hash), zap.String("root", util.ToHex(rootHash)))
	} else {
		Logger.Info("TODO: state sync", zap.Int64("round", b.Round))
		pndb = util.NewMemoryNodeDB() // TODO: state sync
	}
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, pndb, false)
	b.ClientState = util.NewMerklePatriciaTrie(ndb)
	b.ClientState.SetRoot(rootHash)
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

/*MergeVerificationTickets - merge the verification tickets with what's already there */
func (b *Block) MergeVerificationTickets(vts []*VerificationTicket) {
	if b.VerificationTickets == nil || len(b.VerificationTickets) == 0 {
		b.VerificationTickets = vts
		return
	}
	tickets, blockTickets := vts, b.VerificationTickets
	if len(blockTickets) > len(tickets) {
		tickets, blockTickets = blockTickets, tickets
	}

	sort.Slice(tickets, func(i, j int) bool { return tickets[i].VerifierID < tickets[j].VerifierID })
	ticketsLen := len(tickets)
	for _, ticket := range blockTickets {
		ticketIndex := sort.Search(ticketsLen, func(i int) bool { return tickets[i].VerifierID >= ticket.VerifierID })
		if ticketIndex < ticketsLen && ticket.VerifierID == tickets[ticketIndex].VerifierID { // present in both
			continue
		}
		tickets = append(tickets, ticket)
	}
	b.VerificationTickets = tickets
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

func (b *Block) getHashData() string {
	mt := b.GetMerkleTree()
	merkleRoot := mt.GetRoot()
	hashData := fmt.Sprintf("%v:%v:%v:%v", b.CreationDate, b.Round, b.RoundRandomSeed, merkleRoot)
	return hashData
}

/*ComputeHash - compute the hash of the block */
func (b *Block) ComputeHash() string {
	hashData := b.getHashData()
	hash := encryption.Hash(hashData)
	//Logger.Debug("hash of the block", zap.String("hash", hash), zap.String("hashdata", hashData))
	return hash
}

/*HashBlock - compute and set the hash of the block */
func (b *Block) HashBlock() {
	b.Hash = b.ComputeHash()
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
	var pb = b
	for cb := b; cb != nil; pb, cb = cb, cb.PrevBlock {
		if cb.Round == 0 {
			return false, nil
		}
		if cb.HasTransaction(txn.Hash) {
			return true, nil
		}
		if cb.CreationDate < txn.CreationDate {
			return false, nil
		}
	}
	if false {
		Logger.Debug("chain has txn", zap.Int64("round", b.Round), zap.Int64("upto_round", pb.Round), zap.Any("txn_ts", txn.CreationDate), zap.Any("upto_block_ts", pb.CreationDate))
	}
	return false, common.NewError("insufficient_chain", "Chain length not sufficient to confirm the presence of this transaction")
}

/*GetSummary - get the block summary of this block */
func (b *Block) GetSummary() *BlockSummary {
	bs := datastore.GetEntityMetadata("block_summary").Instance().(*BlockSummary)
	bs.Version = b.Version
	bs.Hash = b.Hash
	bs.Round = b.Round
	bs.RoundRandomSeed = b.RoundRandomSeed
	bs.CreationDate = b.CreationDate
	bs.MerkleTreeRoot = b.GetMerkleTree().GetRoot()
	return bs
}

/*Weight - weight of the block */
func (b *Block) Weight() float64 {
	var w = 1.0
	for i := 0; i < b.RoundRank; i++ {
		w /= 2
	}
	return w
}

/*ComputeChainWeight - compute the weight of the chain up to this block */
func (b *Block) ComputeChainWeight() {
	if b.PrevBlock == nil {
		b.ChainWeight = b.Weight()
	} else {
		b.ChainWeight = b.PrevBlock.ChainWeight + b.Weight()
	}
}

/*Clear - clear the block */
func (b *Block) Clear() {
	b.PrevBlock = nil
	b.PrevBlockVerficationTickets = nil
	b.VerificationTickets = nil
	b.Txns = nil
	b.TxnsMap = nil
	b.StateMutex = nil
}

/*SetBlockState - set the state of the block */
func (b *Block) SetBlockState(blockState int8) {
	b.blockState = blockState
}

/*GetBlockState - get the state of the block */
func (b *Block) GetBlockState() int8 {
	return b.blockState
}

/*GetClients - get all the clients of this block */
func (b *Block) GetClients() []*client.Client {
	clientMetadataProvider := datastore.GetEntityMetadata("client")
	cmap := make(map[string]*client.Client)
	for _, t := range b.Txns {
		if t.PublicKey == "" {
			continue
		}
		if _, ok := cmap[t.PublicKey]; ok {
			continue
		}
		c := clientMetadataProvider.Instance().(*client.Client)
		c.SetPublicKey(t.PublicKey)
		cmap[t.PublicKey] = c
		t.PublicKey = ""
	}
	clients := make([]*client.Client, len(cmap))
	idx := 0
	for _, c := range cmap {
		clients[idx] = c
		idx++
	}
	return clients
}

/*GetStateStatus - indicates if the client state of the block is computed */
func (b *Block) GetStateStatus() int8 {
	return b.StateStatus
}

/*IsStateComputed - is the state of this block computed? */
func (b *Block) IsStateComputed() bool {
	if b.StateStatus == StateSuccessful {
		return true
	}
	//TODO: the following is temporary
	if b.StateStatus == StateFailed {
		return true
	}
	return false
}

/*SetStateStatus - set if the client state is computed or not for the block */
func (b *Block) SetStateStatus(status int8) {
	b.StateStatus = status
}
