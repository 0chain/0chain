package chain

import (
	"container/ring"
	"context"
	"fmt"
	"math"
	"sync"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/state"
	"0chain.net/transaction"
	"0chain.net/util"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

//PreviousBlockUnavailable - to indicate an error condition when the previous block of a given block is not available
const PreviousBlockUnavailable = "previous_block_unavailable"

//ErrPreviousBlockUnavailable - error for previous block is not available
var ErrPreviousBlockUnavailable = common.NewError(PreviousBlockUnavailable, "Previous block is not available")

const (
	NOTARIZED = 1
	FINALIZED = 2
)

const (
	AllMiners  = 11
	Generator  = 12
	Generators = 13
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
	UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity)
	UpdateFinalizedBlock(ctx context.Context, b *block.Block)
}

/*Chain - data structure that holds the chain data*/
type Chain struct {
	datastore.IDField
	datastore.VersionField
	datastore.CreationDateField
	OwnerID       datastore.Key `json:"owner_id"`                  // Client who created this chain
	ParentChainID datastore.Key `json:"parent_chain_id,omitempty"` // Chain from which this chain is forked off

	Decimals         int8  `json:"decimals"`           // Number of decimals allowed for the token on this chain
	BlockSize        int32 `json:"block_size"`         // Number of transactions in a block
	NumGenerators    int   `json:"num_generators"`     // Number of block generators
	NumSharders      int   `json:"num_sharders"`       // Number of sharders that can store the block
	ThresholdByCount int   `json:"threshold_by_count"` // Threshold count for a block to be notarized
	ThresholdByStake int   `json:"threshold_by_stake"` // Stake threshold for a block to be notarized

	/*Miners - this is the pool of miners */
	Miners *node.Pool `json:"-"`

	/*Sharders - this is the pool of sharders */
	Sharders *node.Pool `json:"-"`

	/*Blobbers - this is the pool of blobbers */
	Blobbers *node.Pool `json:"-"`

	GenesisBlockHash string `json:"genesis_block_hash"`

	blocksMutex *sync.Mutex
	/* This is a cache of blocks that may include speculative blocks */
	Blocks               map[datastore.Key]*block.Block `json:"-"`
	LatestFinalizedBlock *block.Block                   `json:"latest_finalized_block,omitempty"` // Latest block on the chain the program is aware of
	CurrentRound         int64                          `json:"-"`
	CurrentMagicBlock    *block.Block                   `json:"-"`

	BlocksToSharder       int `json:"blocks_to_sharder"`
	VerificationTicketsTo int `json:"verification_tickets_to"`

	StateDB                 util.NodeDB `json:"-"`
	stateMutex              *sync.Mutex
	ClientStateDeserializer state.DeserializerI `json:"-"`

	FinalizedRoundsChannel chan *round.Round `json:"-"`
	MissedBlocks           int64             `json:"-"`

	ZeroNotarizedBlocksCount  int64 `json:"-"`
	MultiNotarizedBlocksCount int64 `json:"-"`

	ValidationBatchSize int   `json:"validation_size"`
	RoundRange          int64 `json:"round_range"`

	BlockChain    *ring.Ring `json:"-"`
	TxnMaxPayload int        `json:"transaction_max_payload"`

	/*PruneStateBelowCount - prune state below these many rounds */
	PruneStateBelowCount int
	minersStake          map[datastore.Key]int
	stakeMutex           *sync.Mutex
}

var chainEntityMetadata *datastore.EntityMetadataImpl

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
	chain.BlockSize = viper.GetInt32("server_chain.block.size")
	chain.NumGenerators = viper.GetInt("server_chain.block.generators")
	chain.NumSharders = viper.GetInt("server_chain.block.sharders")
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
	return chain
}

/*Provider - entity provider for chain object */
func Provider() datastore.Entity {
	c := &Chain{}
	c.Initialize()
	c.Version = "1.0"
	c.blocksMutex = &sync.Mutex{}
	c.stateMutex = &sync.Mutex{}
	c.stakeMutex = &sync.Mutex{}
	c.InitializeCreationDate()
	c.Miners = node.NewPool(node.NodeTypeMiner)
	c.Sharders = node.NewPool(node.NodeTypeSharder)
	c.Blobbers = node.NewPool(node.NodeTypeBlobber)
	return c
}

