package chain

import (
	"container/ring"
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/storagesc"
	"github.com/herumi/bls/ffi/go/bls"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/minersc"
)

// notifySyncLFRStateTimeout is the maximum time allowed for sending a notification
// to a channel for syncing the latest finalized round state.
const notifySyncLFRStateTimeout = 3 * time.Second

// genesisRandomSeed is the geneisis block random seed
const genesisRandomSeed = 839695260482366273

var (
	ErrInsufficientChain = common.NewError("insufficient_chain",
		"Chain length not sufficient to perform the logic")
)

const (
	NOTARIZED = 1
	FINALIZED = 2

	AllMiners  = 11
	Generator  = 12
	Generators = 13

	// ViewChangeOffset is offset between block with new MB (501) and the block
	// where the new MB should be used (505).
	ViewChangeOffset = 4
)

/*ServerChain - the chain object of the chain  the server is responsible for */
var ServerChain *Chain

/*SetServerChain - set the server chain object */
func SetServerChain(c *Chain) {
	ServerChain = c
}

/*GetServerChain - returns the chain object for the server chain */
func GetServerChain() *Chain {
	return ServerChain
}

/*BlockStateHandler - handles the block state changes */
type BlockStateHandler interface {
	// SaveMagicBlock in store if it's about LFMB next in chain. It can return
	// nil if it don't have this ability.
	SaveMagicBlock() MagicBlockSaveFunc
	UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity)
	UpdateFinalizedBlock(ctx context.Context, b *block.Block)
}

type updateLFMBWithReply struct {
	block *block.Block
	clone *block.Block
	reply chan struct{}
}

/*Chain - data structure that holds the chain data*/
type Chain struct {
	datastore.IDField
	datastore.VersionField
	datastore.CreationDateField

	mutexViewChangeMB sync.RWMutex //nolint: structcheck, unused

	//Chain config goes into this object
	config.ChainConfig
	BlocksToSharder int

	MagicBlockStorage round.RoundStorage `json:"-"`

	PreviousMagicBlock *block.MagicBlock `json:"-"`
	mbMutex            sync.RWMutex

	getLFMB                      chan *block.Block `json:"-"`
	getLFMBClone                 chan *block.Block
	updateLFMB                   chan *updateLFMBWithReply `json:"-"`
	lfmbMutex                    sync.RWMutex
	latestOwnFinalizedBlockRound int64 // finalized by this node

	/* This is a cache of blocks that may include speculative blocks */
	blocks      map[datastore.Key]*block.Block
	blocksMutex *sync.RWMutex

	rounds      map[int64]round.RoundI
	roundsMutex *sync.RWMutex

	currentRound int64 `json:"-"`

	FeeStats transaction.FeeStats `json:"fee_stats"`

	LatestFinalizedBlock *block.Block `json:"latest_finalized_block,omitempty"` // Latest block on the chain the program is aware of
	lfbMutex             sync.RWMutex
	lfbSummary           *block.BlockSummary

	LatestDeterministicBlock *block.Block `json:"latest_deterministic_block,omitempty"`

	stateDB    util.NodeDB
	stateMutex *sync.RWMutex

	finalizedRoundsChannel chan round.RoundI
	finalizedBlocksChannel chan *finalizeBlockWithReply

	*Stats `json:"-"`

	BlockChain *ring.Ring `json:"-"`

	minersStake map[datastore.Key]int
	stakeMutex  *sync.Mutex

	nodePoolScorer node.PoolScorer

	GenerateTimeout int `json:"-"`
	genTimeoutMutex *sync.Mutex

	// syncStateTimeout is the timeout for syncing a MPT state from network
	syncStateTimeout time.Duration
	// bcStuckCheckInterval represents the BC stuck checking period
	bcStuckCheckInterval time.Duration
	// bcStuckTimeThreshold is the threshold time for checking if a BC is stuck
	bcStuckTimeThreshold time.Duration

	retryWaitTime  int
	retryWaitMutex *sync.Mutex

	blockFetcher *BlockFetcher

	crtCount int64 // Continuous/Current Round Timeout Count

	fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
	viewChanger                  ViewChanger
	afterFetcher                 AfterFetcher
	magicBlockSaver              MagicBlockSaver

	pruneStats *util.PruneStats
	// channel to trigger client state prune process
	pruneClientStateC chan struct{}

	configInfoDB string

	configInfoStore datastore.Store
	RoundF          round.RoundFactory

	magicBlockStartingRounds map[int64]*block.Block // block MB by starting round VC

	EventDb    *event.EventDb
	eventMutex *sync.RWMutex
	// LFB tickets channels
	getLFBTicket          chan *LFBTicket          // check out (any time)
	updateLFBTicket       chan *LFBTicket          // receive
	broadcastLFBTicket    chan *block.Block        // broadcast (update by LFB)
	subLFBTicket          chan chan *LFBTicket     // } wait for a received LFBTicket
	unsubLFBTicket        chan chan *LFBTicket     // }
	lfbTickerWorkerIsDone chan struct{}            // get rid out of context misuse
	syncLFBStateC         chan *block.BlockSummary // sync MPT state for latest finalized round
	syncLFBStateNowC      chan struct{}            // sync latest finalized round state from network immediately
	// precise DKG phases tracking
	phaseEvents chan PhaseEvent

	vldTxnsMtx               *sync.Mutex
	validatedTxnsCache       map[string]string // validated transactions, key as hash, value as signature
	verifyTicketsWithContext *common.WithContextFunc

	notarizedBlockVerifyC map[string]chan struct{}
	nbvcMutex             *sync.Mutex
	blockSyncC            map[string]chan chan *block.Block
	bscMutex              *sync.Mutex

	// compute state
	computeBlockStateC chan struct{}

	OnBlockAdded func(b *block.Block)
}

// SyncBlockReq represents a request to sync blocks, it will be
// send to sync block worker.
type SyncBlockReq struct {
	Hash     string
	Round    int64
	Num      int64
	SaveToDB bool
}

func (c *Chain) SetupEventDatabase() error {
	c.eventMutex.Lock()
	defer c.eventMutex.Unlock()
	if c.EventDb != nil {
		c.EventDb.Close()
		c.EventDb = nil
	}
	if !c.ChainConfig.DbsEvents().Enabled {
		return nil
	}

	time.Sleep(time.Second * 2)

	var err error
	c.EventDb, err = event.NewEventDb(c.ChainConfig.DbsEvents())
	if err != nil {
		return err
	}
	return nil
}

func (c *Chain) GetEventDb() *event.EventDb {
	c.eventMutex.RLock()
	defer c.eventMutex.RUnlock()
	return c.EventDb
}

// SyncLFBStateNow notify workers to start the LFB state sync immediately.
func (c *Chain) SyncLFBStateNow() {
	c.syncLFBStateNowC <- struct{}{}
}

// SetBCStuckTimeThreshold sets the BC stuck time threshold
func (c *Chain) SetBCStuckTimeThreshold(threshold time.Duration) {
	c.bcStuckTimeThreshold = threshold
}

// SetBCStuckCheckInterval sets the time interval for checking BC stuck
func (c *Chain) SetBCStuckCheckInterval(interval time.Duration) {
	c.bcStuckCheckInterval = interval
}

// SetSyncStateTimeout sets the state sync timeout
func (c *Chain) SetSyncStateTimeout(syncStateTimeout time.Duration) {
	c.syncStateTimeout = syncStateTimeout
}

var chainEntityMetadata *datastore.EntityMetadataImpl

func getNodePath(path string) util.Path {
	return util.Path(encryption.Hash(path))
}

