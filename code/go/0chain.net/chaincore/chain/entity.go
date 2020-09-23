package chain

import (
	"container/ring"
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

//PreviousBlockUnavailable - to indicate an error condition when the previous
// block of a given block is not available.
const PreviousBlockUnavailable = "previous_block_unavailable"

var (
	// ErrPreviousBlockUnavailable - error for previous block is not available.
	ErrPreviousBlockUnavailable = common.NewError(PreviousBlockUnavailable,
		"Previous block is not available")
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

/*Chain - data structure that holds the chain data*/
type Chain struct {
	datastore.IDField
	datastore.VersionField
	datastore.CreationDateField

	mutexViewChangeMB sync.RWMutex

	//Chain config goes into this object
	*Config

	MagicBlockStorage round.RoundStorage `json:"-"`

	PreviousMagicBlock *block.MagicBlock `json:"-"`
	mbMutex            sync.RWMutex

	LatestFinalizedMagicBlock    *block.Block `json:"-"`
	lfmbMutex                    sync.RWMutex
	lfmbSummary                  *block.BlockSummary
	latestOwnFinalizedBlockRound int64 // finalized by this node

	/* This is a cache of blocks that may include speculative blocks */
	blocks      map[datastore.Key]*block.Block
	blocksMutex *sync.RWMutex

	rounds      map[int64]round.RoundI
	roundsMutex *sync.RWMutex

	CurrentRound int64 `json:"-"`

	LatestFinalizedBlock *block.Block `json:"latest_finalized_block,omitempty"` // Latest block on the chain the program is aware of
	lfbMutex             sync.RWMutex
	lfbSummary           *block.BlockSummary

	LatestDeterministicBlock *block.Block `json:"latest_deterministic_block,omitempty"`

	clientStateDeserializer state.DeserializerI
	stateDB                 util.NodeDB
	stateMutex              *sync.RWMutex

	finalizedRoundsChannel chan round.RoundI
	finalizedBlocksChannel chan *block.Block

	*Stats `json:"-"`

	BlockChain *ring.Ring `json:"-"`

	minersStake map[datastore.Key]int
	stakeMutex  *sync.Mutex

	nodePoolScorer node.PoolScorer

	GenerateTimeout int `json:"-"`
	genTimeoutMutex *sync.Mutex

	retry_wait_time  int
	retry_wait_mutex *sync.Mutex

	blockFetcher *BlockFetcher

	crtCount int64 // Continuous/Current Round Timeout Count

	fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
	viewChanger                  ViewChanger
	afterFetcher                 AfterFetcher

	pruneStats *util.PruneStats

	configInfoDB string

	configInfoStore datastore.Store
	RoundF          round.RoundFactory

	magicBlockStartingRounds map[int64]*block.Block // block MB by starting round VC

	// LFB tickets channels
	getLFBTicket       chan *LFBTicket      // check out (any time)
	updateLFBTicket    chan *LFBTicket      // receive
	broadcastLFBTicket chan *block.Block    // broadcast (update by LFB)
	subLFBTicket       chan chan *LFBTicket // } wait for a received LFBTicket
	unsubLFBTicket     chan chan *LFBTicket // }
}

var chainEntityMetadata *datastore.EntityMetadataImpl

func getNodePath(path string) util.Path {
	return util.Path(encryption.Hash(path))
}

func (mc *Chain) GetBlockStateNode(block *block.Block, path string) (
	seri util.Serializable, err error) {

	mc.stateMutex.Lock()
	defer mc.stateMutex.Unlock()

	if block.ClientState == nil {
		return nil, common.NewErrorf("get_block_state_node",
			"client state is nil, round %d", block.Round)
	}

	var state = CreateTxnMPT(block.ClientState)
	return state.GetNodeValue(getNodePath(path))
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

	rn = mbRoundOffset(rn)
	return c.GetMagicBlock(rn)
}

func (c *Chain) GetLatestMagicBlock() *block.MagicBlock {
	c.mbMutex.RLock()
	defer c.mbMutex.RUnlock()
	entity := c.MagicBlockStorage.GetLatest()
	if entity == nil {
		Logger.Panic("failed to get magic block from mb storage")
	}
	return entity.(*block.MagicBlock)
}

func (c *Chain) GetMagicBlock(round int64) *block.MagicBlock {

	round = mbRoundOffset(round)

	c.mbMutex.RLock()
	defer c.mbMutex.RUnlock()
	entity := c.MagicBlockStorage.Get(round)
	if entity == nil {
		entity = c.MagicBlockStorage.GetLatest()
	}
	if entity == nil {
		Logger.Panic("failed to get magic block from mb storage")
	}
	return entity.(*block.MagicBlock)
}

func (c *Chain) GetPrevMagicBlock(round int64) *block.MagicBlock {

	round = mbRoundOffset(round)

	c.mbMutex.RLock()
	defer c.mbMutex.RUnlock()
	indexMB := c.MagicBlockStorage.FindRoundIndex(round)
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

	var round = mbRoundOffset(mb.StartingRound)

	return c.GetPrevMagicBlock(round)
}

func (c *Chain) SetMagicBlock(mb *block.MagicBlock) {
	c.mbMutex.Lock()
	defer c.mbMutex.Unlock()
	if err := c.MagicBlockStorage.Put(mb, mb.StartingRound); err != nil {
		Logger.Error("failed to put magic block", zap.Error(err))
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
	if datastore.IsEmpty(c.OwnerID) {
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

//NewChainFromConfig - create a new chain from config
func NewChainFromConfig() *Chain {
	chain := Provider().(*Chain)
	chain.ID = datastore.ToKey(config.Configuration.ChainID)
	chain.Decimals = int8(viper.GetInt("server_chain.decimals"))
	chain.BlockSize = viper.GetInt32("server_chain.block.max_block_size")
	chain.MinBlockSize = viper.GetInt32("server_chain.block.min_block_size")
	chain.MaxByteSize = viper.GetInt64("server_chain.block.max_byte_size")
	chain.NumGenerators = viper.GetInt("server_chain.block.generators")
	chain.NotariedBlocksCounts = make([]int64, chain.NumGenerators+1)
	chain.NumReplicators = viper.GetInt("server_chain.block.replicators")
	chain.ThresholdByCount = viper.GetInt("server_chain.block.consensus.threshold_by_count")
	chain.ThresholdByStake = viper.GetInt("server_chain.block.consensus.threshold_by_stake")
	chain.OwnerID = viper.GetString("server_chain.owner")
	chain.ValidationBatchSize = viper.GetInt("server_chain.block.validation.batch_size")
	chain.RoundRange = viper.GetInt64("server_chain.round_range")
	chain.TxnMaxPayload = viper.GetInt("server_chain.transaction.payload.max_size")
	chain.PruneStateBelowCount = viper.GetInt("server_chain.state.prune_below_count")
	verificationTicketsTo := viper.GetString("server_chain.messages.verification_tickets_to")
	if verificationTicketsTo == "" || verificationTicketsTo == "all_miners" || verificationTicketsTo == "11" {
		chain.VerificationTicketsTo = AllMiners
	} else {
		chain.VerificationTicketsTo = Generator
	}

	// Health Check related counters
	// Work on deep scan
	config := &chain.HCCycleScan[DeepScan]

	config.Enabled = viper.GetBool("server_chain.health_check.deep_scan.enabled")
	config.BatchSize = viper.GetInt64("server_chain.health_check.deep_scan.batch_size")
	config.Window = viper.GetInt64("server_chain.health_check.deep_scan.window")

	config.SettleSecs = viper.GetInt("server_chain.health_check.deep_scan.settle_secs")
	config.Settle = time.Duration(config.SettleSecs) * time.Second

	config.RepeatIntervalMins = viper.GetInt("server_chain.health_check.deep_scan.repeat_interval_mins")
	config.RepeatInterval = time.Duration(config.RepeatIntervalMins) * time.Minute

	config.ReportStatusMins = viper.GetInt("server_chain.health_check.deep_scan.report_status_mins")
	config.ReportStatus = time.Duration(config.ReportStatusMins) * time.Minute

	// Work on proximity scan
	config = &chain.HCCycleScan[ProximityScan]

	config.Enabled = viper.GetBool("server_chain.health_check.proximity_scan.enabled")
	config.BatchSize = viper.GetInt64("server_chain.health_check.proximity_scan.batch_size")
	config.Window = viper.GetInt64("server_chain.health_check.proximity_scan.window")

	config.SettleSecs = viper.GetInt("server_chain.health_check.proximity_scan.settle_secs")
	config.Settle = time.Duration(config.SettleSecs) * time.Second

	config.RepeatIntervalMins = viper.GetInt("server_chain.health_check.proximity_scan.repeat_interval_mins")
	config.RepeatInterval = time.Duration(config.RepeatIntervalMins) * time.Minute

	config.ReportStatusMins = viper.GetInt("server_chain.health_check.proximity_scan.report_status_mins")
	config.ReportStatus = time.Duration(config.ReportStatusMins) * time.Minute

	chain.HealthShowCounters = viper.GetBool("server_chain.health_check.show_counters")

	chain.BlockProposalMaxWaitTime = viper.GetDuration("server_chain.block.proposal.max_wait_time") * time.Millisecond
	waitMode := viper.GetString("server_chain.block.proposal.wait_mode")
	if waitMode == "static" {
		chain.BlockProposalWaitMode = BlockProposalWaitStatic
	} else if waitMode == "dynamic" {
		chain.BlockProposalWaitMode = BlockProposalWaitDynamic
	}
	chain.ReuseTransactions = viper.GetBool("server_chain.block.reuse_txns")
	chain.SetSignatureScheme(viper.GetString("server_chain.client.signature_scheme"))

	chain.MinActiveSharders = viper.GetInt("server_chain.block.sharding.min_active_sharders")
	chain.MinActiveReplicators = viper.GetInt("server_chain.block.sharding.min_active_replicators")
	chain.SmartContractTimeout = viper.GetDuration("server_chain.smart_contract.timeout") * time.Millisecond
	chain.RoundTimeoutSofttoMin = viper.GetInt("server_chain.round_timeouts.softto_min")
	chain.RoundTimeoutSofttoMult = viper.GetInt("server_chain.round_timeouts.softto_mult")
	chain.RoundRestartMult = viper.GetInt("server_chain.round_timeouts.round_restart_mult")

	return chain
}

/*Provider - entity provider for chain object */
func Provider() datastore.Entity {
	c := &Chain{}
	c.Config = &Config{}
	c.Initialize()
	c.Version = "1.0"

	c.blocks = make(map[string]*block.Block)
	c.blocksMutex = &sync.RWMutex{}

	c.rounds = make(map[int64]round.RoundI)
	c.roundsMutex = &sync.RWMutex{}

	c.retry_wait_mutex = &sync.Mutex{}
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

	c.getLFBTicket = make(chan *LFBTicket)              // should be unbuffered
	c.updateLFBTicket = make(chan *LFBTicket, 100)      //
	c.broadcastLFBTicket = make(chan *block.Block, 100) //
	c.subLFBTicket = make(chan chan *LFBTicket, 1)      //
	c.unsubLFBTicket = make(chan chan *LFBTicket, 1)    //

	return c
}

/*Initialize - intializes internal datastructures to start again */
func (c *Chain) Initialize() {
	c.CurrentRound = 0
	c.SetLatestFinalizedBlock(nil)
	c.BlocksToSharder = 1
	c.VerificationTicketsTo = AllMiners
	c.ValidationBatchSize = 2000
	c.finalizedRoundsChannel = make(chan round.RoundI, 128)
	c.finalizedBlocksChannel = make(chan *block.Block, 128)
	c.clientStateDeserializer = &state.Deserializer{}
	c.stateDB = stateDB
	c.BlockChain = ring.New(10000)
	c.minersStake = make(map[datastore.Key]int)
	c.magicBlockStartingRounds = make(map[int64]*block.Block)
	c.MagicBlockStorage = round.NewRoundStartingStorage()
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	chainEntityMetadata = datastore.MetadataProvider()
	chainEntityMetadata.Name = "chain"
	chainEntityMetadata.Provider = Provider
	chainEntityMetadata.Store = store
	datastore.RegisterEntityMetadata("chain", chainEntityMetadata)
	SetupStateDB()
}

var stateDB *util.PNodeDB

//SetupStateDB - setup the state db
func SetupStateDB() {
	db, err := util.NewPNodeDB("data/rocksdb/state", "/0chain/log/rocksdb/state")
	if err != nil {
		panic(err)
	}
	stateDB = db
}

func (c *Chain) SetupConfigInfoDB() {
	c.configInfoDB = "configdb"
	c.configInfoStore = ememorystore.GetStorageProvider()
	db, err := ememorystore.CreateDB("data/rocksdb/config")
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

func (c *Chain) getInitialState() util.Serializable {
	tokens := viper.GetInt64("server_chain.tokens")
	balance := &state.State{}
	balance.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
	var cents int64 = 1
	for i := int8(0); i < c.Decimals; i++ {
		cents *= 10
	}
	balance.Balance = state.Balance(tokens * cents)
	return balance
}

/*setupInitialState - setup the initial state based on configuration */
func (c *Chain) setupInitialState() util.MerklePatriciaTrieI {
	pmt := util.NewMerklePatriciaTrie(c.stateDB, util.Sequence(0))
	pmt.Insert(util.Path(c.OwnerID), c.getInitialState())
	pmt.SaveChanges(c.stateDB, false)
	Logger.Info("initial state root", zap.Any("hash", util.ToHex(pmt.GetRoot())))
	return pmt
}

/*GenerateGenesisBlock - Create the genesis block for the chain */
func (c *Chain) GenerateGenesisBlock(hash string, genesisMagicBlock *block.MagicBlock) (round.RoundI, *block.Block) {
	c.GenesisBlockHash = hash
	gb := block.NewBlock(c.GetKey(), 0)
	gb.Hash = hash
	gb.ClientState = c.setupInitialState()
	gb.SetStateStatus(block.StateSuccessful)
	gb.SetBlockState(block.StateNotarized)
	gb.ClientStateHash = gb.ClientState.GetRoot()
	gb.MagicBlock = genesisMagicBlock
	c.UpdateMagicBlock(gb.MagicBlock)
	c.UpdateNodesFromMagicBlock(gb.MagicBlock)
	gr := round.NewRound(0)
	c.SetRandomSeed(gr, 839695260482366273)
	gr.ComputeMinerRanks(gb.MagicBlock.Miners)
	gr.Block = gb
	gr.AddNotarizedBlock(gb)
	return gr, gb
}

/*AddGenesisBlock - adds the genesis block to the chain */
func (c *Chain) AddGenesisBlock(b *block.Block) {
	if b.Round != 0 {
		return
	}
	c.SetLatestFinalizedMagicBlock(b)
	c.SetLatestFinalizedBlock(b)
	c.SetLatestDeterministicBlock(b)
	c.blocks[b.Hash] = b
	return
}

// AddLoadedFinalizedBlock - adds the genesis block to the chain.
func (c *Chain) AddLoadedFinalizedBlocks(lfb, lfmb *block.Block) {
	c.SetLatestFinalizedMagicBlock(lfmb)
	c.SetLatestFinalizedBlock(lfb)
	// c.LatestDeterministicBlock left as genesis
	c.blocks[lfb.Hash] = lfb
	return
}

// AddBlockNoPrevious adds block to cache and never calls
// async fetch previous block.
func (c *Chain) AddBlockNoPrevious(b *block.Block) *block.Block {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	return c.addBlockNoPrevious(b)
}

/*AddBlock - adds a block to the cache */
func (c *Chain) AddBlock(b *block.Block) *block.Block {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	return c.addBlock(b)
}

/*AddNotarizedBlockToRound - adds notarized block to cache and sync  info from notarized block to round  */
func (c *Chain) AddNotarizedBlockToRound(r round.RoundI, b *block.Block) (*block.Block, round.RoundI) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()

	/*
		Since this nb mostly from a diff node, addBlock will return local block with the same hash if exists.
		Either way the block content is same, but we will get it from the local.
	*/
	b = c.addBlock(b)

	if b.Round == c.GetCurrentRound() {
		Logger.Info("Adding a notarized block for current round", zap.Int64("Round", r.GetRoundNumber()))
	}

	// Get round data insync as it is the notarized block.
	if r.GetRandomSeed() != b.GetRoundRandomSeed() || r.GetTimeoutCount() != b.RoundTimeoutCount {
		Logger.Info("AddNotarizedBlockToRound round and block random seed different",
			zap.Int64("Round", r.GetRoundNumber()),
			zap.Int64("Round_rrs", r.GetRandomSeed()),
			zap.Int64("Block_rrs", b.GetRoundRandomSeed()))
		r.SetRandomSeedForNotarizedBlock(b.GetRoundRandomSeed())
		r.SetTimeoutCount(b.RoundTimeoutCount)
		r.ComputeMinerRanks(c.GetMiners(r.GetRoundNumber()))
	}

	c.SetRoundRank(r, b)
	if b.PrevBlock != nil {
		b.ComputeChainWeight()
	}
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
	b.SetRoundRandomSeed(r.GetRandomSeed())
	b.RoundTimeoutCount = r.GetTimeoutCount()
	c.SetRoundRank(r, b)
	if b.PrevBlock != nil {
		b.ComputeChainWeight()
	}
	return b
}

func (c *Chain) addBlockNoPrevious(b *block.Block) *block.Block {
	if eb, ok := c.blocks[b.Hash]; ok {
		if eb != b {
			c.MergeVerificationTickets(common.GetRootContext(), eb, b.GetVerificationTickets())
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
	return b
}

func (c *Chain) addBlock(b *block.Block) *block.Block {
	if eb, ok := c.blocks[b.Hash]; ok {
		if eb != b {
			c.MergeVerificationTickets(common.GetRootContext(), eb, b.GetVerificationTickets())
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
			Logger.Debug("found block to delete", zap.Int64("round", round),
				zap.Int64("block_round", b.Round),
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.Int64("lf_round", lfb.Round))
			blocks = append(blocks, b)
		}
	}
	Logger.Info("delete blocks below round",
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
	var mb = c.GetLatestFinalizedMagicBlockRound(mr.GetRoundNumber())
	return b.LatestFinalizedMagicBlockHash == mb.Hash
}

// GetGenerators - get all the block generators for a given round.
func (c *Chain) GetGenerators(r round.RoundI) []*node.Node {
	var miners []*node.Node
	miners = r.GetMinersByRank(c.GetMiners(r.GetRoundNumber()))
	if c.NumGenerators > len(miners) {
		Logger.Warn("get generators -- the number of generators is greater than the number of miners",
			zap.Any("num_generators", c.NumGenerators), zap.Any("miner_by_rank", miners),
			zap.Any("round", r.GetRoundNumber()))
		return miners
	}
	return miners[:c.NumGenerators]
}

/*GetMiners - get all the miners for a given round */
func (c *Chain) GetMiners(round int64) *node.Pool {
	mb := c.GetMagicBlock(round)
	Logger.Debug("get miners -- current magic block",
		zap.Any("miners", mb.Miners.N2NURLs()), zap.Any("round", round))

	return mb.Miners
}

/*IsBlockSharder - checks if the sharder can store the block in the given round */
func (c *Chain) IsBlockSharder(b *block.Block, sharder *node.Node) bool {
	if c.NumReplicators <= 0 {
		return true
	}
	scores := c.nodePoolScorer.ScoreHashString(c.GetMagicBlock(b.Round).Sharders, b.Hash)
	return sharder.IsInTop(scores, c.NumReplicators)
}

func (c *Chain) IsBlockSharderFromHash(nRound int64, bHash string, sharder *node.Node) bool {
	if c.NumReplicators <= 0 {
		return true
	}
	scores := c.nodePoolScorer.ScoreHashString(c.GetMagicBlock(nRound).Sharders, bHash)
	return sharder.IsInTop(scores, c.NumReplicators)
}

/*CanShardBlockWithReplicators - checks if the sharder can store the block with nodes that store this block*/
func (c *Chain) CanShardBlockWithReplicators(nRound int64, hash string, sharder *node.Node) (bool, []*node.Node) {
	if c.NumReplicators <= 0 {
		return true, c.GetMagicBlock(nRound).Sharders.CopyNodes()
	}
	scores := c.nodePoolScorer.ScoreHashString(c.GetMagicBlock(nRound).Sharders, hash)
	return sharder.IsInTopWithNodes(scores, c.NumReplicators)
}

// GetBlockSharders - get the list of sharders who would be replicating the block.
func (c *Chain) GetBlockSharders(b *block.Block) (sharders []string) {
	//TODO: sharders list needs to get resolved per the magic block of the block
	var (
		sharderPool  = c.GetMagicBlock(b.Round).Sharders
		sharderNodes = sharderPool.CopyNodes()
	)
	if c.NumReplicators > 0 {
		scores := c.nodePoolScorer.ScoreHashString(sharderPool, b.Hash)
		sharderNodes = node.GetTopNNodes(scores, c.NumReplicators)
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
		Logger.Info("Received a block from non-generator", zap.Int("miner", miner.SetIndex), zap.Int64("RRS", r.GetRandomSeed()))
		gens := c.GetGenerators(r)

		Logger.Info("Generators are: ", zap.Int64("round", r.GetRoundNumber()))
		for _, n := range gens {
			Logger.Info("generator", zap.Int("Node", n.SetIndex))
		}
	}
	return isGen
}

/*GetNotarizationThresholdCount - gives the threshold count for block to be notarized*/
func (c *Chain) GetNotarizationThresholdCount(minersNumber int) int {
	notarizedPercent := float64(c.ThresholdByCount) / 100
	thresholdCount := float64(minersNumber) * notarizedPercent
	return int(math.Ceil(thresholdCount))
}

// AreAllNodesActive - use this to check if all nodes needs to be active as in DKG
func (c *Chain) AreAllNodesActive() bool {
	mb := c.GetCurrentMagicBlock()
	active := mb.Miners.GetActiveCount()
	return active >= mb.Miners.Size()
}

/*CanStartNetwork - check whether the network can start */
func (c *Chain) CanStartNetwork() bool {
	mb := c.GetCurrentMagicBlock()
	active := mb.Miners.GetActiveCount()
	threshold := c.GetNotarizationThresholdCount(mb.Miners.Size())
	return active >= threshold && c.CanShardBlocks(c.GetCurrentRound())
}

/*ReadNodePools - read the node pools from configuration */
func (c *Chain) ReadNodePools(configFile string) {
	nodeConfig := config.ReadConfig(configFile)
	config := nodeConfig.Get("miners")
	mb := c.GetCurrentMagicBlock()
	if miners, ok := config.([]interface{}); ok {
		mb.Miners.AddNodes(miners)
		mb.Miners.ComputeProperties()
		c.InitializeMinerPool(mb)
	}
	config = nodeConfig.Get("sharders")
	if sharders, ok := config.([]interface{}); ok {
		mb.Sharders.AddNodes(sharders)
		mb.Sharders.ComputeProperties()
	}
}

/*ChainHasTransaction - indicates if this chain has the transaction */
func (c *Chain) ChainHasTransaction(ctx context.Context, b *block.Block, txn *transaction.Transaction) (bool, error) {
	var pb = b
	for cb := b; cb != nil; pb, cb = cb, c.GetPreviousBlock(ctx, cb) {
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
	if false {
		Logger.Debug("chain has txn", zap.Int64("round", b.Round), zap.Int64("upto_round", pb.Round), zap.Any("txn_ts", txn.CreationDate), zap.Any("upto_block_ts", pb.CreationDate))
	}
	return false, ErrInsufficientChain
}

func (c *Chain) updateMiningStake(minerID datastore.Key, stake int) {
	c.stakeMutex.Lock()
	defer c.stakeMutex.Unlock()
	c.minersStake[minerID] = stake
}

func (c *Chain) getMiningStake(minerID datastore.Key) int {
	return c.minersStake[minerID]
}

//InitializeMinerPool - initialize the miners after their configuration is read
func (c *Chain) InitializeMinerPool(mb *block.MagicBlock) {
	for _, nd := range mb.Miners.CopyNodes() {
		ms := &MinerStats{}
		ms.GenerationCountByRank = make([]int64, c.NumGenerators)
		ms.FinalizationCountByRank = make([]int64, c.NumGenerators)
		ms.VerificationTicketsByRank = make([]int64, c.NumGenerators)
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
	round, ok := c.rounds[roundNumber]
	if !ok {
		return nil
	}
	return round
}

/*DeleteRound - delete a round and associated block data */
func (c *Chain) DeleteRound(ctx context.Context, r round.RoundI) {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	delete(c.rounds, r.GetRoundNumber())
}

/*DeleteRoundsBelow - delete rounds below */
func (c *Chain) DeleteRoundsBelow(ctx context.Context, roundNumber int64) {
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
		return false
	}
	r.SetRandomSeed(randomSeed)
	r.ComputeMinerRanks(c.GetMiners(r.GetRoundNumber()))
	roundNumber := r.GetRoundNumber()
	if roundNumber > c.CurrentRound {
		c.CurrentRound = roundNumber
	}
	return true
}

// GetCurrentRound async safe.
func (c *Chain) GetCurrentRound() int64 {
	c.roundsMutex.RLock()
	defer c.roundsMutex.RUnlock()

	return c.CurrentRound
}

func (c *Chain) SetCurrentRound(round int64) {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	c.CurrentRound = round
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
	if miners == nil || miners.MapSize() == 0 {
		Logger.DPanic("set_round_rank  --  empty miners", zap.Any("round", r.GetRoundNumber()), zap.Any("block", b.Hash))
	}
	bNode := miners.GetNode(b.MinerID)
	if bNode == nil {
		Logger.Warn("set_round_rank  --  get node by id", zap.Any("round", r.GetRoundNumber()),
			zap.Any("block", b.Hash), zap.Any("miner_id", b.MinerID), zap.Any("miners", miners))
		return
	}
	rank := r.GetMinerRank(bNode)
	if rank >= c.NumGenerators {
		Logger.Error(fmt.Sprintf("Round# %v generator miner ID %v rank is greater than num generators. State= %v, rank= %v, generators = %v", r.GetRoundNumber(), bNode.SetIndex, r.GetState(), rank, c.NumGenerators))
	}
	b.RoundRank = rank
	//TODO: Remove this log
	Logger.Info(fmt.Sprintf("Round# %v generator miner ID %v State= %v, rank= %v", r.GetRoundNumber(), bNode.SetIndex, r.GetState(), rank))
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
	c.retry_wait_mutex.Lock()
	defer c.retry_wait_mutex.Unlock()
	return c.retry_wait_time
}

func (c *Chain) SetRetryWaitTime(newWaitTime int) {
	c.retry_wait_mutex.Lock()
	defer c.retry_wait_mutex.Unlock()
	c.retry_wait_time = newWaitTime
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
	c.crtCount = 0
}

//IncrementRoundTimeoutCount - increment the counter
func (c *Chain) IncrementRoundTimeoutCount() {
	c.crtCount++
}

//GetRoundTimeoutCount - get the counter
func (c *Chain) GetRoundTimeoutCount() int64 {
	return c.crtCount
}

//SetSignatureScheme - set the client signature scheme to be used by this chain
func (c *Chain) SetSignatureScheme(sigScheme string) {
	c.ClientSignatureScheme = sigScheme
	client.SetClientSignatureScheme(c.ClientSignatureScheme)
}

//GetSignatureScheme - get the signature scheme used by this chain
func (c *Chain) GetSignatureScheme() encryption.SignatureScheme {
	return encryption.GetSignatureScheme(c.ClientSignatureScheme)
}

// CanShardBlocks - is the network able to effectively shard the blocks?
func (c *Chain) CanShardBlocks(nRound int64) bool {
	mb := c.GetMagicBlock(nRound)
	return mb.Sharders.GetActiveCount()*100 >= mb.Sharders.Size()*c.MinActiveSharders
}

// CanShardBlocksSharders - is the network able to effectively shard the blocks?
func (c *Chain) CanShardBlocksSharders(sharders *node.Pool) bool {
	return sharders.GetActiveCount()*100 >= sharders.Size()*c.MinActiveSharders
}

// CanReplicateBlock - can the given block be effectively replicated?
func (c *Chain) CanReplicateBlock(b *block.Block) bool {

	if c.NumReplicators <= 0 || c.MinActiveReplicators == 0 {
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
			if arCount*100 >= c.NumReplicators*c.MinActiveReplicators {
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
		Logger.Error("init block state", zap.Int64("round", b.Round),
			zap.String("state", util.ToHex(b.ClientStateHash)),
			zap.Error(err))
		return
	}
	Logger.Info("init block state successful", zap.Int64("round", b.Round),
		zap.String("state", util.ToHex(b.ClientStateHash)))
	return
}

// SetLatestFinalizedBlock - set the latest finalized block.
func (c *Chain) SetLatestFinalizedBlock(b *block.Block) {
	c.lfbMutex.Lock()
	defer c.lfbMutex.Unlock()

	c.LatestFinalizedBlock = b
	if b != nil {
		c.lfbSummary = b.GetSummary()
		c.BroadcastLFBTicket(common.GetRootContext(), b)
	}
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
	return mb.IsActiveNode(selfNodeKey, crn) && lfb.Round == olfbr
}

func (c *Chain) UpdateMagicBlock(newMagicBlock *block.MagicBlock) error {
	if newMagicBlock.Miners == nil || newMagicBlock.Miners.MapSize() == 0 {
		return common.NewError("failed to update magic block",
			"there are no miners in the magic block")
	}

	var (
		self = node.Self.Underlying().GetKey()
		lfmb = c.GetLatestFinalizedMagicBlock()
	)

	if newMagicBlock.IsActiveNode(self, c.GetCurrentRound()) && lfmb != nil &&
		lfmb.MagicBlock.MagicBlockNumber == newMagicBlock.MagicBlockNumber-1 &&
		lfmb.MagicBlock.Hash != newMagicBlock.PreviousMagicBlockHash {

		Logger.Error("failed to update magic block", zap.Any("finalized_magic_block_hash", c.GetLatestFinalizedMagicBlock().MagicBlock.Hash), zap.Any("new_magic_block_previous_hash", newMagicBlock.PreviousMagicBlockHash))
		return common.NewError("failed to update magic block", fmt.Sprintf("magic block's previous magic block hash (%v) doesn't equal latest finalized magic block id (%v)", newMagicBlock.PreviousMagicBlockHash, c.GetLatestFinalizedMagicBlock().MagicBlock.Hash))
	}
	mb := c.GetCurrentMagicBlock()

	Logger.Info("update magic block", zap.Any("old block", mb), zap.Any("new block", newMagicBlock))

	if mb != nil && mb.Hash == newMagicBlock.PreviousMagicBlockHash {
		Logger.Info("update magic block -- hashes match ",
			zap.Any("LFMB previous MB hash", mb.Hash),
			zap.Any("new MB previous MB hash", newMagicBlock.PreviousMagicBlockHash))
		c.PreviousMagicBlock = mb
	}

	c.SetMagicBlock(newMagicBlock)
	return nil
}

func (c *Chain) UpdateNodesFromMagicBlock(newMagicBlock *block.MagicBlock) {

	var (
		prev = c.GetMagicBlock(newMagicBlock.StartingRound - 1) //
		keep = collectNodes(prev, newMagicBlock)                // this and new
	)

	c.SetupNodes(newMagicBlock)

	newMagicBlock.Sharders.ComputeProperties()
	newMagicBlock.Miners.ComputeProperties()

	c.InitializeMinerPool(newMagicBlock)
	c.GetNodesPreviousInfo(newMagicBlock)

	node.DeregisterNodes(keep)

	// reset the monitor
	ResetStatusMonitor(newMagicBlock.StartingRound)
}

func (c *Chain) SetupNodes(mb *block.MagicBlock) {
	for _, miner := range mb.Miners.CopyNodesMap() {
		miner.ComputeProperties()
		node.Setup(miner)
	}
	for _, sharder := range mb.Sharders.CopyNodesMap() {
		sharder.ComputeProperties()
		node.Setup(sharder)
	}
}

// collect nodes from given MBs
func collectNodes(mbs ...*block.MagicBlock) (keep map[string]struct{}) {
	keep = make(map[string]struct{})
	for _, mb := range mbs {
		if mb == nil {
			continue
		}
		for _, pool := range []*node.Pool{mb.Miners, mb.Sharders} {
			for _, k := range pool.Keys() {
				keep[k] = struct{}{}
			}
		}
	}
	return
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

// SetLatestFinalizedBlock - set the latest finalized block.
func (c *Chain) SetLatestFinalizedMagicBlock(b *block.Block) {

	if b == nil || b.MagicBlock == nil {
		return
	}

	c.lfmbMutex.Lock()
	defer c.lfmbMutex.Unlock()

	var latest = c.LatestFinalizedMagicBlock

	if latest != nil && latest.MagicBlock != nil &&
		latest.MagicBlock.MagicBlockNumber == b.MagicBlock.MagicBlockNumber-1 &&
		latest.MagicBlock.Hash != b.MagicBlock.PreviousMagicBlockHash {

		Logger.DPanic(fmt.Sprintf("failed to set finalized magic block -- "+
			"hashes don't match up: chain's finalized block hash %v, block's"+
			" magic block previous hash %v",
			c.LatestFinalizedMagicBlock.Hash,
			b.MagicBlock.PreviousMagicBlockHash))
	}

	c.LatestFinalizedMagicBlock = b
	c.magicBlockStartingRounds[b.MagicBlock.StartingRound] = b
	c.lfmbSummary = b.GetSummary()
}

func (c *Chain) GetLatestFinalizedMagicBlock() *block.Block {
	c.lfmbMutex.RLock()
	defer c.lfmbMutex.RUnlock()
	return c.LatestFinalizedMagicBlock
}

// GetLatestFinalizedBlockSummary - get the latest finalized block summary.
func (c *Chain) GetLatestFinalizedMagicBlockSummary() *block.BlockSummary {
	c.lfmbMutex.RLock()
	defer c.lfmbMutex.RUnlock()
	return c.lfmbSummary
}

func (c *Chain) GetNodesPreviousInfo(mb *block.MagicBlock) {
	prevMB := c.GetPrevMagicBlockFromMB(mb)
	for key, miner := range mb.Miners.CopyNodesMap() {
		if old := prevMB.Miners.GetNode(key); old != nil {
			miner.SetNodeInfo(old)
			if miner.ProtocolStats == nil {
				ms := &MinerStats{}
				ms.GenerationCountByRank = make([]int64, c.NumGenerators)
				ms.FinalizationCountByRank = make([]int64, c.NumGenerators)
				ms.VerificationTicketsByRank = make([]int64, c.NumGenerators)
			}
		}
	}
	for key, sharder := range mb.Sharders.CopyNodesMap() {
		if old := prevMB.Sharders.GetNode(key); old != nil {
			sharder.SetNodeInfo(old)
		}
	}
}

func (c *Chain) Stop() {
	if stateDB != nil {
		stateDB.Flush()
	}
}

// PruneRoundStorage pruning storage
func (c *Chain) PruneRoundStorage(_ context.Context, getTargetCount func(storage round.RoundStorage) int,
	storages ...round.RoundStorage) {

	for _, storage := range storages {
		targetCount := getTargetCount(storage)
		if targetCount == 0 {
			Logger.Debug("prune storage -- skip. disabled")
			continue
		}
		countRounds := storage.Count()
		if countRounds > targetCount {
			round := storage.GetRounds()[countRounds-targetCount-1]
			if err := storage.Prune(round); err != nil {
				Logger.Error("failed to prune storage",
					zap.Int("count_rounds", countRounds),
					zap.Int("count_dest_prune", targetCount),
					zap.Int64("round", round),
					zap.Error(err))
			} else {
				Logger.Debug("prune storage", zap.Int64("round", round))
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

// The ViewChanger represents node makes view change where a block with new
// magic block finalized.
type ViewChanger interface {
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