/*Initialize - intializes internal datastructures to start again */
func (c *Chain) Initialize() {
	c.Blocks = make(map[string]*block.Block)
	c.CurrentRound = 0
	c.LatestFinalizedBlock = nil
	c.CurrentMagicBlock = nil
	c.BlocksToSharder = 1
	c.VerificationTicketsTo = AllMiners
	c.ValidationBatchSize = 2000
	c.FinalizedRoundsChannel = make(chan *round.Round, 128)
	c.ClientStateDeserializer = &state.Deserializer{}
	c.StateDB = stateDB
	c.BlockChain = ring.New(10000)
	c.minersStake = make(map[datastore.Key]int)
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
	db, err := util.NewPNodeDB("data/rocksdb/state")
	if err != nil {
		panic(err)
	}
	stateDB = db
}

func (c *Chain) getInitialState() util.Serializable {
	tokens := viper.GetInt64("server_chain.tokens")
	balance := &state.State{}
	var cents int64 = 1
	for i := int8(0); i < c.Decimals; i++ {
		cents *= 10
	}
	balance.Balance = state.Balance(tokens * cents)
	return balance
}

/*setupInitialState - setup the initial state based on configuration */
func (c *Chain) setupInitialState() util.MerklePatriciaTrieI {
	pmt := util.NewMerklePatriciaTrie(c.StateDB)
	pmt.Insert(util.Path(c.OwnerID), c.getInitialState())
	pmt.SaveChanges(c.StateDB, 0, false)
	Logger.Info("initial state root", zap.Any("hash", util.ToHex(pmt.GetRoot())))
	return pmt
}

/*GenerateGenesisBlock - Create the genesis block for the chain */
func (c *Chain) GenerateGenesisBlock(hash string) (*round.Round, *block.Block) {
	c.GenesisBlockHash = hash
	gb := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	gb.Hash = hash
	gb.Round = 0
	gb.ClientState = c.setupInitialState()
	gb.SetStateStatus(block.StateSuccessful)
	gb.SetBlockState(block.StateNotarized)
	gb.ClientStateHash = gb.ClientState.GetRoot()
	gr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
	gr.Number = 0
	gr.Block = gb
	gr.AddNotarizedBlock(gb)
	return gr, gb
}

/*AddGenesisBlock - adds the genesis block to the chain */
func (c *Chain) AddGenesisBlock(b *block.Block) {
	if b.Round != 0 {
		return
	}
	c.LatestFinalizedBlock = b // Genesis block is always finalized
	c.CurrentMagicBlock = b    // Genesis block is always a magic block
	c.Blocks[b.Hash] = b
	return
}

/*AddBlock - adds a block to the cache */
func (c *Chain) AddBlock(b *block.Block) *block.Block {
	if b.Round < c.LatestFinalizedBlock.Round {
		return nil
	}
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	return c.addBlock(b)
}

func (c *Chain) addBlock(b *block.Block) *block.Block {
	if eb, ok := c.Blocks[b.Hash]; ok {
		return eb
	}
	c.Blocks[b.Hash] = b
	if b.PrevBlock == nil {
		if pb, ok := c.Blocks[b.PrevHash]; ok {
			b.SetPreviousBlock(pb)
		}
	}
	return b
}

/*GetBlock - returns a known block for a given hash from the cache */
func (c *Chain) GetBlock(ctx context.Context, hash string) (*block.Block, error) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	b, ok := c.Blocks[datastore.ToKey(hash)]
	if ok {
		return b, nil
	}
	return nil, common.NewError(datastore.EntityNotFound, fmt.Sprintf("Block with hash (%v) not found", hash))
}

/*DeleteBlock - delete a block from the cache */
func (c *Chain) DeleteBlock(ctx context.Context, b *block.Block) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	if _, ok := c.Blocks[b.Hash]; !ok {
		return
	}
	delete(c.Blocks, b.Hash)
}

/*GetRoundBlocks - get the blocks for a given round */
func (c *Chain) GetRoundBlocks(round int64) []*block.Block {
	blocks := make([]*block.Block, 0, 1)
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	for _, b := range c.Blocks {
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
	blocks := make([]*block.Block, 0, 1)
	for _, b := range c.Blocks {
		if b.Round < round && b.CreationDate < ts {
			Logger.Debug("found block to delete", zap.Int64("round", round), zap.Int64("block_round", b.Round), zap.Int64("current_round", c.CurrentRound), zap.Int64("lf_round", c.LatestFinalizedBlock.Round))
			blocks = append(blocks, b)
		}
	}
	Logger.Info("delete blocks below round", zap.Int64("round", c.CurrentRound), zap.Int64("below_round", round), zap.Any("before", ts), zap.Int("total", len(c.Blocks)), zap.Int("count", len(blocks)))
	for _, b := range blocks {
		b.Clear()
		delete(c.Blocks, b.Hash)
	}

}

/*DeleteBlocks - delete a list of blocks */
func (c *Chain) DeleteBlocks(blocks []*block.Block) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	for _, b := range blocks {
		b.Clear()
		delete(c.Blocks, b.Hash)
	}
}

