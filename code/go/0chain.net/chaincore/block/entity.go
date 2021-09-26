package block

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
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

	AccessMap map[datastore.Key]*AccessList `json:"accesses,omitempty"`
	// The entire transaction payload to represent full block
	Txns []*transaction.Transaction `json:"transactions,omitempty"`
}

type AccessList struct {
	Reads  []datastore.Key `json:"reads,omitempty"`
	Writes []datastore.Key `json:"writes,omitempty"`
}

func NewAccessList(rset, wset map[datastore.Key]bool) *AccessList {
	var r, w []datastore.Key
	for rkey := range rset {
		r = append(r, rkey)
	}
	for wkey := range wset {
		w = append(w, wkey)
	}

	return &AccessList{
		Reads:  r,
		Writes: w,
	}
}

func (al *AccessList) Includes(rset, wset map[datastore.Key]bool) bool {
	alr := al.Rset()
	alw := al.Wset()

	for rkey := range rset {
		if !alr[rkey] {
			return false
		}
	}
	for wkey := range wset {
		if !alw[wkey] {
			return false
		}
	}

	return true
}

func (al *AccessList) Rset() (rset map[datastore.Key]bool) {
	rset = make(map[datastore.Key]bool)
	for _, r := range al.Reads {
		rset[r] = true
	}

	return rset
}
func (al *AccessList) Wset() (wset map[datastore.Key]bool) {
	wset = make(map[datastore.Key]bool)
	for _, w := range al.Writes {
		wset[w] = true
	}

	return wset
}

func (al *AccessList) Clone() *AccessList {
	clone := &AccessList{
		Reads:  make([]datastore.Key, len(al.Reads)),
		Writes: make([]datastore.Key, len(al.Writes)),
	}
	for _, r := range al.Reads {
		clone.Reads = append(clone.Reads, r)
	}
	for _, w := range al.Writes {
		clone.Writes = append(clone.Writes, w)
	}

	return clone
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
	cloneU.AccessMap = make(map[datastore.Key]*AccessList, len(u.AccessMap))
	for tx_key, al := range u.AccessMap {
		if al != nil {
			cloneU.AccessMap[tx_key] = al.Clone()
		} else {
			cloneU.AccessMap[tx_key] = nil
		}
	}

	return &cloneU
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
	stateStatusMutex      sync.RWMutex `json:"-"`
	stateMutex            sync.RWMutex `json:"-"`
	blockState            int8
	isNotarized           bool
	ticketsMutex          sync.RWMutex
	verificationStatus    int
	RunningTxnCount       int64           `json:"running_txn_count"`
	UniqueBlockExtensions map[string]bool `json:"-"`
	*MagicBlock           `json:"magic_block,omitempty"`
}

