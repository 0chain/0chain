package block

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/config"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
)

const (
	//PreviousBlockUnavailable - to indicate an error condition when the previous
	// block of a given block is not available.
	PreviousBlockUnavailable = "previous_block_unavailable"
	//StateMismatch - indicate if there is a mismatch between computed state and received state of a block
	StateMismatch = "state_mismatch"
)

var (

	//StateSaveTimer - a metric that tracks the time it takes to save the state
	StateSaveTimer metrics.Timer

	//StateChangeSizeMetric - a metric that tracks how many state nodes are changing with each block
	StateChangeSizeMetric metrics.Histogram
)

var (
	ErrBlockHashMismatch      = common.NewError("block_hash_mismatch", "block hash mismatch")
	ErrBlockStateHashMismatch = common.NewError("block_state_hash_mismatch", "block state hash mismatch")

	ErrPreviousStateUnavailable = common.NewError("prev_state_unavailable", "Previous state not available")
	ErrPreviousStateNotComputed = common.NewError("prev_state_not_computed", "Previous state not computed")
	ErrCostTooBig               = common.NewError("cost_too_big", "Block cost is too big")

	// ErrPreviousBlockUnavailable - error for previous block is not available.
	ErrPreviousBlockUnavailable = common.NewError(PreviousBlockUnavailable,
		"Previous block is not available")

	ErrStateMismatch = common.NewError(StateMismatch, "Computed state hash doesn't match with the state hash of the block")
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
	StateCancelled  = iota
	StateFailed     = iota
	StateSuccessful = iota
	StateSynched    = iota
)

const (
	VerificationPending    = 0
	VerificationSuccessful = iota
	VerificationFailed     = iota
)

func init() {
	StateSaveTimer = metrics.GetOrRegisterTimer("state_save_timer", nil)
	StateChangeSizeMetric = metrics.NewHistogram(metrics.NewUniformSample(1024))
}

// UnverifiedBlockBody - used to compute the signature
// This is what is used to verify the correctness of the block & the associated signature
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

// SetRoundRandomSeed - set the random seed.
func (u *UnverifiedBlockBody) SetRoundRandomSeed(seed int64) {
	atomic.StoreInt64(&u.RoundRandomSeed, seed)
}

// GetRoundRandomSeed - returns the random seed of the round.
func (u *UnverifiedBlockBody) GetRoundRandomSeed() int64 {
	return atomic.LoadInt64(&u.RoundRandomSeed)
}

// Clone returns a clone of the UnverifiedBlockBody
func (u *UnverifiedBlockBody) Clone() *UnverifiedBlockBody {
	cloneU := *u
	cloneU.PrevBlockVerificationTickets = copyVerificationTickets(u.PrevBlockVerificationTickets)

	cloneU.Txns = make([]*transaction.Transaction, 0, len(u.Txns))
	for _, t := range u.Txns {
		if t != nil {
			cloneU.Txns = append(cloneU.Txns, t.Clone())
		}
	}

	return &cloneU
}

/*Block - data structure that holds the block data */
// swagger:model
type Block struct {
	UnverifiedBlockBody
	VerificationTickets []*VerificationTicket `json:"verification_tickets,omitempty"`

	datastore.HashIDField
	Signature string `json:"signature"`

	ChainID   datastore.Key `json:"chain_id"`
	RoundRank int           `json:"-" msgpack:"-"` // rank of the block in the round it belongs to
	PrevBlock *Block        `json:"-" msgpack:"-"`
	Events    []event.Event `json:"-" msgpack:"-"`

	TxnsMap   map[string]bool `json:"-" msgpack:"-"`
	mutexTxns sync.RWMutex    `json:"-" msgpack:"-"`

	ClientState           util.MerklePatriciaTrieI `json:"-" msgpack:"-"`
	stateStatus           int8
	stateStatusMutex      sync.RWMutex `json:"-" msgpack:"-"`
	stateMutex            sync.RWMutex `json:"-" msgpack:"-"`
	blockState            int8
	isNotarized           bool
	isFinalised           bool         // set this field when the block is finalised
	ticketsMutex          sync.RWMutex `json:"-" msgpack:"-"`
	verificationStatus    int
	RunningTxnCount       int64           `json:"running_txn_count"`
	uniqueBlockExtensions map[string]bool `json:"-" msgpack:"-"`
	uniqueBlockExtMutex   sync.RWMutex    `json:"-" msgpack:"-"`
	*MagicBlock           `json:"magic_block,omitempty" msgpack:"mb,omitempty"`
	// StateChangesCount represents the state changes number in client state of current block.
	// this will be used to verify the state changes acquire from remote
	StateChangesCount int `json:"state_changes_count"`
}