func (c *Chain) GetBlockStateNode(block *block.Block, path string, v util.MPTSerializable) error {

	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	if block.ClientState == nil {
		return common.NewErrorf("get_block_state_node",
			"client state is nil, round %d", block.Round)
	}

	s := CreateTxnMPT(block.ClientState)
	return s.GetNodeValue(getNodePath(path), v)
}

func mbRoundOffset(rn int64) int64 {
	if rn < ViewChangeOffset+1 {
		return rn // the same
	}
	return rn - ViewChangeOffset // MB offset
}

// GetCurrentMagicBlock returns MB for current round
func (c *Chain) GetCurrentMagicBlock() *block.MagicBlock {
	var rn = c.GetCurrentRound()
	if rn == 0 {
		return c.GetLatestMagicBlock()
	}

	return c.GetMagicBlock(rn)
}

func (c *Chain) GetLatestMagicBlock() *block.MagicBlock {
	c.mbMutex.RLock()
	defer c.mbMutex.RUnlock()
	entity := c.MagicBlockStorage.GetLatest()
	if entity == nil {
		logging.Logger.Panic("failed to get magic block from mb storage")
	}
	return entity.(*block.MagicBlock)
}

func (c *Chain) GetMagicBlock(round int64) *block.MagicBlock {

	round = mbRoundOffset(round)

	c.mbMutex.RLock()
	entity := c.MagicBlockStorage.Get(round)
	if entity == nil {
		entity = c.MagicBlockStorage.GetLatest()
	}
	if entity == nil {
		logging.Logger.Panic("failed to get magic block from mb storage")
	}
	c.mbMutex.RUnlock()
	return entity.(*block.MagicBlock)
}

// GetMagicBlockNoOffset returns magic block of a given round with out offset
func (c *Chain) GetMagicBlockNoOffset(round int64) *block.MagicBlock {
	c.mbMutex.RLock()
	defer c.mbMutex.RUnlock()
	entity := c.MagicBlockStorage.Get(round)
	if entity == nil {
		entity = c.MagicBlockStorage.GetLatest()
	}
	if entity == nil {
		logging.Logger.Panic("failed to get magic block from mb storage")
	}
	return entity.(*block.MagicBlock)
}

func (c *Chain) GetPrevMagicBlock(r int64) *block.MagicBlock {

	r = mbRoundOffset(r)

	c.mbMutex.RLock()
	defer c.mbMutex.RUnlock()
	indexMB := c.MagicBlockStorage.FindRoundIndex(r)
	if indexMB <= 0 {
		return c.PreviousMagicBlock
	}
	prevRoundVC := c.MagicBlockStorage.GetRound(indexMB - 1)
	entity := c.MagicBlockStorage.Get(prevRoundVC)
	if entity != nil {
		return entity.(*block.MagicBlock)
	}
	return c.PreviousMagicBlock
}

func (c *Chain) GetPrevMagicBlockFromMB(mb *block.MagicBlock) (
	pmb *block.MagicBlock) {

	r := mbRoundOffset(mb.StartingRound)

	return c.GetPrevMagicBlock(r)
}

func (c *Chain) SetMagicBlock(mb *block.MagicBlock) {
	c.mbMutex.Lock()
	defer c.mbMutex.Unlock()
	if err := c.MagicBlockStorage.Put(mb, mb.StartingRound); err != nil {
		logging.Logger.Error("failed to put magic block", zap.Error(err))
	}

}

/*GetEntityMetadata - implementing the interface */
func (c *Chain) GetEntityMetadata() datastore.EntityMetadata {
	return chainEntityMetadata
}

/*Validate - implementing the interface */
func (c *Chain) Validate(ctx context.Context) error {
	if datastore.IsEmpty(c.ID) {
		return common.InvalidRequest("chain id is required")
	}
	if datastore.IsEmpty(c.OwnerID()) {
		return common.InvalidRequest("owner id is required")
	}
	return nil
}

/*Read - store read */
func (c *Chain) Read(ctx context.Context, key datastore.Key) error {
	return c.GetEntityMetadata().GetStore().Read(ctx, key, c)
}

/*Write - store read */
func (c *Chain) Write(ctx context.Context) error {
	return c.GetEntityMetadata().GetStore().Write(ctx, c)
}

/*Delete - store read */
func (c *Chain) Delete(ctx context.Context) error {
	return c.GetEntityMetadata().GetStore().Delete(ctx, c)
}

// DefaultSmartContractTimeout represents the default smart contract execution timeout
const DefaultSmartContractTimeout = time.Second

//NewChainFromConfig - create a new chain from config
func NewChainFromConfig() *Chain {
	chain := Provider().(*Chain)
	chain.ID = datastore.ToKey(config.Configuration().ChainID)

	chain.NotarizedBlocksCounts = make([]int64, chain.MinGenerators()+1)
	client.SetClientSignatureScheme(chain.ClientSignatureScheme())

	return chain
}

/*Provider - entity provider for chain object */
func Provider() datastore.Entity {
	c := &Chain{}
	c.ChainConfig = NewConfigImpl(&ConfigData{})
	c.ChainConfig.FromViper() //nolint: errcheck

	config.Configuration().ChainConfig = c.ChainConfig

	c.Initialize()
	c.Version = "1.0"

	c.blocks = make(map[string]*block.Block)
	c.blocksMutex = &sync.RWMutex{}

	c.rounds = make(map[int64]round.RoundI)
	c.roundsMutex = &sync.RWMutex{}
	c.eventMutex = &sync.RWMutex{}

	c.retryWaitMutex = &sync.Mutex{}
	c.genTimeoutMutex = &sync.Mutex{}
	c.stateMutex = &sync.RWMutex{}
	c.stakeMutex = &sync.Mutex{}
	c.InitializeCreationDate()
	c.nodePoolScorer = node.NewHashPoolScorer(encryption.NewXORHashScorer())

	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)
	c.SetMagicBlock(mb)
	c.Stats = &Stats{}
	c.blockFetcher = NewBlockFetcher()

	c.getLFBTicket = make(chan *LFBTicket) // should be unbuffered
	c.getLFMB = make(chan *block.Block)
	c.getLFMBClone = make(chan *block.Block)
	c.updateLFMB = make(chan *updateLFMBWithReply, 100)
	c.updateLFBTicket = make(chan *LFBTicket, 100)      //
	c.broadcastLFBTicket = make(chan *block.Block, 100) //
	c.subLFBTicket = make(chan chan *LFBTicket, 1)      //
	c.unsubLFBTicket = make(chan chan *LFBTicket, 1)    //
	c.lfbTickerWorkerIsDone = make(chan struct{})       //
	c.syncLFBStateC = make(chan *block.BlockSummary)
	c.syncLFBStateNowC = make(chan struct{})
	c.pruneClientStateC = make(chan struct{}, 1)

	c.phaseEvents = make(chan PhaseEvent, 1) // at least 1 for buffer required

	c.vldTxnsMtx = &sync.Mutex{}
	c.validatedTxnsCache = make(map[string]string)
	c.verifyTicketsWithContext = common.NewWithContextFunc(4)
	c.notarizedBlockVerifyC = make(map[string]chan struct{})
	c.nbvcMutex = &sync.Mutex{}
	c.blockSyncC = make(map[string]chan chan *block.Block)
	c.bscMutex = &sync.Mutex{}

	c.computeBlockStateC = make(chan struct{}, 1)
	return c
}