// NewBlock - create a new empty block
func NewBlock(chainID datastore.Key, round int64) *Block {
	b := datastore.GetEntityMetadata("block").Instance().(*Block)
	b.Round = round
	b.ChainID = chainID
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

	b.mutexTxns.Lock()
	defer b.mutexTxns.Unlock()
	if b.Txns != nil {
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
func (b *Block) SetStateDB(prevBlock *Block, stateDB util.NodeDB) {
	var pndb util.NodeDB
	var rootHash util.Key
	if prevBlock.ClientState == nil {
		logging.Logger.Debug("Set state db -- prior state not available")
		if state.Debug() {
			logging.Logger.DPanic("Set state db - prior state not available")
		} else {
			pndb = stateDB
		}
	} else {
		pndb = prevBlock.ClientState.GetNodeDB()
	}
	rootHash = prevBlock.ClientStateHash
	logging.Logger.Debug("Prev state root", zap.Int64("round", b.Round),
		zap.String("prev_block", prevBlock.Hash),
		zap.String("root", util.ToHex(rootHash)))
	b.CreateState(pndb, rootHash)
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

//CreateState - create the state from the prior state db
func (b *Block) CreateState(pndb util.NodeDB, root util.Key) {
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, pndb, false)
	b.ClientState = util.NewMerklePatriciaTrie(ndb, util.Sequence(b.Round), root)
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
	hashData := b.MinerID + ":" + b.PrevHash + ":" + common.TimeToString(b.CreationDate) + ":" + strconv.FormatInt(b.Round, 10) + ":" + strconv.FormatInt(b.GetRoundRandomSeed(), 10) + ":" + merkleRoot + ":" + rMerkleRoot
	if b.MagicBlock != nil {
		if b.MagicBlock.Hash == "" {
			b.MagicBlock.Hash = b.MagicBlock.GetHash()
		}
		hashData += ":" + b.MagicBlock.Hash
	}
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

// Clone returns a clone of the block instance
func (b *Block) Clone() *Block {
	clone := &Block{
		UnverifiedBlockBody: *b.UnverifiedBlockBody.Clone(),
		VerificationTickets: copyVerificationTickets(b.VerificationTickets),
		HashIDField:         b.HashIDField,
		Signature:           b.Signature,
		ChainID:             b.ChainID,
		ChainWeight:         b.ChainWeight,
		RoundRank:           b.RoundRank,
		PrevBlock:           b.PrevBlock,
		RunningTxnCount:     b.RunningTxnCount,
		stateStatus:         b.stateStatus,
		blockState:          b.blockState,
		isNotarized:         b.isNotarized,
		verificationStatus:  b.verificationStatus,

		MagicBlock: b.MagicBlock.Clone(),
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

	clone.UniqueBlockExtensions = make(map[string]bool, len(b.UniqueBlockExtensions))
	for k, v := range b.UniqueBlockExtensions {
		clone.UniqueBlockExtensions[k] = v
	}

	return clone
}

type Chainer interface {
	GetPreviousBlock(ctx context.Context, b *Block) *Block
	GetBlockStateChange(b *Block) error
	ComputeState(ctx context.Context, pb *Block) error
	GetStateDB() util.NodeDB
	UpdateState(ctx context.Context, b *Block, txn *transaction.Transaction) (rset, wset map[datastore.Key]bool, err error)
}

// ComputeState computes block client state
func (b *Block) ComputeState(ctx context.Context, c Chainer) error {
	select {
	case <-ctx.Done():
		logging.Logger.Warn("computeState context done", zap.Error(ctx.Err()))
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
	if pb == nil {
		pb = c.GetPreviousBlock(ctx, b)
		if pb == nil {
			b.SetStateStatus(StateFailed)
			logging.Logger.Error("compute state - previous block not available",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.String("prev_block", b.PrevHash))
			return ErrPreviousBlockUnavailable
		}
	}

	if pb == b {
		b.PrevBlock = nil // reset (a real case, may be unexpected)
		logging.Logger.Error("computing block state",
			zap.String("error", "block_prev points to itself, or its state mutex does it"),
			zap.Int64("round", b.Round))
		return common.NewError("computing block state",
			"prev_block points to itself, or its state mutex does it")
	}
	if !pb.IsStateComputed() {
		if pb.GetStateStatus() == StateFailed {
			if err := c.GetBlockStateChange(pb); err != nil {
				logging.Logger.Error("fetchMissingStates failed", zap.Error(err))
				return err
			}
			if !pb.IsStateComputed() {
				return ErrPreviousStateUnavailable
			}
			logging.Logger.Debug("fetch previous block state from network successfully",
				zap.Int64("prev_round", pb.Round),
				zap.Any("hash", pb.Hash),
				zap.Any("prev_state", util.ToHex(pb.ClientStateHash)))
		} else {
			logging.Logger.Info("compute state - previous block state not ready",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.String("prev_block", b.PrevHash),
				zap.Int8("prev_block_state", pb.GetBlockState()),
				zap.Int8("prev_block_state_status", pb.GetStateStatus()))
			err := c.ComputeState(ctx, pb)
			if err != nil {
				pb.SetStateStatus(StateFailed)
				if state.DebugBlock() {
					logging.Logger.Error("compute state - error computing previous state",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.String("prev_block", b.PrevHash), zap.Error(err))
				} else {
					logging.Logger.Error("compute state - error computing previous state",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.String("prev_block", b.PrevHash), zap.Error(err))
				}
				return err
			}
		}
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
		logging.Logger.Error("previous state not compute successfully",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Any("state status", pb.GetStateStatus()))
		return ErrPreviousStateNotComputed
	}
	b.SetStateDB(pb, c.GetStateDB())

	beginState := b.ClientState.GetRoot()

	batcher := &ContentionFreeBatcher{8}
	err := b.applyTransactions(ctx, c, batcher)
	if err != nil {
		return err
	}

	logging.Logger.Info("compute state", zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("client_state", util.ToHex(b.ClientStateHash)),
		zap.String("begin_client_state", util.ToHex(beginState)),
		zap.String("after_client_state", util.ToHex(b.ClientState.GetRoot())),
		zap.String("prev_block", b.PrevHash),
		zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)))

	if bytes.Compare(b.ClientStateHash, b.ClientState.GetRoot()) != 0 {
		b.SetStateStatus(StateFailed)
		logging.Logger.Error("compute state - state hash mismatch",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.Int("block_size", len(b.Txns)),
			zap.Int("changes", b.ClientState.GetChangeCount()),
			zap.String("block_state_hash", util.ToHex(b.ClientStateHash)),
			zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())))
		return ErrStateMismatch
	}
	StateSanityCheck(ctx, b)
	b.SetStateStatus(StateSuccessful)
	logging.Logger.Info("compute state successful", zap.Int64("round", b.Round),
		zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)),
		zap.Int("changes", b.ClientState.GetChangeCount()),
		zap.String("block_state_hash", util.ToHex(b.ClientStateHash)),
		zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())))
	return nil
}