// NewBlock - create a new empty block
func NewBlock(chainID datastore.Key, round int64) *Block {
	b := datastore.GetEntityMetadata("block").Instance().(*Block)
	b.Round = round
	b.ChainID = chainID
	return b
}

func (b *Block) GetUniqueBlockExtensions() map[string]bool {
	b.uniqueBlockExtMutex.RLock()
	defer b.uniqueBlockExtMutex.RUnlock()

	cb := make(map[string]bool, len(b.uniqueBlockExtensions))
	for k, v := range b.uniqueBlockExtensions {
		cb[k] = v
	}
	return cb
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
func (b *Block) ComputeProperties() error {
	if datastore.IsEmpty(b.ChainID) {
		b.ChainID = datastore.ToKey(config.GetServerChainID())
	}

	b.mutexTxns.Lock()
	defer b.mutexTxns.Unlock()
	if b.Txns != nil {
		b.TxnsMap = make(map[string]bool, len(b.Txns))
		for _, txn := range b.Txns {
			if err := txn.ComputeProperties(); err != nil {
				return err
			}
			b.TxnsMap[txn.Hash] = true
		}
	}
	return nil
}

// Decode decodes block from json bytes
func (b *Block) Decode(input []byte) error {
	return json.Unmarshal(input, b)
}

/*Validate - implementing the interface */
func (b *Block) Validate(_ context.Context) error {
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
		return common.NewError("incorrect_block_hash",
			fmt.Sprintf("computed block hash doesn't match with the hash of the block: %v: %v: %v",
				b.Hash, hash, b.getHashData()))
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
func (b *Block) GetScore() (int64, error) {
	return b.Round, nil
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

// SetPreviousBlock - set the previous block of this block
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

// InitStateDB - initialize the block's state from the db
// (assuming it's already computed).
func (b *Block) InitStateDB(ndb util.NodeDB) error {
	if _, err := ndb.GetNode(b.ClientStateHash); err != nil {
		b.SetStateStatus(StateFailed)
		return err
	}

	b.CreateState(ndb, b.ClientStateHash)
	b.SetStateStatus(StateSuccessful)
	return nil
}

// CreateState - create the state from the prior state db
func (b *Block) CreateState(pndb util.NodeDB, root util.Key) {
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, pndb, false)
	b.ClientState = util.NewMerklePatriciaTrie(ndb, util.Sequence(b.Round), root, statecache.NewEmpty())
}

// setClientState sets the block client state
// note: must be called with b.stateMutex protection
func (b *Block) setClientState(s util.MerklePatriciaTrieI) {
	b.ClientState = s
	b.ClientStateHash = s.GetRoot()
}

// SetClientState - set the block client state and update its ClientStateHash
func (b *Block) SetClientState(s util.MerklePatriciaTrieI) {
	b.stateMutex.Lock()
	b.setClientState(s)
	b.stateMutex.Unlock()
}

func (b *Block) SetStateChangesCount(s util.MerklePatriciaTrieI) {
	b.StateChangesCount = s.GetChangeCount()
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
				logging.Logger.Error("merge verification tickets - null ticket")
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

	hashBuilder := strings.Builder{}
	hashBuilder.WriteString(b.MinerID)
	hashBuilder.WriteString(":")
	hashBuilder.WriteString(b.PrevHash)
	hashBuilder.WriteString(":")
	hashBuilder.WriteString(common.TimeToString(b.CreationDate))
	hashBuilder.WriteString(":")
	hashBuilder.WriteString(strconv.FormatInt(b.Round, 10))
	hashBuilder.WriteString(":")
	hashBuilder.WriteString(strconv.FormatInt(b.GetRoundRandomSeed(), 10))
	hashBuilder.WriteString(":")
	hashBuilder.WriteString(strconv.Itoa(b.StateChangesCount))
	hashBuilder.WriteString(":")
	hashBuilder.WriteString(merkleRoot)
	hashBuilder.WriteString(":")
	hashBuilder.WriteString(rMerkleRoot)

	if b.MagicBlock != nil {
		if b.MagicBlock.Hash == "" {
			b.MagicBlock.Hash = b.MagicBlock.GetHash()
		}

		hashBuilder.WriteString(":")
		hashBuilder.WriteString(b.MagicBlock.Hash)
	}

	return hashBuilder.String()
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
	bs.StateChangesCount = b.StateChangesCount
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
func (b *Block) GetClients() ([]*client.Client, error) {
	cmap := make(map[string]*client.Client)
	for _, t := range b.Txns {
		if t.PublicKey == "" {
			continue
		}
		if _, ok := cmap[t.PublicKey]; ok {
			continue
		}
		c, err := client.GetClientFromCache(t.ClientID)
		if err != nil {
			c = client.NewClient()
			if err := c.SetPublicKey(t.PublicKey); err != nil {
				return nil, err
			}
		}

		cmap[t.PublicKey] = c
	}
	clients := make([]*client.Client, len(cmap))
	idx := 0
	for _, c := range cmap {
		clients[idx] = c
		idx++
	}
	return clients, nil
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
	return b.stateStatus >= StateSuccessful
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

// GetTransaction - get the transaction from the block
func (b *Block) GetTransaction(hash string) *transaction.Transaction {
	for _, txn := range b.Txns {
		if txn.GetKey() == hash {
			return txn
		}
	}
	return nil
}

// SetBlockNotarized - set the block as notarized
func (b *Block) SetBlockNotarized() {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()
	b.isNotarized = true
}

// IsBlockNotarized - is block notarized?
func (b *Block) IsBlockNotarized() bool {
	b.ticketsMutex.RLock()
	defer b.ticketsMutex.RUnlock()

	return b.isNotarized
}

// SetBlockFinalised - set the block as finalised
func (b *Block) SetBlockFinalised() {
	b.ticketsMutex.Lock()
	defer b.ticketsMutex.Unlock()
	b.isFinalised = true
}

// IsBlockFinalised - is block notarized?
func (b *Block) IsBlockFinalised() bool {
	b.ticketsMutex.RLock()
	defer b.ticketsMutex.RUnlock()

	return b.isFinalised
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
			logging.Logger.Error("unknown tickets - null ticket")
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
	b.uniqueBlockExtMutex.Lock()
	defer b.uniqueBlockExtMutex.Unlock()
	//TODO: We need to compare for view change and add the eb.MinerID only if he was in the view that b belongs to
	if b.uniqueBlockExtensions == nil {
		b.uniqueBlockExtensions = make(map[string]bool)
	}
	b.uniqueBlockExtensions[eb.MinerID] = true
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

// Clone returns a clone of the block instance
func (b *Block) Clone() *Block {
	clone := &Block{
		UnverifiedBlockBody: *b.UnverifiedBlockBody.Clone(),
		VerificationTickets: copyVerificationTickets(b.VerificationTickets),
		HashIDField:         b.HashIDField,
		Signature:           b.Signature,
		ChainID:             b.ChainID,
		RoundRank:           b.RoundRank,
		PrevBlock:           b.PrevBlock,
		RunningTxnCount:     b.RunningTxnCount,
		stateStatus:         b.stateStatus,
		blockState:          b.blockState,
		isNotarized:         b.isNotarized,
		verificationStatus:  b.verificationStatus,
		StateChangesCount:   b.StateChangesCount,
	}
	if b.MagicBlock != nil {
		clone.MagicBlock = b.MagicBlock.Clone()
	}

	b.mutexTxns.RLock()
	clone.TxnsMap = make(map[string]bool, len(b.TxnsMap))
	for k, v := range b.TxnsMap {
		clone.TxnsMap[k] = v
	}
	b.mutexTxns.RUnlock()

	b.stateMutex.RLock()
	if b.ClientState != nil {
		clone.CreateState(b.ClientState.GetNodeDB(), b.ClientStateHash)
	}
	b.stateMutex.RUnlock()

	clone.uniqueBlockExtensions = b.GetUniqueBlockExtensions()

	return clone
}

type Chainer interface {
	GetPreviousBlock(ctx context.Context, b *Block) *Block
	GetBlockStateChange(b *Block) error
	ComputeState(ctx context.Context, pb *Block, waitC ...chan struct{}) error
	GetStateDB() util.NodeDB
	UpdateState(ctx context.Context,
		b *Block, bState util.MerklePatriciaTrieI,
		txn *transaction.Transaction,
		blockStateCache *statecache.BlockCache,
		waitC ...chan struct{}) ([]event.Event, error)
	GetEventDb() *event.EventDb
	GetStateCache() *statecache.StateCache
}

// CreateStateWithPreviousBlock creates block client state with previous block
func CreateStateWithPreviousBlock(prevBlock *Block, stateDB util.NodeDB, round int64) util.MerklePatriciaTrieI {
	var pndb util.NodeDB
	var rootHash util.Key
	if prevBlock.ClientState == nil {
		logging.Logger.Error("create state db - prior state not available",
			zap.Int64("round", round),
			zap.Int64("previous round", prevBlock.Round),
			zap.String("previous block", prevBlock.Hash))
		pndb = stateDB
	} else {
		pndb = prevBlock.ClientState.GetNodeDB()
		if !bytes.Equal(prevBlock.ClientStateHash, prevBlock.ClientState.GetRoot()) {
			logging.Logger.Error("create state db - previous block state root does not match",
				zap.String("state root", string(prevBlock.ClientState.GetRoot())),
				zap.String("client state root", string(prevBlock.ClientStateHash)),
				zap.String("prev block", prevBlock.Hash),
				zap.Int64("prev round", prevBlock.Round))
		}
	}
	rootHash = prevBlock.ClientStateHash

	return CreateState(pndb, round, rootHash)
}

// CreateState creates state with state db and root
func CreateState(stateDB util.NodeDB, round int64, root util.Key) util.MerklePatriciaTrieI {
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, stateDB, false)
	return util.NewMerklePatriciaTrie(ndb, util.Sequence(round), root, statecache.NewEmpty())
}

// ComputeState computes block client state
func (b *Block) ComputeState(ctx context.Context, c Chainer, waitC ...chan struct{}) error {
	select {
	case <-ctx.Done():
		logging.Logger.Warn("computeState context done", zap.Error(ctx.Err()))
		b.SetStateStatus(StateCancelled)
		return ctx.Err()
	default:
	}

	if b.IsStateComputed() {
		return nil
	}

	b.stateMutex.Lock()
	defer b.stateMutex.Unlock()

	pb := b.PrevBlock
	if pb == b {
		b.PrevBlock = nil // reset (a real case, may be unexpected)
	}

	if pb == nil || !pb.IsStateComputed() {
		pb = c.GetPreviousBlock(ctx, b)
		if pb == nil {
			b.SetStateStatus(StateFailed)
			logging.Logger.Error("compute state - previous block not available",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.String("prev_block", b.PrevHash))
			return ErrPreviousBlockUnavailable
		}

		if !pb.IsStateComputed() {
			logging.Logger.Error("compute state - previous state is not computed",
				zap.Int64("round", b.Round),
				zap.Int64("prev_round", b.Round-1),
				zap.String("block", b.Hash))
			return ErrPreviousStateUnavailable
		}

		logging.Logger.Debug("compute state - set previous block",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int64("prev_round", pb.Round),
			zap.String("prev_block", pb.Hash))
	}

	if pb == b {
		b.PrevBlock = nil // reset (a real case, may be unexpected)
		logging.Logger.Error("computing block state",
			zap.String("error", "block_prev points to itself, or its state mutex does it"),
			zap.Int64("round", b.Round))
		return common.NewError("computing block state",
			"prev_block points to itself, or its state mutex does it")
	}

	if pb.ClientState == nil {
		logging.Logger.Error("compute state - previous state nil",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("prev_block", b.PrevHash),
			zap.Int8("prev_block_status", b.PrevBlock.GetStateStatus()))
		return ErrPreviousStateUnavailable
	}

	// Before continue the the following state update for transactions, the previous
	// block's state must be computed successfully.
	if !pb.IsStateComputed() {
		logging.Logger.Error("previous state not compute yet",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int8("state status", pb.GetStateStatus()))
		return ErrPreviousStateNotComputed
	}
	//b.SetStateDB(pb, c.GetStateDB())

	bState := CreateStateWithPreviousBlock(pb, c.GetStateDB(), b.Round)
	blockStateCache := statecache.NewBlockCache(c.GetStateCache(), statecache.Block{
		Round:    b.Round,
		Hash:     b.Hash,
		PrevHash: b.PrevHash,
	})

	beginStateRoot := bState.GetRoot()
	b.Events = []event.Event{}
	ts := time.Now()
	for _, txn := range b.Txns {
		if datastore.IsEmpty(txn.ClientID) {
			if err := txn.ComputeClientID(); err != nil {
				return err
			}
		}

		b.Events = append(b.Events, event.Event{
			BlockNumber: b.Round,
			TxHash:      txn.Hash,
			Type:        event.TypeStats,
			Tag:         event.TagAddTransactions,
			Index:       txn.Hash,
			Data:        transactionNodeToEventTransaction(txn, b.Hash, b.Round),
		})

		b.Events = append(b.Events, event.Event{
			Type:  event.TypeStats,
			Tag:   event.TagUpdateUserPayedFees,
			Index: txn.ClientID,
			Data: event.UserAggregate{
				UserID:    txn.ClientID,
				PayedFees: int64(txn.Fee),
			},
		})

		events, err := c.UpdateState(ctx, b, bState, txn, blockStateCache, waitC...)
		switch err {
		case context.Canceled:
			b.SetStateStatus(StateCancelled)
			logging.Logger.Debug("compute state - cancelled",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.String("client_state", util.ToHex(b.ClientStateHash)),
				zap.String("prev_block", b.PrevHash),
				zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)),
				zap.Error(err))
			//rollback changes for the next attempt
			//b.SetStateDB(b.PrevBlock, c.GetStateDB())
			b.Events = nil
			return err
		case context.DeadlineExceeded:
			b.SetStateStatus(StateCancelled)
			logging.Logger.Error("compute state - deadline exceeded",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.String("client_state", util.ToHex(b.ClientStateHash)),
				zap.String("prev_block", b.PrevHash),
				zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)),
				zap.Error(err))
			//rollback changes for the next attempt
			//b.SetStateDB(b.PrevBlock, c.GetStateDB())
			b.Events = nil
			return err
		case transaction.ErrSmartContractContext:
			b.SetStateStatus(StateCancelled)
			logging.Logger.Error("compute state - smart contract timeout",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.String("client_state", util.ToHex(b.ClientStateHash)),
				zap.String("prev_block", b.PrevHash),
				zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)),
				zap.Error(err))
			//rollback changes for the next attempt
			//b.SetStateDB(b.PrevBlock, c.GetStateDB())
			b.Events = nil
			return err
		default:
			if err != nil {
				b.SetStateStatus(StateFailed)
				logging.Logger.Error("compute state - update state failed",
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash),
					zap.String("client_state", util.ToHex(b.ClientStateHash)),
					zap.String("prev_block", b.PrevHash),
					zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)),
					zap.Error(err))
				return common.NewError("state_update_error", err.Error())
			}
		}
		b.Events = append(b.Events, events...)
	}

	if !bytes.Equal(b.ClientStateHash, bState.GetRoot()) {
		b.SetStateStatus(StateFailed)
		logging.Logger.Error("compute state - state hash mismatch",
			zap.String("minerID", b.MinerID),
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int("block_size", len(b.Txns)),
			zap.Int("changes", bState.GetChangeCount()),
			zap.String("begin_client_state", util.ToHex(beginStateRoot)),
			zap.String("computed_state_hash", util.ToHex(bState.GetRoot())),
			zap.String("block_state_hash", util.ToHex(b.ClientStateHash)),
			zap.String("prev_block", b.PrevHash),
			zap.String("prev_block_client_state", util.ToHex(pb.ClientStateHash)))
		return ErrStateMismatch
	}

	b.setClientState(bState)
	StateSanityCheck(ctx, b)
	b.SetStateStatus(StateSuccessful)

	// commit the block state cache to the global state cache
	blockStateCache.Commit()

	logging.Logger.Info("compute state successful",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("block ptr", fmt.Sprintf("%p", b)),
		zap.Int("block_size", len(b.Txns)),
		zap.Any("duration", time.Since(ts)),
		zap.Int("changes", b.ClientState.GetChangeCount()),
		zap.String("begin_client_state", util.ToHex(beginStateRoot)),
		zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())),
		zap.String("block_state_hash", util.ToHex(b.ClientStateHash)),
		zap.String("prev_block", b.PrevHash),
		zap.String("prev_block_client_state", util.ToHex(pb.ClientStateHash)))

	return nil
}

