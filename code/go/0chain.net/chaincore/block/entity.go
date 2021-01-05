package block

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

var (
	ErrBlockHashMismatch      = common.NewError("block_hash_mismatch", "Block hash mismatch")
	ErrBlockStateHashMismatch = common.NewError("block_state_hash_mismatch", "Block state hash mismatch")
)

const (
	StateGenerated              = 1
	StateVerificationPending    = iota
	StateVerificationAccepted   = iota
	StateVerificationRejected   = iota
	StateVerifying              = iota
	StateVerificationSuccessful = iota
	StateVerificationFailed     = iota
	StateNotarized              = iota
)

const (
	StatePending    = 0
	StateComputing  = iota
	StateFailed     = iota
	StateSuccessful = iota
	StateSynched    = iota
)

const (
	VerificationPending    = 0
	VerificationSuccessful = iota
	VerificationFailed     = iota
)

/*UnverifiedBlockBody - used to compute the signature
* This is what is used to verify the correctness of the block & the associated signature
 */
type UnverifiedBlockBody struct {
	datastore.VersionField
	datastore.CreationDateField

	LatestFinalizedMagicBlockHash  string                `json:"latest_finalized_magic_block_hash"`
	LatestFinalizedMagicBlockRound int64                 `json:"latest_finalized_magic_block_round"`
	PrevHash                       string                `json:"prev_hash"`
	PrevBlockVerificationTickets   []*VerificationTicket `json:"prev_verification_tickets,omitempty"`

	MinerID           datastore.Key `json:"miner_id"`
	Round             int64         `json:"round"`
	RoundRandomSeed   int64         `json:"round_random_seed"`
	RoundTimeoutCount int           `json:"round_timeout_count"`

	ClientStateHash util.Key `json:"state_hash"`

	// The entire transaction payload to represent full block
	Txns []*transaction.Transaction `json:"transactions,omitempty"`
}

/*Block - data structure that holds the block data */
type Block struct {
	UnverifiedBlockBody
	VerificationTickets []*VerificationTicket `json:"verification_tickets,omitempty"`

	datastore.HashIDField
	Signature string `json:"signature"`

	ChainID     datastore.Key `json:"chain_id"`
	ChainWeight float64       `json:"chain_weight"`
	RoundRank   int           `json:"-"` // rank of the block in the round it belongs to
	PrevBlock   *Block        `json:"-"`

	TxnsMap   map[string]bool `json:"-"`
	mutexTxns sync.RWMutex

	ClientState           util.MerklePatriciaTrieI `json:"-"`
	stateStatus           int8
	stateStatusMutex      *sync.RWMutex `json:"_"`
	StateMutex            *sync.RWMutex `json:"_"`
	blockState            int8
	isNotarized           bool
	ticketsMutex          *sync.RWMutex
	verificationStatus    int
	RunningTxnCount       int64           `json:"running_txn_count"`
	UniqueBlockExtensions map[string]bool `json:"-"`
	*MagicBlock           `json:"magic_block,omitempty"`
}

// NewBlock - create a new empty block
func NewBlock(chainID datastore.Key, round int64) *Block {
	b := datastore.GetEntityMetadata("block").Instance().(*Block)
	b.Round = round
	return b
}

// GetVerificationTickets of the block async safe.
func (b *Block) GetVerificationTickets() (vts []*VerificationTicket) {
	b.ticketsMutex.RLock()
	defer b.ticketsMutex.RUnlock()

	if len(b.VerificationTickets) == 0 {
		return // nil
	}

	vts = make([]*VerificationTicket, 0, len(b.VerificationTickets))
	for _, tk := range b.VerificationTickets {
		vts = append(vts, tk.Copy())
	}

	return
}

// VerificationTicketsSize returns number verification tickets of the Block.
func (b *Block) VerificationTicketsSize() int {
	b.ticketsMutex.RLock()
	defer b.ticketsMutex.RUnlock()

	return len(b.VerificationTickets)
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
		b.mutexTxns.Lock()
		defer b.mutexTxns.Unlock()
		b.TxnsMap = make(map[string]bool, len(b.Txns))
		for _, txn := range b.Txns {
			txn.ComputeProperties()
			b.TxnsMap[txn.Hash] = true
		}
	}
}