func (b *Block) applyTransactions(ctx context.Context, c Chainer, batcher Batcher) error {
	batches := batcher.Batch(b)

	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	finishChan := make(chan bool, 1)

	for i, batch := range batches {
		logging.Logger.Debug("apply transactions - running batch",
			zap.Int("batch", i),
			zap.Int("tx_count", len(batch)),
		)

		for _, txn := range batch {
			ctx, cancel := context.WithCancel(ctx)
			txn := txn
			wg.Add(1)

			go func() {
				defer wg.Done()
				if err := b.applyTransaction(txn, c, ctx); err != nil {
					errChan <- err
				}
			}()

			go func() {
				wg.Wait()
				finishChan <- true
			}()

			select {
			case err := <-errChan:
				logging.Logger.Error("apply transactions - batch failed with error",
					zap.Error(err))
				cancel()
				return err
			case <-ctx.Done():
				cancel()
				return errors.New("batch stopped due to context.Done()")
			case <-finishChan:
				logging.Logger.Debug("apply transactions - batch processed successfully",
					zap.Int("batch", i))
			}

		}
	}

	return nil
}

func (b *Block) applyTransaction(txn *transaction.Transaction, c Chainer, ctx context.Context) error {
	if datastore.IsEmpty(txn.ClientID) {
		txn.ComputeClientID()
	}
	rset, wset, err := c.UpdateState(ctx, b, txn)
	if err != nil {
		b.SetStateStatus(StateFailed)
		logging.Logger.Error("compute state - update state failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.String("client_state", util.ToHex(b.ClientStateHash)),
			zap.String("prev_block", b.PrevHash),
			zap.String("prev_client_state", util.ToHex(b.PrevBlock.ClientStateHash)),
			zap.Error(err))
		return common.NewError("state_update_error", "error updating state")
	}

	//we skip this check for blocks that do not contain access maps yet, in the future we will check more strictly
	if bal, ok := b.AccessMap[txn.GetKey()]; ok && !bal.Includes(rset, wset) {
		b.SetStateStatus(StateFailed)
		logging.Logger.Error("compute state - access lists are not equal",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.String("client_state", util.ToHex(b.ClientStateHash)),
			zap.String("prev_block", b.PrevHash),
			zap.String("prev_client_state", util.ToHex(b.PrevBlock.ClientStateHash)),
			zap.Error(err))
		return common.NewError("state_access_error", "error access lists")
	}

	return nil
}

// ApplyBlockStateChange apply and merge the state changes
func (b *Block) ApplyBlockStateChange(bsc *StateChange, c Chainer) error {
	b.stateMutex.Lock()
	defer b.stateMutex.Unlock()

	if b.Hash != bsc.Block {
		return ErrBlockHashMismatch
	}
	if bytes.Compare(b.ClientStateHash, bsc.Hash) != 0 {
		return ErrBlockStateHashMismatch
	}
	root := bsc.GetRoot()
	if root == nil {
		if b.PrevBlock != nil && bytes.Equal(b.PrevBlock.ClientStateHash, b.ClientStateHash) {
			return nil
		}
		return common.NewError("state_root_error", "state root not correct")
	}
	if b.ClientState == nil {
		b.CreateState(c.GetStateDB(), root.GetHashBytes())
	}

	//c.stateMutex.Lock()
	//defer c.stateMutex.Unlock()

	err := b.ClientState.MergeChanges(bsc.GetRoot().GetHashBytes(), bsc.GetChanges(), nil, bsc.StartRoot)
	if err != nil {
		logging.Logger.Error("apply block state change - error merging",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		return err
	}
	b.SetStateStatus(StateSynched)
	return nil
}

// SaveChanges persistents the state changes
func (b *Block) SaveChanges(ctx context.Context, c Chainer) error {
	b.stateMutex.Lock()
	defer b.stateMutex.Unlock()
	if b.ClientState == nil {
		logging.Logger.Error("save changes - client state is nil",
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash))
		return errors.New("save changes - client state is nil")
	}

	var err error
	ts := time.Now()
	switch b.GetStateStatus() {
	case StateSynched, StateSuccessful:
		err = b.ClientState.SaveChanges(ctx, c.GetStateDB(), false)
		lndb, ok := b.ClientState.GetNodeDB().(*util.LevelNodeDB)
		if ok {
			c.GetStateDB().(*util.PNodeDB).TrackDBVersion(lndb.GetDBVersion())
		}
	default:
		return common.NewError("state_save_without_success", "State can't be saved without successful computation")
	}
	duration := time.Since(ts)
	StateSaveTimer.UpdateSince(ts)
	p95 := StateSaveTimer.Percentile(.95)
	changeCount := b.ClientState.GetChangeCount()
	if changeCount > 0 {
		StateChangeSizeMetric.Update(int64(changeCount))
	}
	if StateSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		logging.Logger.Info("save state - slow", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", changeCount), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	} else {
		logging.Logger.Debug("save state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", changeCount), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration))
	}
	if err != nil {
		logging.Logger.Info("save state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", changeCount), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration), zap.Error(err))
	}

	return err
}

func copyVerificationTickets(src []*VerificationTicket) []*VerificationTicket {
	dst := make([]*VerificationTicket, 0, len(src))
	for i := range src {
		nvt := *src[i]
		dst = append(dst, &nvt)
	}
	return dst
}