func transactionNodeToEventTransaction(tr *transaction.Transaction, blockHash string, round int64) event.Transaction {
	return event.Transaction{
		Hash:              tr.Hash,
		Round:             round,
		BlockHash:         blockHash,
		Version:           tr.Version,
		ClientId:          tr.ClientID,
		ToClientId:        tr.ToClientID,
		TransactionData:   tr.TransactionData,
		Value:             tr.Value,
		Signature:         tr.Signature,
		CreationDate:      int64(tr.CreationDate.Duration()),
		Fee:               tr.Fee,
		Nonce:             tr.Nonce,
		TransactionType:   tr.TransactionType,
		TransactionOutput: tr.TransactionOutput,
		OutputHash:        tr.OutputHash,
		Status:            tr.Status,
	}
}

// ApplyBlockStateChange apply and merge the state changes
func (b *Block) ApplyBlockStateChange(bsc *StateChange, c Chainer) error {
	b.stateMutex.Lock()
	defer b.stateMutex.Unlock()
	if b.stateStatus >= StateSuccessful {
		// already synced and applied by another goroutine
		return nil
	}

	ts := time.Now()
	defer func() {
		du := time.Since(ts)
		if du > 5*time.Second {
			logging.Logger.Error("apply block state changes took too long",
				zap.Duration("duration", du))
		}
	}()

	if b.Hash != bsc.Block {
		return ErrBlockHashMismatch
	}

	if !bytes.Equal(b.ClientStateHash, bsc.Hash) {
		return ErrBlockStateHashMismatch
	}

	root := bsc.GetRoot()
	if root == nil {
		if b.PrevBlock != nil && bytes.Equal(b.PrevBlock.ClientStateHash, b.ClientStateHash) {
			return nil
		}
		return common.NewError("state_root_error", "state root not correct")
	}

	pb := b.PrevBlock
	var clientState util.MerklePatriciaTrieI
	if pb != nil && pb.IsStateComputed() {
		clientState = CreateStateWithPreviousBlock(pb, c.GetStateDB(), b.Round)
	} else {
		clientState = CreateState(c.GetStateDB(), b.Round, root.GetHashBytes())
	}

	if len(bsc.Nodes) != b.StateChangesCount {
		logging.Logger.Error("apply block state changes, malformed state changes",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int("require state changes count", b.StateChangesCount),
			zap.Int("got state changes count", len(bsc.Nodes)))
		return state.ErrMalformedPartialState
	}

	err := clientState.MergeDB(bsc.GetNodeDB(), bsc.GetRoot().GetHashBytes(), bsc.GetDeadNodes())
	if err != nil {
		logging.Logger.Error("apply block state changes - error merging",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		return err
	}

	if !bytes.Equal(b.ClientStateHash, clientState.GetRoot()) {
		return common.NewError("state_mismatch", "Computed state hash doesn't match with the state hash of the block")
	}

	b.setClientState(clientState)
	b.SetStateStatus(StateSynched)

	logging.Logger.Info("sync state - apply block state changes success",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash))
	return nil
}