/*Initialize - intializes internal datastructures to start again */
func (c *Chain) Initialize() {
	c.setCurrentRound(0)
	c.SetLatestFinalizedBlock(nil)
	c.BlocksToSharder = 1
	//c.VerificationTicketsTo = AllMiners
	//c.ValidationBatchSize = 2000
	c.finalizedRoundsChannel = make(chan round.RoundI, 1)
	c.finalizedBlocksChannel = make(chan *finalizeBlockWithReply, 1)
	// TODO: debug purpose, add the stateDB back
	c.stateDB = stateDB
	//c.stateDB = util.NewMemoryNodeDB()
	c.BlockChain = ring.New(10000)
	c.minersStake = make(map[datastore.Key]int)
	c.magicBlockStartingRounds = make(map[int64]*block.Block)
	c.MagicBlockStorage = round.NewRoundStartingStorage()
	c.OnBlockAdded = func(b *block.Block) {
	}
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store, workdir string) {
	chainEntityMetadata = datastore.MetadataProvider()
	chainEntityMetadata.Name = "chain"
	chainEntityMetadata.Provider = Provider
	chainEntityMetadata.Store = store
	datastore.RegisterEntityMetadata("chain", chainEntityMetadata)
	SetupStateDB(workdir)
}

var stateDB *util.PNodeDB

//SetupStateDB - setup the state db
func SetupStateDB(workdir string) {

	datadir := "data/rocksdb/state"
	deadNodesDir := "data/rocksdb/deadnodes"
	logsdir := "/0chain/log/rocksdb/state"
	if len(workdir) > 0 {
		datadir = filepath.Join(workdir, datadir)
		deadNodesDir = filepath.Join(workdir, deadNodesDir)
		logsdir = filepath.Join(workdir, "log/rocksdb/state")
	}

	db, err := util.NewPNodeDB(datadir, deadNodesDir, logsdir)
	if err != nil {
		panic(err)
	}
	stateDB = db
}

// CloseStateDB closes the state db (rocksdb)
func CloseStateDB() {
	logging.Logger.Info("Closing StateDB")
	stateDB.Close()
	logging.Logger.Info("StateDB closed")
}

func (c *Chain) GetStateDB() util.NodeDB { return c.stateDB }

func (c *Chain) SetupConfigInfoDB(workdir string) {
	c.configInfoDB = "configdb"
	c.configInfoStore = ememorystore.GetStorageProvider()
	db, err := ememorystore.CreateDB(filepath.Join(workdir, "data/rocksdb/config"))
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool(c.configInfoDB, db)
}

func (c *Chain) GetConfigInfoDB() string {
	return c.configInfoDB
}

func (c *Chain) GetConfigInfoStore() datastore.Store {
	return c.configInfoStore
}

func (c *Chain) getInitialState(tokens currency.Coin) util.MPTSerializable {
	balance := &state.State{}
	_ = balance.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
	balance.Balance = tokens
	return balance
}

/*setupInitialState - setup the initial state based on configuration */
func (c *Chain) setupInitialState(initStates *state.InitStates) util.MerklePatriciaTrieI {
	pmt := util.NewMerklePatriciaTrie(c.stateDB, util.Sequence(0), nil)
	for _, v := range initStates.States {
		if _, err := pmt.Insert(util.Path(v.ID), c.getInitialState(v.Tokens)); err != nil {
			logging.Logger.Error("chain.stateDB insert failed", zap.Error(err))
		}
	}

	state := cstate.NewStateContext(nil, pmt, nil, nil, nil, nil, nil, nil, nil)
	mustInitPartitions(state)

	if err := pmt.SaveChanges(context.Background(), stateDB, false); err != nil {
		logging.Logger.Error("chain.stateDB save changes failed", zap.Error(err))
	}
	logging.Logger.Info("initial state root", zap.Any("hash", util.ToHex(pmt.GetRoot())))
	return pmt
}

func mustInitPartitions(state cstate.StateContextI) {
	if err := storagesc.InitPartitions(state); err != nil {
		logging.Logger.Panic("storagesc init partitions failed", zap.Error(err))
	}
}

/*GenerateGenesisBlock - Create the genesis block for the chain */
func (c *Chain) GenerateGenesisBlock(hash string, genesisMagicBlock *block.MagicBlock, initStates *state.InitStates) (round.RoundI, *block.Block) {
	//c.GenesisBlockHash = hash
	gb := block.NewBlock(c.GetKey(), 0)
	gb.Hash = hash
	gb.ClientState = c.setupInitialState(initStates)
	gb.SetStateStatus(block.StateSuccessful)
	gb.SetBlockState(block.StateNotarized)
	gb.ClientStateHash = gb.ClientState.GetRoot()
	gb.MagicBlock = genesisMagicBlock
	if err := c.UpdateMagicBlock(gb.MagicBlock); err != nil {
		panic(err)
	}
	gr := round.NewRound(0)
	c.SetRandomSeed(gr, genesisRandomSeed)
	gb.SetRoundRandomSeed(genesisRandomSeed)
	gr.Block = gb
	gr.AddNotarizedBlock(gb)
	gr.BlockHash = gb.Hash
	return gr, gb
}

/*AddGenesisBlock - adds the genesis block to the chain */
func (c *Chain) AddGenesisBlock(b *block.Block) {
	if b.Round != 0 {
		return
	}
	if err := c.UpdateMagicBlock(b.MagicBlock); err != nil {
		panic(err)
	}
	c.SetLatestFinalizedMagicBlock(b)
	c.SetLatestFinalizedBlock(b)
	c.SetLatestDeterministicBlock(b)
	c.blocks[b.Hash] = b
}

// AddLoadedFinalizedBlocks - adds the genesis block to the chain.
func (c *Chain) AddLoadedFinalizedBlocks(lfb, lfmb *block.Block, r *round.Round) {
	err := c.UpdateMagicBlock(lfmb.MagicBlock)
	if err != nil {
		logging.Logger.Warn("update magic block failed", zap.Error(err))
	}
	c.SetLatestFinalizedMagicBlock(lfmb)
	c.SetLatestFinalizedBlock(lfb)
	c.blocks[lfb.Hash] = lfb
	c.rounds[lfb.Round] = r
}

/*AddBlock - adds a block to the cache */
func (c *Chain) AddBlock(b *block.Block) *block.Block {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	return c.addBlock(b)
}

/*AddNotarizedBlockToRound - adds notarized block to round, sets RRS with block's if needed.
Client should check if block is valid notarized block, round should be created*/
func (c *Chain) AddNotarizedBlockToRound(r round.RoundI, b *block.Block) (*block.Block, round.RoundI) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()

	/*
		Since this nb mostly from a diff node, addBlock will return local block with the same hash if exists.
		Either way the block content is same, but we will get it from the local.
	*/
	b = c.addBlock(b)

	if b.Round == c.GetCurrentRound() {
		logging.Logger.Info("Adding a notarized block for current round", zap.Int64("Round", r.GetRoundNumber()))
	}

	// Notarized block can be obtained before timeout, but received after timeout when in theory new RRS is obtained,
	// we accept this notarization and set current round RRS and timeout count
	if r.GetRandomSeed() != b.GetRoundRandomSeed() {
		logging.Logger.Debug("RRS for notarized block is different", zap.Int64("Round_rrs", r.GetRandomSeed()),
			zap.Int64("Block_rrs", b.GetRoundRandomSeed()), zap.Int("notarized_blocks_count", len(r.GetNotarizedBlocks())))
		//reset round RRS only if it has no notarized rounds yet or received notarized block was obtained during next timeout (should not happen ever)
		if len(r.GetNotarizedBlocks()) == 0 || b.RoundTimeoutCount > r.GetTimeoutCount() {
			logging.Logger.Debug("AddNotarizedBlockToRound round and block random seed different",
				zap.Int64("Round", r.GetRoundNumber()),
				zap.Int64("Round_rrs", r.GetRandomSeed()),
				zap.Int64("Block_rrs", b.GetRoundRandomSeed()),
				zap.Int("Round_timeout", r.GetTimeoutCount()),
				zap.Int("Block_round_timeout", b.RoundTimeoutCount))
			r.SetRandomSeedForNotarizedBlock(b.GetRoundRandomSeed(), c.GetMiners(r.GetRoundNumber()).Size())
			r.SetTimeoutCount(b.RoundTimeoutCount)
		}
	}

	//TODO set only if this block rank is better
	c.SetRoundRank(r, b)
	b, _ = r.AddNotarizedBlock(b)

	return b, r
}