/*ValidateMagicBlock - validate the block for a given round has the right magic block */
func (c *Chain) ValidateMagicBlock(ctx context.Context, b *block.Block) bool {
	//TODO: This needs to take the round number into account and go backwards as needed to validate
	return b.MagicBlockHash == c.CurrentMagicBlock.Hash
}

/*CanGenerateRound - checks if the miner can generate a block in the given round */
func (c *Chain) CanGenerateRound(r *round.Round, miner *node.Node) bool {
	return r.GetMinerRank(miner.SetIndex)+1 <= c.NumGenerators
}

/*GetGenerators - get all the block generators for a given round */
func (c *Chain) GetGenerators(r *round.Round) []*node.Node {
	generators := make([]*node.Node, c.NumGenerators)
	i := 0
	for _, node := range c.Miners.Nodes {
		if r.GetMinerRank(node.SetIndex) < c.NumGenerators {
			generators[i] = node
			i++
		}
	}
	return generators
}

/*CanStoreBlock - checks if the sharder can store the block in the given round */
func (c *Chain) CanStoreBlock(r *round.Round, sharder *node.Node) bool {
	return r.GetSharderRank(sharder.SetIndex)+1 <= c.NumSharders
}

/*ValidGenerator - check whether this block is from a valid generator */
func (c *Chain) ValidGenerator(r *round.Round, b *block.Block) bool {
	miner := c.Miners.GetNode(b.MinerID)
	if miner == nil {
		return false
	}
	return c.CanGenerateRound(r, miner)
}

/*GetNotarizationThresholdCount - gives the threshold count for block to be notarized*/
func (c *Chain) GetNotarizationThresholdCount() int {
	notarizedPercent := float64(c.ThresholdByCount) / 100
	thresholdCount := float64(c.Miners.Size()) * notarizedPercent
	return int(math.Ceil(thresholdCount))
}

/*CanStartNetwork - check whether the network can start */
func (c *Chain) CanStartNetwork() bool {
	active := c.Miners.GetActiveCount()
	threshold := c.GetNotarizationThresholdCount()
	if config.DevConfiguration.State {
		threshold = c.Miners.Size()
	}
	return active >= threshold
}

/*ReadNodePools - read the node pools from configuration */
func (c *Chain) ReadNodePools(configFile string) {
	nodeConfig := viper.New()
	nodeConfig.AddConfigPath("./config")
	nodeConfig.SetConfigName(configFile)
	err := nodeConfig.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	config := nodeConfig.Get("miners")
	if miners, ok := config.([]interface{}); ok {
		c.Miners.AddNodes(miners)
		c.Miners.ComputeProperties()
		c.InitializeMinerPool()
	}
	config = nodeConfig.Get("sharders")
	if sharders, ok := config.([]interface{}); ok {
		c.Sharders.AddNodes(sharders)
		c.Sharders.ComputeProperties()
	}
	config = nodeConfig.Get("blobbers")
	if blobbers, ok := config.([]interface{}); ok {
		c.Blobbers.AddNodes(blobbers)
		c.Blobbers.ComputeProperties()
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
		if cb.CreationDate < txn.CreationDate {
			return false, nil
		}
	}
	if false {
		Logger.Debug("chain has txn", zap.Int64("round", b.Round), zap.Int64("upto_round", pb.Round), zap.Any("txn_ts", txn.CreationDate), zap.Any("upto_block_ts", pb.CreationDate))
	}
	return false, common.NewError("insufficient_chain", "Chain length not sufficient to confirm the presence of this transaction")
}

func (c *Chain) updateMiningStake(minerId datastore.Key, stake int) {
	c.stakeMutex.Lock()
	defer c.stakeMutex.Unlock()
	c.minersStake[minerId] = stake
}

func (c *Chain) getMiningStake(minerId datastore.Key) int {
	return c.minersStake[minerId]
}

//InitializeMinerPool - initialize the miners after their configuration is read
func (c *Chain) InitializeMinerPool() {
	for _, nd := range c.Miners.Nodes {
		ms := &MinerStats{}
		ms.FinalizationCountByRank = make([]int64, c.NumGenerators, c.NumGenerators)
		nd.ProtocolStats = ms
	}
}