// SaveChanges persistent the state changes
func (b *Block) SaveChanges(ctx context.Context, c Chainer) error {
	b.stateMutex.Lock()
	defer b.stateMutex.Unlock()
	if b.ClientState == nil {
		logging.Logger.Error("save changes - client state is nil",
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash))
		return common.NewError("save_state_changes", "client state is nil")
	}

	var (
		ts          = time.Now()
		changeCount = b.ClientState.GetChangeCount()
	)

	switch b.GetStateStatus() {
	case StateSynched, StateSuccessful:
		if err := b.ClientState.SaveChanges(ctx, c.GetStateDB(), false); err != nil {
			logging.Logger.Error("save state",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.Int("block_size", len(b.Txns)),
				zap.Int("changes", changeCount),
				zap.String("client_state", util.ToHex(b.ClientStateHash)),
				zap.Duration("duration", time.Since(ts)),
				zap.Error(err))
			return err
		}
	default:
		return common.NewError("save_state_changes", "invalid state status")
	}

	StateSaveTimer.UpdateSince(ts)
	var (
		p95      = StateSaveTimer.Percentile(.95)
		duration = time.Since(ts)
	)

	if changeCount > 0 {
		StateChangeSizeMetric.Update(int64(changeCount))
	}

	if StateSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		logging.Logger.Debug("save state - slow",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int("block_size", len(b.Txns)),
			zap.Int("changes", changeCount),
			zap.String("client_state", util.ToHex(b.ClientStateHash)),
			zap.Duration("duration", duration),
			zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	} else {
		logging.Logger.Info("save state",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int("block_size", len(b.Txns)),
			zap.Int("changes", changeCount),
			zap.String("client_state", util.ToHex(b.ClientStateHash)),
			zap.Duration("duration", duration))
	}

	return nil
}

func copyVerificationTickets(src []*VerificationTicket) []*VerificationTicket {
	dst := make([]*VerificationTicket, 0, len(src))
	for i := range src {
		nvt := *src[i]
		dst = append(dst, &nvt)
	}
	return dst
}