/*AddRoundBlock - add a block for a given round to the cache */
func (c *Chain) AddRoundBlock(r round.RoundI, b *block.Block) *block.Block {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	b2 := c.addBlock(b)
	if b2 != b {
		return b2
	}
	//TODO very dangerous code, we can break block hash with changing it!!! sort it out
	//b.SetRoundRandomSeed(r.GetRandomSeed())
	//b.RoundTimeoutCount = r.GetTimeoutCount()
	c.SetRoundRank(r, b)
	return b
}

func (c *Chain) addBlock(b *block.Block) *block.Block {
	if eb, ok := c.blocks[b.Hash]; ok {
		if eb != b {
			c.MergeVerificationTickets(eb, b.GetVerificationTickets())
		}
		return eb
	}
	c.blocks[b.Hash] = b

	if b.PrevBlock == nil {
		if pb, ok := c.blocks[b.PrevHash]; ok {
			b.SetPreviousBlock(pb)
		}
	}
	for pb := b.PrevBlock; pb != nil && pb != c.LatestDeterministicBlock; pb = pb.PrevBlock {
		pb.AddUniqueBlockExtension(b)
		if c.IsFinalizedDeterministically(pb) {
			c.SetLatestDeterministicBlock(pb)
			break
		}
	}

	c.OnBlockAdded(b)
	return b
}

/*GetBlock - returns a known block for a given hash from the cache */
func (c *Chain) GetBlock(ctx context.Context, hash string) (*block.Block, error) {
	c.blocksMutex.RLock()
	defer c.blocksMutex.RUnlock()
	return c.getBlock(ctx, hash)
}

func (c *Chain) getBlock(ctx context.Context, hash string) (*block.Block, error) {
	if b, ok := c.blocks[datastore.ToKey(hash)]; ok {
		return b, nil
	}
	return nil, common.NewError(datastore.EntityNotFound, fmt.Sprintf("Block with hash (%v) not found", hash))
}

/*DeleteBlock - delete a block from the cache */
func (c *Chain) DeleteBlock(ctx context.Context, b *block.Block) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	// if _, ok := c.blocks[b.Hash]; !ok {
	// 	return
	// }
	delete(c.blocks, b.Hash)
}