/*ComputeProperties - Entity implementation */
func (b *Block) Decode(input []byte) error {
	return json.Unmarshal(input, b)
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

	b.mutexTxns.RLock()
	if b.TxnsMap != nil {
		if len(b.Txns) != len(b.TxnsMap) {
			b.mutexTxns.RUnlock()
			return common.NewError("duplicate_transactions", "Block has duplicate transactions")
		}
	}
	b.mutexTxns.RUnlock()

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

/*GetScore - score for write*/
func (b *Block) GetScore() int64 {
	return b.Round
}

/*Write - store read */
func (b *Block) Write(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Write(ctx, b)
}

/*Delete - store read */
func (b *Block) Delete(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Delete(ctx, b)
}

/*Provider - entity provider for block object */
func Provider() datastore.Entity {
	b := &Block{}
	b.Version = "1.0"
	b.ChainID = datastore.ToKey(config.GetServerChainID())
	b.InitializeCreationDate()
	b.StateMutex = &sync.RWMutex{}
	b.stateStatusMutex = &sync.RWMutex{}
	b.ticketsMutex = &sync.RWMutex{}
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
	SetupBVTEntity()
}

/*SetPreviousBlock - set the previous block of this block */
func (b *Block) SetPreviousBlock(prevBlock *Block) {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()

	b.PrevBlock = prevBlock
	b.PrevHash = prevBlock.Hash
	b.Round = prevBlock.Round + 1
	if len(b.PrevBlockVerificationTickets) == 0 {
		b.PrevBlockVerificationTickets = prevBlock.GetVerificationTickets()
	}
}

/*SetStateDB - set the state from the previous block */
func (b *Block) SetStateDB(prevBlock *Block) {
	var pndb util.NodeDB
	var rootHash util.Key
	if prevBlock.ClientState == nil {
		Logger.Debug("Set state db -- prior state not available")
		if state.Debug() {
			Logger.DPanic("Set state db - prior state not available")
		} else {
			pndb = util.NewMemoryNodeDB()
		}
	} else {
		pndb = prevBlock.ClientState.GetNodeDB()
	}
	rootHash = prevBlock.ClientStateHash
	Logger.Debug("Prev state root", zap.Int64("round", b.Round),
		zap.String("prev_block", prevBlock.Hash),
		zap.String("root", util.ToHex(rootHash)))
	b.CreateState(pndb)
	b.ClientState.SetRoot(rootHash)
}

// InitStateDB - initialize the block's state from the db
// (assuming it's already computed).
func (b *Block) InitStateDB(ndb util.NodeDB) (err error) {
	if _, err = ndb.GetNode(b.ClientStateHash); err != nil {
		b.SetStateStatus(StateFailed)
		return
	}
	b.CreateState(ndb)
	b.ClientState.SetRoot(b.ClientStateHash)
	b.SetStateStatus(StateSuccessful)
	return nil
}

//CreateState - create the state from the prior state db
func (b *Block) CreateState(pndb util.NodeDB) {
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, pndb, false)
	b.ClientState = util.NewMerklePatriciaTrie(ndb, util.Sequence(b.Round))
}

/*AddTransaction - add a transaction to the block */
func (b *Block) AddTransaction(t *transaction.Transaction) {
	t.OutputHash = t.ComputeOutputHash()
}

/*AddVerificationTicket - Add a verification ticket to a block if it's not already present */
func (b *Block) AddVerificationTicket(vt *VerificationTicket) bool {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()
	bvt := b.VerificationTickets
	for _, t := range bvt {
		if datastore.IsEqual(vt.VerifierID, t.VerifierID) {
			return false
		}
	}
	bvt = append(bvt, vt)
	b.VerificationTickets = bvt
	return true
}

/*MergeVerificationTickets - merge the verification tickets with what's already present
* Only appends without modifying the order of exisitng tickets to ensure concurrent marshalling doesn't cause duplicate tickets
 */