/*GetRoundBlocks - get the blocks for a given round */
func (c *Chain) GetRoundBlocks(round int64) []*block.Block {
	blocks := make([]*block.Block, 0, 1)
	c.blocksMutex.RLock()
	defer c.blocksMutex.RUnlock()
	for _, b := range c.blocks {
		if b.Round == round {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

/*DeleteBlocksBelowRound - delete all the blocks below this round */
func (c *Chain) DeleteBlocksBelowRound(round int64) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	ts := common.Now() - 60
	blocks := make([]*block.Block, 0, len(c.blocks))
	lfb := c.GetLatestFinalizedBlock()
	for _, b := range c.blocks {
		if b.Round < round && b.CreationDate < ts && b.Round < c.LatestDeterministicBlock.Round {
			logging.Logger.Debug("found block to delete", zap.Int64("round", round),
				zap.Int64("block_round", b.Round),
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.Int64("lf_round", lfb.Round))
			blocks = append(blocks, b)
		}
	}
	logging.Logger.Info("delete blocks below round",
		zap.Int64("round", c.GetCurrentRound()),
		zap.Int64("below_round", round), zap.Any("before", ts),
		zap.Int("total", len(c.blocks)), zap.Int("count", len(blocks)))
	for _, b := range blocks {
		b.Clear()
		delete(c.blocks, b.Hash)
	}

}

/*DeleteBlocks - delete a list of blocks */
func (c *Chain) DeleteBlocks(blocks []*block.Block) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	for _, b := range blocks {
		b.Clear()
		delete(c.blocks, b.Hash)
	}
}

/*PruneChain - prunes the chain */
func (c *Chain) PruneChain(_ context.Context, b *block.Block) {
	c.DeleteBlocksBelowRound(b.Round - 50)
}

/*ValidateMagicBlock - validate the block for a given round has the right magic block */
func (c *Chain) ValidateMagicBlock(ctx context.Context, mr *round.Round, b *block.Block) bool {
	mb := c.GetLatestFinalizedMagicBlockRound(mr.GetRoundNumber())
	if mb == nil {
		logging.Logger.Error("can't get lfmb`")
		return false
	}
	return b.LatestFinalizedMagicBlockHash == mb.Hash
}

// GetGenerators - get all the block generators for a given round.
func (c *Chain) GetGenerators(r round.RoundI) []*node.Node {
	miners := r.GetMinersByRank(c.GetMiners(r.GetRoundNumber()).CopyNodes())
	genNum := getGeneratorsNum(len(miners), c.MinGenerators(), c.GeneratorsPercent())
	if genNum > len(miners) {
		logging.Logger.Warn("get generators -- the number of generators is greater than the number of miners",
			zap.Any("num_generators", genNum), zap.Int("miner_by_rank", len(miners)),
			zap.Any("round", r.GetRoundNumber()))
		return miners
	}
	return miners[:genNum]
}

// GetGeneratorsNumOfMagicBlock returns the number of generators of given magic block
func (c *Chain) GetGeneratorsNumOfMagicBlock(mb *block.MagicBlock) int {
	if mb == nil {
		return c.MinGenerators()
	}

	return getGeneratorsNum(mb.Miners.Size(), c.MinGenerators(), c.GeneratorsPercent())
}

// GetGeneratorsNumOfRound returns the number of generators of a given round
func (c *Chain) GetGeneratorsNumOfRound(r int64) int {
	if mb := c.GetMagicBlock(r); mb != nil {
		return getGeneratorsNum(mb.Miners.Size(), c.MinGenerators(), c.GeneratorsPercent())
	}

	return c.MinGenerators()
}

// GetGeneratorsNum returns the number of generators that calculated base on current magic block
func (c *Chain) GetGeneratorsNum() int {
	if mb := c.GetCurrentMagicBlock(); mb != nil {
		return getGeneratorsNum(mb.Miners.Size(), c.MinGenerators(), c.GeneratorsPercent())
	}

	return c.MinGenerators()
}

// getGeneratorsNum calculates the number of generators
func getGeneratorsNum(minersNum, minGenerators int, generatorsPercent float64) int {
	return int(math.Max(float64(minGenerators), math.Ceil(float64(minersNum)*generatorsPercent)))
}

/*GetMiners - get all the miners for a given round */
func (c *Chain) GetMiners(round int64) *node.Pool {
	return c.GetMagicBlock(round).Miners
}

/*IsBlockSharder - checks if the sharder can store the block in the given round */
func (c *Chain) IsBlockSharder(b *block.Block, sharder *node.Node) bool {
	if c.NumReplicators() <= 0 {
		return true
	}
	scores := c.nodePoolScorer.ScoreHashString(c.GetMagicBlock(b.Round).Sharders, b.Hash)
	return sharder.IsInTop(scores, c.NumReplicators())
}

func (c *Chain) IsBlockSharderFromHash(nRound int64, bHash string, sharder *node.Node) bool {
	if c.NumReplicators() <= 0 {
		return true
	}
	scores := c.nodePoolScorer.ScoreHashString(c.GetMagicBlock(nRound).Sharders, bHash)
	return sharder.IsInTop(scores, c.NumReplicators())
}

/*CanShardBlockWithReplicators - checks if the sharder can store the block with nodes that store this block*/
func (c *Chain) CanShardBlockWithReplicators(nRound int64, hash string, sharder *node.Node) (bool, []*node.Node) {
	if c.NumReplicators() <= 0 {
		return true, c.GetMagicBlock(nRound).Sharders.CopyNodes()
	}
	scores := c.nodePoolScorer.ScoreHashString(c.GetMagicBlock(nRound).Sharders, hash)
	return sharder.IsInTopWithNodes(scores, c.NumReplicators())
}

// GetBlockSharders - get the list of sharders who would be replicating the block.
func (c *Chain) GetBlockSharders(b *block.Block) (sharders []string) {
	//TODO: sharders list needs to get resolved per the magic block of the block
	var (
		sharderPool  = c.GetMagicBlock(b.Round).Sharders
		sharderNodes = sharderPool.CopyNodes()
	)
	if c.NumReplicators() > 0 {
		scores := c.nodePoolScorer.ScoreHashString(sharderPool, b.Hash)
		sharderNodes = node.GetTopNNodes(scores, c.NumReplicators())
	}
	for _, sharder := range sharderNodes {
		sharders = append(sharders, sharder.GetKey())
	}
	return sharders
}

/*ValidGenerator - check whether this block is from a valid generator */
func (c *Chain) ValidGenerator(r round.RoundI, b *block.Block) bool {
	miner := c.GetMiners(r.GetRoundNumber()).GetNode(b.MinerID)
	if miner == nil {
		return false
	}

	isGen := c.IsRoundGenerator(r, miner)
	if !isGen {
		//This is a Byzantine condition?
		logging.Logger.Info("Received a block from non-generator", zap.Int("miner", miner.SetIndex), zap.Int64("RRS", r.GetRandomSeed()))
		gens := c.GetGenerators(r)

		logging.Logger.Info("Generators are: ", zap.Int64("round", r.GetRoundNumber()))
		for _, n := range gens {
			logging.Logger.Info("generator", zap.Int("Node", n.SetIndex))
		}
	}
	return isGen
}

/*GetNotarizationThresholdCount - gives the threshold count for block to be notarized*/
func (c *Chain) GetNotarizationThresholdCount(minersNumber int) int {
	notarizedPercent := float64(c.ThresholdByCount()) / 100
	thresholdCount := float64(minersNumber) * notarizedPercent
	return int(math.Ceil(thresholdCount))
}

/*ChainHasTransaction - indicates if this chain has the transaction */
func (c *Chain) chainHasTransaction(ctx context.Context, b *block.Block, txn *transaction.Transaction) (bool, error) {
	var pb = b
	visited := 0
	for cb := b; cb != nil; pb, cb = cb, c.GetLocalPreviousBlock(ctx, cb) {
		visited++
		if cb.Round == 0 {
			return false, nil
		}
		if cb.HasTransaction(txn.Hash) {
			return true, nil
		}
		if cb.CreationDate < txn.CreationDate-common.Timestamp(transaction.TXN_TIME_TOLERANCE) {
			return false, nil
		}
	}
	if true {
		logging.Logger.Debug("chain has txn", zap.Int64("round", b.Round), zap.Int64("upto_round", pb.Round),
			zap.Any("txn_ts", txn.CreationDate), zap.Any("upto_block_ts", pb.CreationDate), zap.Int("visited", visited))
	}
	return false, ErrInsufficientChain
}

func (c *Chain) getMiningStake(minerID datastore.Key) int {
	return c.minersStake[minerID]
}

//InitializeMinerPool - initialize the miners after their configuration is read
func (c *Chain) InitializeMinerPool(mb *block.MagicBlock) {
	numGenerators := c.GetGeneratorsNumOfMagicBlock(mb)
	for _, nd := range mb.Miners.CopyNodes() {
		ms := &MinerStats{}
		ms.GenerationCountByRank = make([]int64, numGenerators)
		ms.FinalizationCountByRank = make([]int64, numGenerators)
		ms.VerificationTicketsByRank = make([]int64, numGenerators)
		nd.ProtocolStats = ms
	}
}

/*AddRound - Add Round to the block */
func (c *Chain) AddRound(r round.RoundI) round.RoundI {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	roundNumber := r.GetRoundNumber()
	er, ok := c.rounds[roundNumber]
	if ok {
		return er
	}
	c.rounds[roundNumber] = r
	return r
}

/*GetRound - get a round */
func (c *Chain) GetRound(roundNumber int64) round.RoundI {
	c.roundsMutex.RLock()
	defer c.roundsMutex.RUnlock()
	r, ok := c.rounds[roundNumber]
	if !ok {
		return nil
	}
	return r
}

/*DeleteRound - delete a round and associated block data */
func (c *Chain) deleteRound(ctx context.Context, r round.RoundI) {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	delete(c.rounds, r.GetRoundNumber())
}

/*DeleteRoundsBelow - delete rounds below */
func (c *Chain) deleteRoundsBelow(roundNumber int64) {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	rounds := make([]round.RoundI, 0, 1)
	for _, r := range c.rounds {
		if r.GetRoundNumber() < roundNumber-10 && r.GetRoundNumber() != 0 {
			rounds = append(rounds, r)
		}
	}
	for _, r := range rounds {
		r.Clear()
		delete(c.rounds, r.GetRoundNumber())
	}
}

// SetRandomSeed - set the random seed for the round.
func (c *Chain) SetRandomSeed(r round.RoundI, randomSeed int64) bool {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	if r.HasRandomSeed() && randomSeed == r.GetRandomSeed() {
		logging.Logger.Debug("SetRandomSeed round already has the seed")
		return false
	}
	//if round was notarized once it is guaranteed that RRS is set, and in this case we will not reset it,
	//this RRS is protected with consensus and can be reset only with consensus in SetRandomSeedForNotarizedBlock
	if len(r.GetNotarizedBlocks()) > 0 {
		logging.Logger.Error("Current round has notarized block")
		return false
	}
	if randomSeed == 0 {
		logging.Logger.Error("SetRandomSeed -- seed is 0")
		return false
	}
	r.SetRandomSeed(randomSeed, c.GetMiners(r.GetRoundNumber()).Size())
	return true
}

// GetCurrentRound async safe.
func (c *Chain) GetCurrentRound() int64 {
	c.roundsMutex.RLock()
	defer c.roundsMutex.RUnlock()

	return c.getCurrentRound()
}

func (c *Chain) SetCurrentRound(r int64) {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	current := c.getCurrentRound()
	if current > r {
		logging.Logger.Error("set_current_round trying to set previous round as current, skipping",
			zap.Int64("current_round", current), zap.Int64("to_set_round", r))
		return
	}
	if current < r {
		logging.Logger.Info("Moving to the next round", zap.Int64("next_round", r))
		c.setCurrentRound(r)
		return
	}
}

func (c *Chain) setCurrentRound(r int64) {
	c.currentRound = r
}

func (c *Chain) getCurrentRound() int64 {
	return c.currentRound
}

func (c *Chain) getBlocks() []*block.Block {
	c.blocksMutex.RLock()
	defer c.blocksMutex.RUnlock()
	var bl []*block.Block
	for _, v := range c.blocks {
		bl = append(bl, v)
	}
	return bl
}

// SetRoundRank - set the round rank of the block.
func (c *Chain) SetRoundRank(r round.RoundI, b *block.Block) {
	miners := c.GetMiners(r.GetRoundNumber())
	if miners == nil || miners.Size() == 0 {
		logging.Logger.DPanic("set_round_rank  --  empty miners", zap.Any("round", r.GetRoundNumber()), zap.Any("block", b.Hash))
	}
	bNode := miners.GetNode(b.MinerID)
	if bNode == nil {
		logging.Logger.Warn("set_round_rank  --  get node by id", zap.Any("round", r.GetRoundNumber()),
			zap.Any("block", b.Hash), zap.Any("miner_id", b.MinerID), zap.Any("miners", miners))
		return
	}
	b.RoundRank = r.GetMinerRank(bNode)
}

func (c *Chain) SetGenerationTimeout(newTimeout int) {
	c.genTimeoutMutex.Lock()
	defer c.genTimeoutMutex.Unlock()
	c.GenerateTimeout = newTimeout
}

func (c *Chain) GetGenerationTimeout() int {
	c.genTimeoutMutex.Lock()
	defer c.genTimeoutMutex.Unlock()
	return c.GenerateTimeout
}

func (c *Chain) GetRetryWaitTime() int {
	c.retryWaitMutex.Lock()
	defer c.retryWaitMutex.Unlock()
	return c.retryWaitTime
}

func (c *Chain) SetRetryWaitTime(newWaitTime int) {
	c.retryWaitMutex.Lock()
	defer c.retryWaitMutex.Unlock()
	c.retryWaitTime = newWaitTime
}

/*GetUnrelatedBlocks - get blocks that are not related to the chain of the given block */
func (c *Chain) GetUnrelatedBlocks(maxBlocks int, b *block.Block) []*block.Block {
	c.blocksMutex.RLock()
	defer c.blocksMutex.RUnlock()
	var blocks []*block.Block
	var chain = make(map[datastore.Key]*block.Block)
	var prevRound = b.Round
	for pb := b.PrevBlock; pb != nil; pb = pb.PrevBlock {
		prevRound = pb.Round
		chain[pb.Hash] = pb
	}
	for _, rb := range c.blocks {
		if rb.Round >= prevRound && rb.Round < b.Round && common.WithinTime(int64(b.CreationDate), int64(rb.CreationDate), transaction.TXN_TIME_TOLERANCE) {
			if _, ok := chain[rb.Hash]; !ok {
				blocks = append(blocks, rb)
			}
			if len(blocks) >= maxBlocks {
				break
			}
		}
	}
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].Round > blocks[j].Round })
	return blocks
}

//ResetRoundTimeoutCount - reset the counter
func (c *Chain) ResetRoundTimeoutCount() {
	atomic.SwapInt64(&c.crtCount, 0)
}

//IncrementRoundTimeoutCount - increment the counter
func (c *Chain) IncrementRoundTimeoutCount() {
	atomic.AddInt64(&c.crtCount, 1)
}

//GetRoundTimeoutCount - get the counter
func (c *Chain) GetRoundTimeoutCount() int64 {
	return atomic.LoadInt64(&c.crtCount)
}

//GetSignatureScheme - get the signature scheme used by this chain
func (c *Chain) GetSignatureScheme() encryption.SignatureScheme {
	return encryption.GetSignatureScheme(c.ClientSignatureScheme())
}

// CanShardBlocks - is the network able to effectively shard the blocks?
func (c *Chain) CanShardBlocks(nRound int64) bool {
	mb := c.GetMagicBlock(nRound)
	activeShardersNum := mb.Sharders.GetActiveCount()
	mbShardersNum := mb.Sharders.Size()

	if activeShardersNum*100 < mbShardersNum*c.MinActiveSharders() {
		logging.Logger.Error("CanShardBlocks - can not shard blocks",
			zap.Int("active sharders", activeShardersNum),
			zap.Int("sharders size", mbShardersNum),
			zap.Int("min active sharders", c.MinActiveSharders()),
			zap.Int("left", activeShardersNum*100),
			zap.Int("right", mbShardersNum*c.MinActiveSharders()))
		return false
	}

	return true
}

// CanShardBlocksSharders - is the network able to effectively shard the blocks?
func (c *Chain) CanShardBlocksSharders(sharders *node.Pool) bool {
	return sharders.GetActiveCount()*100 >= sharders.Size()*c.MinActiveSharders()
}

// CanReplicateBlock - can the given block be effectively replicated?
func (c *Chain) CanReplicateBlock(b *block.Block) bool {

	if c.NumReplicators() <= 0 || c.MinActiveReplicators() == 0 {
		return c.CanShardBlocks(b.Round)
	}

	var (
		mb     = c.GetMagicBlock(b.Round)
		scores = c.nodePoolScorer.ScoreHashString(mb.Sharders, b.Hash)

		arCount  int
		minScore int32
	)

	if len(scores) == 0 {
		return c.CanShardBlocks(b.Round)
	}

	minScore = scores[len(scores)-1].Score

	for i := 0; i < len(scores); i++ {
		if scores[i].Score < minScore {
			break
		}
		if scores[i].Node.IsActive() {
			arCount++
			if arCount*100 >= c.NumReplicators()*c.MinActiveReplicators() {
				return true
			}
		}
	}

	return false
}

//SetFetchedNotarizedBlockHandler - setter for FetchedNotarizedBlockHandler
func (c *Chain) SetFetchedNotarizedBlockHandler(fnbh FetchedNotarizedBlockHandler) {
	c.fetchedNotarizedBlockHandler = fnbh
}

func (c *Chain) SetViewChanger(vcr ViewChanger) {
	c.viewChanger = vcr
}

func (c *Chain) SetAfterFetcher(afr AfterFetcher) {
	c.afterFetcher = afr
}

func (c *Chain) SetMagicBlockSaver(mbs MagicBlockSaver) {
	c.magicBlockSaver = mbs
}

//GetPruneStats - get the current prune stats
func (c *Chain) GetPruneStats() *util.PruneStats {
	return c.pruneStats
}

// HasClientStateStored returns true if given client state can be obtained
// from state db of the Chain.
func (c *Chain) HasClientStateStored(clientStateHash util.Key) bool {
	_, err := c.stateDB.GetNode(clientStateHash)
	return err == nil
}

// InitBlockState - initialize the block's state with the database state.
func (c *Chain) InitBlockState(b *block.Block) (err error) {
	if err = b.InitStateDB(c.stateDB); err != nil {
		logging.Logger.Error("init block state", zap.Int64("round", b.Round),
			zap.String("state", util.ToHex(b.ClientStateHash)),
			zap.Error(err))

		if err == util.ErrNodeNotFound {
			// get state from network
			logging.Logger.Info("init block state by syncing block state from network")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			doneC := make(chan struct{})
			errC := make(chan error)
			go func() {
				defer close(doneC)
				if err := c.GetBlockStateChange(b); err != nil {
					errC <- err
				}
			}()

			select {
			case <-ctx.Done():
				logging.Logger.Error("init block state failed",
					zap.Int64("round", b.Round),
					zap.Error(err))
			case err := <-errC:
				logging.Logger.Error("init block state failed", zap.Error(err))
			case <-doneC:
				logging.Logger.Info("init block state by synching block state from network successfully",
					zap.Int64("round", b.Round))
			}
		}
		return
	}
	logging.Logger.Info("init block state successful", zap.Int64("round", b.Round),
		zap.String("state", util.ToHex(b.ClientStateHash)))
	return
}

// SetLatestFinalizedBlock - set the latest finalized block.
func (c *Chain) SetLatestFinalizedBlock(b *block.Block) {
	c.lfbMutex.Lock()
	c.LatestFinalizedBlock = b
	if b != nil {
		logging.Logger.Debug("set lfb",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Bool("state_computed", b.IsStateComputed()))
		bs := b.GetSummary()
		c.lfbSummary = bs
		c.BroadcastLFBTicket(context.Background(), b)
		if !node.Self.IsSharder() {
			go c.notifyToSyncFinalizedRoundState(bs)
		}
	}
	c.lfbMutex.Unlock()

	// add LFB to blocks cache
	if b != nil {
		c.updateConfig(b)
		c.blocksMutex.Lock()
		defer c.blocksMutex.Unlock()
		cb, ok := c.blocks[b.Hash]
		if !ok {
			c.blocks[b.Hash] = b
		} else {
			if b.ClientState != nil && cb.ClientState != b.ClientState {
				cb.ClientState = b.ClientState
			}
		}
	}
}