func (b *Block) MergeVerificationTickets(vts []*VerificationTicket) {
	unionVerificationTickets := func(alreadyHave []*VerificationTicket, received []*VerificationTicket) []*VerificationTicket {
		if len(alreadyHave) == 0 {
			return received
		}
		if len(received) == 0 {
			return alreadyHave
		}
		alreadyHaveMap := make(map[string]*VerificationTicket, len(alreadyHave))
		for _, t := range alreadyHave {
			alreadyHaveMap[t.VerifierID] = t
		}
		union := make([]*VerificationTicket, len(alreadyHave))
		copy(union, alreadyHave)
		for _, rec := range received {
			if rec == nil {
				Logger.Error("merge verification tickets - null ticket")
				return alreadyHave
			}
			if _, ok := alreadyHaveMap[rec.VerifierID]; !ok {
				union = append(union, rec)
				alreadyHaveMap[rec.VerifierID] = rec
			}
		}
		if len(union) == len(alreadyHave) {
			return alreadyHave
		}
		return union
	}
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()
	b.VerificationTickets = unionVerificationTickets(b.VerificationTickets, vts)
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
	rmt := b.GetReceiptsMerkleTree()
	rMerkleRoot := rmt.GetRoot()
	hashData := b.MinerID + ":" + b.PrevHash + ":" + common.TimeToString(b.CreationDate) + ":" + strconv.FormatInt(b.Round, 10) + ":" + strconv.FormatInt(b.GetRoundRandomSeed(), 10) + ":" + merkleRoot + ":" + rMerkleRoot
	return hashData
}

/*ComputeHash - compute the hash of the block */
func (b *Block) ComputeHash() string {
	hashData := b.getHashData()
	hash := encryption.Hash(hashData)
	return hash
}

/*HashBlock - compute and set the hash of the block */
func (b *Block) HashBlock() {
	b.Hash = b.ComputeHash()
}

/*ComputeTxnMap - organize the transactions into a hashmap for check if the txn exists*/
func (b *Block) ComputeTxnMap() {
	b.mutexTxns.Lock()
	defer b.mutexTxns.Unlock()
	b.TxnsMap = make(map[string]bool, len(b.Txns))
	for _, txn := range b.Txns {
		b.TxnsMap[txn.Hash] = true
	}
}

/*HasTransaction - check if the transaction exists in this block */
func (b *Block) HasTransaction(hash string) bool {
	b.mutexTxns.RLock()
	defer b.mutexTxns.RUnlock()
	_, ok := b.TxnsMap[hash]
	return ok
}