func (c *Chain) getClientState(pb *block.Block) (util.MerklePatriciaTrieI, error) {
	if pb == nil || pb.ClientState == nil {
		return nil, fmt.Errorf("cannot get MPT from latest finalized block %v", pb)
	}
	return pb.ClientState, nil
}

func getConfigMap(clientState util.MerklePatriciaTrieI) (*minersc.GlobalSettings, error) {
	if clientState == nil {
		return nil, errors.New("client state is nil")
	}

	gl := &minersc.GlobalSettings{
		Fields: make(map[string]string),
	}
	err := clientState.GetNodeValue(util.Path(encryption.Hash(minersc.GLOBALS_KEY)), gl)
	if err != nil {
		return nil, err
	}

	return gl, nil
}

func (c *Chain) updateConfig(pb *block.Block) {
	clientState, err := c.getClientState(pb)
	if err != nil {
		// This might happen after stopping and starting the miners
		// and the MPT has not been setup yet.
		logging.Logger.Error("cannot get the client state from last block",
			zap.Error(err),
		)
		return
	}

	configMap, err := getConfigMap(clientState)
	if err != nil {
		logging.Logger.Info("cannot get global settings",
			zap.Int64("start of round", pb.Round),
			zap.Error(err),
		)
		return
	}

	err = c.ChainConfig.Update(configMap.Fields, configMap.Version)
	if err != nil {
		logging.Logger.Error("cannot update global settings",
			zap.Int64("start of round", pb.Round),
			zap.Error(err),
		)
	}
	logging.Logger.Info("config has been updated successfully",
		zap.Int64("start of round", pb.Round))

}

// GetLatestFinalizedBlock - get the latest finalized block.
func (c *Chain) GetLatestFinalizedBlock() *block.Block {
	c.lfbMutex.RLock()
	defer c.lfbMutex.RUnlock()
	return c.LatestFinalizedBlock
}

// GetLatestFinalizedBlockSummary - get the latest finalized block summary.
func (c *Chain) GetLatestFinalizedBlockSummary() *block.BlockSummary {
	c.lfbMutex.RLock()
	defer c.lfbMutex.RUnlock()
	return c.lfbSummary
}

func (c *Chain) IsActiveInChain() bool {
	var (
		mb          = c.GetCurrentMagicBlock()
		lfb         = c.GetLatestFinalizedBlock()
		olfbr       = c.LatestOwnFinalizedBlockRound()
		selfNodeKey = node.Self.Underlying().GetKey()
		crn         = c.GetCurrentRound()
	)
	return lfb.Round == olfbr && mb.IsActiveNode(selfNodeKey, crn)
}

func (c *Chain) UpdateMagicBlock(newMagicBlock *block.MagicBlock) error {
	if newMagicBlock.Miners == nil || newMagicBlock.Miners.Size() == 0 {
		return common.NewError("failed to update magic block",
			"there are no miners in the magic block")
	}

	var (
		self = node.Self.Underlying().GetKey()
	)
	lfmb := c.GetLatestFinalizedMagicBlock(common.GetRootContext())

	if lfmb != nil && newMagicBlock.IsActiveNode(self, c.GetCurrentRound()) &&
		lfmb.MagicBlockNumber == newMagicBlock.MagicBlockNumber-1 &&
		lfmb.MagicBlock.Hash != newMagicBlock.PreviousMagicBlockHash {

		logging.Logger.Error("failed to update magic block",
			zap.Any("finalized_magic_block_hash", lfmb.MagicBlock.Hash),
			zap.Any("new_magic_block_previous_hash", newMagicBlock.PreviousMagicBlockHash))
		return common.NewError("failed to update magic block",
			fmt.Sprintf("magic block's previous magic block hash (%v) doesn't equal latest finalized magic block id (%v)", newMagicBlock.PreviousMagicBlockHash, lfmb.MagicBlock.Hash))
	}

	// there's no new magic block
	if lfmb != nil && newMagicBlock.StartingRound <= lfmb.StartingRound {
		return nil
	}

	// initialize magicblock nodepools
	if err := c.UpdateNodesFromMagicBlock(newMagicBlock); err != nil {
		return common.NewErrorf("failed to update magic block", "%v", err)
	}

	if lfmb != nil {
		logging.Logger.Info("update magic block",
			zap.Int("old magic block miners num", lfmb.Miners.Size()),
			zap.Int("new magic block miners num", newMagicBlock.Miners.Size()),
			zap.Int64("old mb starting round", lfmb.StartingRound),
			zap.Int64("new mb starting round", newMagicBlock.StartingRound))

		if lfmb.Hash == newMagicBlock.PreviousMagicBlockHash {
			logging.Logger.Info("update magic block -- hashes match ",
				zap.Any("LFMB previous MB hash", lfmb.PreviousMagicBlockHash),
				zap.Any("new MB previous MB hash", newMagicBlock.PreviousMagicBlockHash))
			c.PreviousMagicBlock = lfmb.MagicBlock
		}
	}

	c.SetMagicBlock(newMagicBlock)
	return nil
}

func (c *Chain) UpdateNodesFromMagicBlock(newMagicBlock *block.MagicBlock) error {
	if err := c.SetupNodes(newMagicBlock); err != nil {
		return err
	}

	c.InitializeMinerPool(newMagicBlock)
	c.GetNodesPreviousInfo(newMagicBlock)

	// reset the monitor
	ResetStatusMonitor(newMagicBlock.StartingRound)
	return nil
}

func (c *Chain) SetupNodes(mb *block.MagicBlock) error {
	for _, mn := range mb.Miners.CopyNodesMap() {
		if err := node.Setup(mn); err != nil {
			return err
		}
	}
	for _, sh := range mb.Sharders.CopyNodesMap() {
		if err := node.Setup(sh); err != nil {
			return err
		}
	}

	return nil
}

func (c *Chain) SetLatestOwnFinalizedBlockRound(r int64) {
	c.lfbMutex.Lock()
	defer c.lfbMutex.Unlock()

	c.latestOwnFinalizedBlockRound = r
}

func (c *Chain) LatestOwnFinalizedBlockRound() int64 {
	c.lfbMutex.RLock()
	defer c.lfbMutex.RUnlock()

	return c.latestOwnFinalizedBlockRound
}

// SetLatestFinalizedMagicBlock - set the latest finalized block.
// TODO: this should be called when UpdateMagicBlock is called successfully
func (c *Chain) SetLatestFinalizedMagicBlock(b *block.Block) {

	if b == nil || b.MagicBlock == nil {
		return
	}

	latest := c.GetLatestFinalizedMagicBlock(common.GetRootContext())
	if latest != nil && latest.MagicBlock != nil &&
		latest.MagicBlock.MagicBlockNumber == b.MagicBlock.MagicBlockNumber-1 &&
		latest.MagicBlock.Hash != b.MagicBlock.PreviousMagicBlockHash {

		logging.Logger.DPanic(fmt.Sprintf("failed to set finalized magic block -- "+
			"hashes don't match up: chain's finalized block hash %v, block's"+
			" magic block previous hash %v",
			latest.MagicBlock.Hash,
			b.MagicBlock.PreviousMagicBlockHash))
	}

	if latest != nil && latest.MagicBlock.Hash == b.MagicBlock.Hash {
		return
	}

	logging.Logger.Warn("update lfmb",
		zap.Int64("mb_sr", b.MagicBlock.StartingRound),
		zap.String("mb_hash", b.MagicBlock.Hash))

	c.lfmbMutex.Lock()
	c.magicBlockStartingRounds[b.MagicBlock.StartingRound] = b
	c.lfmbMutex.Unlock()

	if latest == nil || b.StartingRound >= latest.StartingRound {
		c.updateLatestFinalizedMagicBlock(context.Background(), b)
	}
}

// GetLatestFinalizedMagicBlock returns a the latest finalized magic block
func (c *Chain) GetLatestFinalizedMagicBlock(ctx context.Context) (lfb *block.Block) {
	select {
	case lfb = <-c.getLFMB:
	case <-ctx.Done():
	}
	return
}

// GetLatestFinalizedMagicBlockClone returns a deep cloned latest finalized magic block
func (c *Chain) GetLatestFinalizedMagicBlockClone(ctx context.Context) (lfb *block.Block) {
	select {
	case lfb = <-c.getLFMBClone:
	case <-ctx.Done():
	}
	return
}

func (c *Chain) GetNodesPreviousInfo(mb *block.MagicBlock) {
	prevMB := c.GetPrevMagicBlockFromMB(mb)
	if prevMB != nil {
		numGenerators := c.GetGeneratorsNumOfMagicBlock(prevMB)
		for key, miner := range mb.Miners.CopyNodesMap() {
			if old := prevMB.Miners.GetNode(key); old != nil {
				miner.SetNode(old)
				if miner.ProtocolStats == nil {
					ms := &MinerStats{}
					ms.GenerationCountByRank = make([]int64, numGenerators)
					ms.FinalizationCountByRank = make([]int64, numGenerators)
					ms.VerificationTicketsByRank = make([]int64, numGenerators)
					miner.ProtocolStats = ms
				}
			}
		}
		for key, sharder := range mb.Sharders.CopyNodesMap() {
			if old := prevMB.Sharders.GetNode(key); old != nil {
				sharder.SetNode(old)
			}
		}
	}
}

func (c *Chain) Stop() {
	if stateDB != nil {
		stateDB.Flush()
	}
}

// PruneRoundStorage pruning storage
func (c *Chain) PruneRoundStorage(getTargetCount func(storage round.RoundStorage) int,
	storages ...round.RoundStorage) {

	for _, storage := range storages {
		targetCount := getTargetCount(storage)
		if targetCount == 0 {
			logging.Logger.Debug("prune storage -- skip. disabled")
			continue
		}
		rounds := storage.GetRounds()
		countRounds := len(rounds)
		if countRounds > targetCount {
			r := rounds[countRounds-targetCount-1]
			if err := storage.Prune(r); err != nil {
				logging.Logger.Error("failed to prune storage",
					zap.Int("count_rounds", countRounds),
					zap.Int("count_dest_prune", targetCount),
					zap.Int64("round", r),
					zap.Error(err))
			} else {
				logging.Logger.Debug("prune storage", zap.Int64("round", r))
			}
		}
	}
}

// ResetStatusMonitor resetting the node monitoring worker
func ResetStatusMonitor(round int64) {
	UpdateNodes <- round
}

func (c *Chain) SetLatestDeterministicBlock(b *block.Block) {
	lfb := c.LatestDeterministicBlock
	if lfb == nil || b.Round >= lfb.Round {
		c.LatestDeterministicBlock = b
	}
}

func (c *Chain) callViewChange(ctx context.Context, lfb *block.Block) ( //nolint: unused
	err error) {

	if c.viewChanger == nil {
		return // no ViewChanger no work here
	}

	// extract and send DKG phase first
	var pn minersc.PhaseNode
	if pn, err = c.GetPhaseOfBlock(lfb); err != nil {
		return common.NewErrorf("view_change", "getting phase node: %v", err)
	}

	// even if it executed on a shader we don't treat this phase as obtained
	// from sharders
	c.sendPhase(pn, false) // optimistic, never block here

	// this work is different for miners and sharders
	return c.viewChanger.ViewChange(ctx, lfb)
}

func (c *Chain) notifyToSyncFinalizedRoundState(bs *block.BlockSummary) {
	select {
	case c.syncLFBStateC <- bs:
	case <-time.NewTimer(notifySyncLFRStateTimeout).C:
		logging.Logger.Error("Send sync state for finalized round timeout")
	}
}

// UpdateBlocks updates block
func (c *Chain) UpdateBlocks(bs []*block.Block) {
	for i := range bs {
		r := c.GetRound(bs[i].Round)
		if r != nil {
			r.UpdateNotarizedBlock(bs[i])
		}
	}
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	for i := range bs {
		c.blocks[bs[i].Hash] = bs[i]
	}
}

// The ViewChanger represents node makes view change where a block with new
// magic block finalized. It called for every finalized block and used not
// only for a ViewChange.
type ViewChanger interface {
	// ViewChange is method called for every finalized block.
	ViewChange(ctx context.Context, lfb *block.Block) (err error)
}

// The AfterFetcher represents hooks performed during asynchronous finalized
// blocks fetching.
type AfterFetcher interface {
	// AfterFetch performed just after a block fetched verified and validated.
	// E.g. before the fetch function made some changes in the Chan. The
	// AfterFetch can be used to reject the block returning error. It never
	// receive an unverified and invalid block.
	AfterFetch(ctx context.Context, b *block.Block) (err error)
}

func (c *Chain) LoadMinersPublicKeys() error {
	mb := c.GetLatestFinalizedMagicBlock(common.GetRootContext())
	if mb == nil {
		return nil
	}

	for _, nd := range mb.Miners.Nodes {
		var pk bls.PublicKey
		if err := pk.DeserializeHexStr(nd.PublicKey); err != nil {
			return err
		}
	}

	return nil
}

func (c *Chain) AddValidatedTxns(hash, sig string) {
	c.vldTxnsMtx.Lock()
	c.validatedTxnsCache[hash] = sig
	c.vldTxnsMtx.Unlock()
}

func (c *Chain) DeleteValidatedTxns(hashes []string) {
	c.vldTxnsMtx.Lock()
	for _, hash := range hashes {
		delete(c.validatedTxnsCache, hash)
	}
	c.vldTxnsMtx.Unlock()
}

// FilterOutValidatedTxns filters out validated transactions
func (c *Chain) FilterOutValidatedTxns(txns []*transaction.Transaction) []*transaction.Transaction {
	needValidTxns := make([]*transaction.Transaction, 0, len(txns))
	c.vldTxnsMtx.Lock()
	for i, txn := range txns {
		sig, ok := c.validatedTxnsCache[txn.Hash]
		if ok && txn.Signature == sig {
			continue
		}

		needValidTxns = append(needValidTxns, txns[i])
	}
	c.vldTxnsMtx.Unlock()

	return needValidTxns
}

// BlockTicketsVerifyWithLock ensures that only one goroutine is allowed
// to verify the tickets for the same block.
func (c *Chain) BlockTicketsVerifyWithLock(ctx context.Context, blockHash string, f func() error) error {
	c.nbvcMutex.Lock()
	defer c.nbvcMutex.Unlock()
	ch, ok := c.notarizedBlockVerifyC[blockHash]
	if !ok {
		// only one gorountine is allowed for each block notarization tickets verification
		ch = make(chan struct{}, 1)
		c.notarizedBlockVerifyC[blockHash] = ch
	}

	select {
	case ch <- struct{}{}:
		defer func() {
			<-ch
		}()

		return f()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// MaxDeadNodesCount represents the max allowed dead nodes number in state db
func (c *Chain) MaxDeadNodesCount() int {
	return 5000
}