/*GetSummary - get the block summary of this block */
func (b *Block) GetSummary() *BlockSummary {
	bs := datastore.GetEntityMetadata("block_summary").Instance().(*BlockSummary)
	bs.Version = b.Version
	bs.Hash = b.Hash
	bs.MinerID = b.MinerID
	bs.Round = b.Round
	bs.RoundRandomSeed = b.GetRoundRandomSeed()
	bs.CreationDate = b.CreationDate
	bs.MerkleTreeRoot = b.GetMerkleTree().GetRoot()
	bs.ClientStateHash = b.ClientStateHash
	bs.ReceiptMerkleTreeRoot = b.GetReceiptsMerkleTree().GetRoot()
	bs.NumTxns = len(b.Txns)
	bs.MagicBlock = b.MagicBlock
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
	cmap := make(map[string]*client.Client)
	for _, t := range b.Txns {
		if t.PublicKey == "" {
			continue
		}
		if _, ok := cmap[t.PublicKey]; ok {
			continue
		}
		c := client.NewClient()
		c.SetPublicKey(t.PublicKey)
		cmap[t.PublicKey] = c
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
	b.stateStatusMutex.RLock()
	defer b.stateStatusMutex.RUnlock()
	return b.stateStatus
}

/*IsStateComputed - is the state of this block computed? */
func (b *Block) IsStateComputed() bool {
	b.stateStatusMutex.RLock()
	defer b.stateStatusMutex.RUnlock()
	if b.stateStatus >= StateSuccessful {
		return true
	}
	return false
}

/*SetStateStatus - set if the client state is computed or not for the block */
func (b *Block) SetStateStatus(status int8) {
	b.stateStatusMutex.Lock()
	defer b.stateStatusMutex.Unlock()
	b.stateStatus = status
}

/*GetReceiptsMerkleTree - return the merkle tree of this block using the transactions as leaf nodes */
func (b *Block) GetReceiptsMerkleTree() *util.MerkleTree {
	var hashables = make([]util.Hashable, len(b.Txns))
	for idx, txn := range b.Txns {
		hashables[idx] = transaction.NewTransactionReceipt(txn)
	}
	var mt util.MerkleTree
	mt.ComputeTree(hashables)
	return &mt
}

//GetTransaction - get the transaction from the block
func (b *Block) GetTransaction(hash string) *transaction.Transaction {
	for _, txn := range b.Txns {
		if txn.GetKey() == hash {
			return txn
		}
	}
	return nil
}

//SetBlockNotarized - set the block as notarized
func (b *Block) SetBlockNotarized() {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()

	b.isNotarized = true
}

//IsBlockNotarized - is block notarized?
func (b *Block) IsBlockNotarized() bool {
	b.ticketsMutex.RLock()
	defer b.ticketsMutex.RUnlock()

	return b.isNotarized
}

/*SetVerificationStatus - set the verification status of the block by this node */
func (b *Block) SetVerificationStatus(status int) {
	b.verificationStatus = status
}

/*GetVerificationStatus - get the verification status of the block */
func (b *Block) GetVerificationStatus() int {
	return b.verificationStatus
}

/*UnknownTickets - compute the list of unknown tickets from a given set of tickets */
func (b *Block) UnknownTickets(vts []*VerificationTicket) []*VerificationTicket {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()
	ticketsMap := make(map[string]*VerificationTicket, len(b.VerificationTickets))
	for _, t := range b.VerificationTickets {
		ticketsMap[t.VerifierID] = t
	}
	var newTickets []*VerificationTicket
	for _, t := range vts {
		if t == nil {
			Logger.Error("unknown tickets - null ticket")
			return nil
		}
		if _, ok := ticketsMap[t.VerifierID]; !ok {
			newTickets = append(newTickets, t)
			ticketsMap[t.VerifierID] = t
		}
	}
	return newTickets
}

// AddUniqueBlockExtension - add unique block extensions.
func (b *Block) AddUniqueBlockExtension(eb *Block) {
	//TODO: We need to compare for view change and add the eb.MinerID only if he was in the view that b belongs to
	if b.UniqueBlockExtensions == nil {
		b.UniqueBlockExtensions = make(map[string]bool)
	}
	b.UniqueBlockExtensions[eb.MinerID] = true
}

// DoReadLock - implement ReadLockable interface.
func (b *Block) DoReadLock() {
	b.ticketsMutex.RLock()
}

// DoReadUnlock - implement ReadLockable interface.
func (b *Block) DoReadUnlock() {
	b.ticketsMutex.RUnlock()
}

// GetPrevBlockVerificationTickets returns
// verification tickets of previous Block.
func (b *Block) GetPrevBlockVerificationTickets() (pbvts []*VerificationTicket) {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()

	if len(b.PrevBlockVerificationTickets) == 0 {
		return // nil
	}

	pbvts = make([]*VerificationTicket, 0, len(b.PrevBlockVerificationTickets))
	for _, tk := range b.PrevBlockVerificationTickets {
		pbvts = append(pbvts, tk.Copy())
	}

	return
}

// PrevBlockVerificationTicketsSize returns number of
// verification tickets of previous Block.
func (b *Block) PrevBlockVerificationTicketsSize() int {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()

	return len(b.PrevBlockVerificationTickets)
}

// SetPrevBlockVerificationTickets - set previous block verification tickets.
func (b *Block) SetPrevBlockVerificationTickets(bvt []*VerificationTicket) {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()
	b.PrevBlockVerificationTickets = bvt
}

// SetRoundRandomSeed - set the random seed.
func (u *UnverifiedBlockBody) SetRoundRandomSeed(seed int64) {
	atomic.StoreInt64(&u.RoundRandomSeed, seed)
}

// GetRoundRandomSeed - returns the random seed of the round.
func (u *UnverifiedBlockBody) GetRoundRandomSeed() int64 {
	return atomic.LoadInt64(&u.RoundRandomSeed)
}
