package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ChainTestSuite struct {
	suite.Suite
}

func TestChainTestSuite(t *testing.T) {
	suite.Run(t, &ChainTestSuite{})
}


func (suite *ChainTestSuite) TestAddBlock() {

	c := NewChainFromConfig()
	b := block.Provider().(*block.Block)

	c.AddBlock(b)
}

func (suite *ChainTestSuite) TestChain_AddBlockNoPrevious() {

	c := NewChainFromConfig()
	b := block.Provider().(*block.Block)
	
	c.AddBlockNoPrevious(b)
	
}

func (suite *ChainTestSuite) TestChain_AddGenesisBlock() {
	c := NewChainFromConfig()
	b := block.Provider().(*block.Block)

	c.AddGenesisBlock(b)
}

func (suite *ChainTestSuite) TestAddLoadedFinalizedBlocks() {
	c := NewChainFromConfig()
	b := block.Provider().(*block.Block)
	b1 := block.Provider().(*block.Block)

	c.AddLoadedFinalizedBlocks(b, b1)
}

func (suite *ChainTestSuite) TestAddNotarizedBlockToRound() {
	c := NewChainFromConfig()
	b := block.Provider().(*block.Block)
	r := round.NewRound(5)

	c.AddNotarizedBlockToRound(r, b)
}

func (suite *ChainTestSuite) TestAddRound() {
	c := NewChainFromConfig()
	r := round.NewRound(5)

	c.AddRound(r)
}

func (suite *ChainTestSuite) TestAddRoundBlock() {
	c := NewChainFromConfig()
	b := block.Provider().(*block.Block)
	r := round.NewRound(5)

	c.AddRoundBlock(r, b)
}

func (suite *ChainTestSuite) TestAreAllNodesActive() {
	c := NewChainFromConfig()

	c.AreAllNodesActive()
}

func (suite *ChainTestSuite) TestCanReplicateBlock() {
	c := NewChainFromConfig()
	b := block.Provider().(*block.Block)

	c.CanReplicateBlock(b)
}

func (suite *ChainTestSuite) TestCanShardBlockWithReplicators() {
	c := NewChainFromConfig()
	//b := block.Provider().(*block.Block)
	n, _ := node.NewNode(map[interface{}]interface{}{})

	c.CanShardBlockWithReplicators(5, "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509", n)
}

//func TestChain_CanShardBlocks(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		nRound int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.CanShardBlocks(tt.args.nRound); got != tt.want {
//				t.Errorf("CanShardBlocks() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

//func TestChain_CanShardBlocksSharders(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		sharders *node.Pool
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.CanShardBlocksSharders(tt.args.sharders); got != tt.want {
//				t.Errorf("CanShardBlocksSharders() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_CanStartNetwork(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.CanStartNetwork(); got != tt.want {
//				t.Errorf("CanStartNetwork() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_ChainHasTransaction(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//		b   *block.Block
//		txn *transaction.Transaction
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    bool
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			got, err := c.ChainHasTransaction(tt.args.ctx, tt.args.b, tt.args.txn)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("ChainHasTransaction() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if got != tt.want {
//				t.Errorf("ChainHasTransaction() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_Delete(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if err := c.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
//				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestChain_DeleteBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//		b   *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_DeleteBlocks(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		blocks []*block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_DeleteBlocksBelowRound(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		round int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_DeleteRound(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//		r   round.RoundI
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_DeleteRoundsBelow(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx         context.Context
//		roundNumber int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_GenerateGenesisBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		hash              string
//		genesisMagicBlock *block.MagicBlock
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   round.RoundI
//		want1  *block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			got, got1 := c.GenerateGenesisBlock(tt.args.hash, tt.args.genesisMagicBlock)
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GenerateGenesisBlock() got = %v, want %v", got, tt.want)
//			}
//			if !reflect.DeepEqual(got1, tt.want1) {
//				t.Errorf("GenerateGenesisBlock() got1 = %v, want %v", got1, tt.want1)
//			}
//		})
//	}
//}
//
//func TestChain_GetBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx  context.Context
//		hash string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    *block.Block
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			got, err := c.GetBlock(tt.args.ctx, tt.args.hash)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("GetBlock() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetBlock() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetBlockSharders(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b *block.Block
//	}
//	tests := []struct {
//		name         string
//		fields       fields
//		args         args
//		wantSharders []string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if gotSharders := c.GetBlockSharders(tt.args.b); !reflect.DeepEqual(gotSharders, tt.wantSharders) {
//				t.Errorf("GetBlockSharders() = %v, want %v", gotSharders, tt.wantSharders)
//			}
//		})
//	}
//}
//
//func TestChain_GetBlockStateNode(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		block *block.Block
//		path  string
//	}
//	tests := []struct {
//		name     string
//		fields   fields
//		args     args
//		wantSeri util.Serializable
//		wantErr  bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			mc := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			gotSeri, err := mc.GetBlockStateNode(tt.args.block, tt.args.path)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("GetBlockStateNode() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(gotSeri, tt.wantSeri) {
//				t.Errorf("GetBlockStateNode() gotSeri = %v, want %v", gotSeri, tt.wantSeri)
//			}
//		})
//	}
//}
//
//func TestChain_GetConfigInfoDB(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetConfigInfoDB(); got != tt.want {
//				t.Errorf("GetConfigInfoDB() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetConfigInfoStore(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   datastore.Store
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetConfigInfoStore(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetConfigInfoStore() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetCurrentMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *block.MagicBlock
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetCurrentMagicBlock(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetCurrentMagicBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetCurrentRound(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   int64
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetCurrentRound(); got != tt.want {
//				t.Errorf("GetCurrentRound() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetEntityMetadata(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   datastore.EntityMetadata
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetGenerationTimeout(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   int
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetGenerationTimeout(); got != tt.want {
//				t.Errorf("GetGenerationTimeout() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetGenerators(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		r round.RoundI
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   []*node.Node
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetGenerators(tt.args.r); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetGenerators() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetLatestFinalizedBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetLatestFinalizedBlock(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetLatestFinalizedBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetLatestFinalizedBlockSummary(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *block.BlockSummary
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetLatestFinalizedBlockSummary(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetLatestFinalizedBlockSummary() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetLatestFinalizedMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetLatestFinalizedMagicBlock(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetLatestFinalizedMagicBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetLatestFinalizedMagicBlockSummary(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *block.BlockSummary
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetLatestFinalizedMagicBlockSummary(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetLatestFinalizedMagicBlockSummary() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetLatestMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *block.MagicBlock
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetLatestMagicBlock(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetLatestMagicBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		round int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   *block.MagicBlock
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetMagicBlock(tt.args.round); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetMagicBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetMiners(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		round int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   *node.Pool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetMiners(tt.args.round); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetMiners() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetNodesPreviousInfo(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		mb *block.MagicBlock
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_GetNotarizationThresholdCount(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		minersNumber int
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   int
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetNotarizationThresholdCount(tt.args.minersNumber); got != tt.want {
//				t.Errorf("GetNotarizationThresholdCount() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetPrevMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		round int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   *block.MagicBlock
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetPrevMagicBlock(tt.args.round); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetPrevMagicBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetPrevMagicBlockFromMB(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		mb *block.MagicBlock
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantPmb *block.MagicBlock
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if gotPmb := c.GetPrevMagicBlockFromMB(tt.args.mb); !reflect.DeepEqual(gotPmb, tt.wantPmb) {
//				t.Errorf("GetPrevMagicBlockFromMB() = %v, want %v", gotPmb, tt.wantPmb)
//			}
//		})
//	}
//}
//
//func TestChain_GetPruneStats(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   *util.PruneStats
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetPruneStats(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetPruneStats() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetRetryWaitTime(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   int
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetRetryWaitTime(); got != tt.want {
//				t.Errorf("GetRetryWaitTime() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetRound(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		roundNumber int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   round.RoundI
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetRound(tt.args.roundNumber); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetRound() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetRoundBlocks(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		round int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   []*block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetRoundBlocks(tt.args.round); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetRoundBlocks() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetRoundTimeoutCount(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   int64
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetRoundTimeoutCount(); got != tt.want {
//				t.Errorf("GetRoundTimeoutCount() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetSignatureScheme(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   encryption.SignatureScheme
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetSignatureScheme(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetSignatureScheme() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_GetUnrelatedBlocks(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		maxBlocks int
//		b         *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   []*block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.GetUnrelatedBlocks(tt.args.maxBlocks, tt.args.b); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetUnrelatedBlocks() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_HasClientStateStored(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		clientStateHash util.Key
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.HasClientStateStored(tt.args.clientStateHash); got != tt.want {
//				t.Errorf("HasClientStateStored() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_IncrementRoundTimeoutCount(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_InitBlockState(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b *block.Block
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if err := c.InitBlockState(tt.args.b); (err != nil) != tt.wantErr {
//				t.Errorf("InitBlockState() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestChain_Initialize(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_InitializeMinerPool(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		mb *block.MagicBlock
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_IsActiveInChain(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.IsActiveInChain(); got != tt.want {
//				t.Errorf("IsActiveInChain() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_IsBlockSharder(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b       *block.Block
//		sharder *node.Node
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.IsBlockSharder(tt.args.b, tt.args.sharder); got != tt.want {
//				t.Errorf("IsBlockSharder() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_IsBlockSharderFromHash(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		nRound  int64
//		bHash   string
//		sharder *node.Node
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.IsBlockSharderFromHash(tt.args.nRound, tt.args.bHash, tt.args.sharder); got != tt.want {
//				t.Errorf("IsBlockSharderFromHash() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_LatestOwnFinalizedBlockRound(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   int64
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.LatestOwnFinalizedBlockRound(); got != tt.want {
//				t.Errorf("LatestOwnFinalizedBlockRound() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_PruneChain(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		in0 context.Context
//		b   *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_PruneRoundStorage(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		in0            context.Context
//		getTargetCount func(storage round.RoundStorage) int
//		storages       []round.RoundStorage
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_Read(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//		key datastore.Key
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if err := c.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
//				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestChain_ReadNodePools(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		configFile string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_ResetRoundTimeoutCount(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetAfterFetcher(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		afr AfterFetcher
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetCurrentRound(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		round int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetFetchedNotarizedBlockHandler(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		fnbh FetchedNotarizedBlockHandler
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetGenerationTimeout(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		newTimeout int
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetLatestDeterministicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetLatestFinalizedBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetLatestFinalizedMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetLatestOwnFinalizedBlockRound(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		r int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		mb *block.MagicBlock
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetMagicBlockSaver(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		mbs MagicBlockSaver
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetRandomSeed(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		r          round.RoundI
//		randomSeed int64
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.SetRandomSeed(tt.args.r, tt.args.randomSeed); got != tt.want {
//				t.Errorf("SetRandomSeed() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_SetRetryWaitTime(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		newWaitTime int
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetRoundRank(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		r round.RoundI
//		b *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetSignatureScheme(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		sigScheme string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetViewChanger(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		vcr ViewChanger
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetupConfigInfoDB(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_SetupNodes(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		mb *block.MagicBlock
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_Stop(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_UpdateMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		newMagicBlock *block.MagicBlock
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if err := c.UpdateMagicBlock(tt.args.newMagicBlock); (err != nil) != tt.wantErr {
//				t.Errorf("UpdateMagicBlock() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestChain_UpdateNodesFromMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		newMagicBlock *block.MagicBlock
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestChain_ValidGenerator(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		r round.RoundI
//		b *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.ValidGenerator(tt.args.r, tt.args.b); got != tt.want {
//				t.Errorf("ValidGenerator() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_Validate(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if err := c.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
//				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestChain_ValidateMagicBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//		mr  *round.Round
//		b   *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.ValidateMagicBlock(tt.args.ctx, tt.args.mr, tt.args.b); got != tt.want {
//				t.Errorf("ValidateMagicBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_Write(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if err := c.Write(tt.args.ctx); (err != nil) != tt.wantErr {
//				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestChain_addBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   *block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.addBlock(tt.args.b); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("addBlock() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_addBlockNoPrevious(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		b *block.Block
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   *block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.addBlockNoPrevious(tt.args.b); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("addBlockNoPrevious() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_callViewChange(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx context.Context
//		lfb *block.Block
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if err := c.callViewChange(tt.args.ctx, tt.args.lfb); (err != nil) != tt.wantErr {
//				t.Errorf("callViewChange() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestChain_getBlock(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		ctx  context.Context
//		hash string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    *block.Block
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			got, err := c.getBlock(tt.args.ctx, tt.args.hash)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("getBlock() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("getBlock() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_getBlocks(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   []*block.Block
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.getBlocks(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("getBlocks() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_getInitialState(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   util.Serializable
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.getInitialState(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("getInitialState() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_getMiningStake(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		minerID datastore.Key
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   int
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.getMiningStake(tt.args.minerID); got != tt.want {
//				t.Errorf("getMiningStake() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_setupInitialState(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   util.MerklePatriciaTrieI
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//			if got := c.setupInitialState(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("setupInitialState() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestChain_updateMiningStake(t *testing.T) {
//	type fields struct {
//		IDField                      datastore.IDField
//		VersionField                 datastore.VersionField
//		CreationDateField            datastore.CreationDateField
//		mutexViewChangeMB            sync.RWMutex
//		Config                       *Config
//		MagicBlockStorage            round.RoundStorage
//		PreviousMagicBlock           *block.MagicBlock
//		mbMutex                      sync.RWMutex
//		LatestFinalizedMagicBlock    *block.Block
//		lfmbMutex                    sync.RWMutex
//		lfmbSummary                  *block.BlockSummary
//		latestOwnFinalizedBlockRound int64
//		blocks                       map[datastore.Key]*block.Block
//		blocksMutex                  *sync.RWMutex
//		rounds                       map[int64]round.RoundI
//		roundsMutex                  *sync.RWMutex
//		CurrentRound                 int64
//		FeeStats                     transaction.TransactionFeeStats
//		LatestFinalizedBlock         *block.Block
//		lfbMutex                     sync.RWMutex
//		lfbSummary                   *block.BlockSummary
//		LatestDeterministicBlock     *block.Block
//		clientStateDeserializer      state.DeserializerI
//		stateDB                      util.NodeDB
//		stateMutex                   *sync.RWMutex
//		finalizedRoundsChannel       chan round.RoundI
//		finalizedBlocksChannel       chan *block.Block
//		Stats                        *Stats
//		BlockChain                   *ring.Ring
//		minersStake                  map[datastore.Key]int
//		stakeMutex                   *sync.Mutex
//		nodePoolScorer               node.PoolScorer
//		GenerateTimeout              int
//		genTimeoutMutex              *sync.Mutex
//		retry_wait_time              int
//		retry_wait_mutex             *sync.Mutex
//		blockFetcher                 *BlockFetcher
//		crtCount                     int64
//		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
//		viewChanger                  ViewChanger
//		afterFetcher                 AfterFetcher
//		magicBlockSaver              MagicBlockSaver
//		pruneStats                   *util.PruneStats
//		configInfoDB                 string
//		configInfoStore              datastore.Store
//		RoundF                       round.RoundFactory
//		magicBlockStartingRounds     map[int64]*block.Block
//		getLFBTicket                 chan *LFBTicket
//		updateLFBTicket              chan *LFBTicket
//		broadcastLFBTicket           chan *block.Block
//		subLFBTicket                 chan chan *LFBTicket
//		unsubLFBTicket               chan chan *LFBTicket
//		lfbTickerWorkerIsDone        chan struct{}
//		phaseEvents                  chan PhaseEvent
//	}
//	type args struct {
//		minerID datastore.Key
//		stake   int
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				IDField:                      tt.fields.IDField,
//				VersionField:                 tt.fields.VersionField,
//				CreationDateField:            tt.fields.CreationDateField,
//				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
//				Config:                       tt.fields.Config,
//				MagicBlockStorage:            tt.fields.MagicBlockStorage,
//				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
//				mbMutex:                      tt.fields.mbMutex,
//				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
//				lfmbMutex:                    tt.fields.lfmbMutex,
//				lfmbSummary:                  tt.fields.lfmbSummary,
//				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
//				blocks:                       tt.fields.blocks,
//				blocksMutex:                  tt.fields.blocksMutex,
//				rounds:                       tt.fields.rounds,
//				roundsMutex:                  tt.fields.roundsMutex,
//				CurrentRound:                 tt.fields.CurrentRound,
//				FeeStats:                     tt.fields.FeeStats,
//				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
//				lfbMutex:                     tt.fields.lfbMutex,
//				lfbSummary:                   tt.fields.lfbSummary,
//				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
//				clientStateDeserializer:      tt.fields.clientStateDeserializer,
//				stateDB:                      tt.fields.stateDB,
//				stateMutex:                   tt.fields.stateMutex,
//				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
//				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
//				Stats:                        tt.fields.Stats,
//				BlockChain:                   tt.fields.BlockChain,
//				minersStake:                  tt.fields.minersStake,
//				stakeMutex:                   tt.fields.stakeMutex,
//				nodePoolScorer:               tt.fields.nodePoolScorer,
//				GenerateTimeout:              tt.fields.GenerateTimeout,
//				genTimeoutMutex:              tt.fields.genTimeoutMutex,
//				retry_wait_time:              tt.fields.retry_wait_time,
//				retry_wait_mutex:             tt.fields.retry_wait_mutex,
//				blockFetcher:                 tt.fields.blockFetcher,
//				crtCount:                     tt.fields.crtCount,
//				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
//				viewChanger:                  tt.fields.viewChanger,
//				afterFetcher:                 tt.fields.afterFetcher,
//				magicBlockSaver:              tt.fields.magicBlockSaver,
//				pruneStats:                   tt.fields.pruneStats,
//				configInfoDB:                 tt.fields.configInfoDB,
//				configInfoStore:              tt.fields.configInfoStore,
//				RoundF:                       tt.fields.RoundF,
//				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
//				getLFBTicket:                 tt.fields.getLFBTicket,
//				updateLFBTicket:              tt.fields.updateLFBTicket,
//				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
//				subLFBTicket:                 tt.fields.subLFBTicket,
//				unsubLFBTicket:               tt.fields.unsubLFBTicket,
//				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
//				phaseEvents:                  tt.fields.phaseEvents,
//			}
//		})
//	}
//}
//
//func TestCloseStateDB(t *testing.T) {
//	tests := []struct {
//		name string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//		})
//	}
//}
//
//func TestGetServerChain(t *testing.T) {
//	tests := []struct {
//		name string
//		want *Chain
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := GetServerChain(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("GetServerChain() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestNewChainFromConfig(t *testing.T) {
//	tests := []struct {
//		name string
//		want *Chain
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := NewChainFromConfig(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewChainFromConfig() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestProvider(t *testing.T) {
//	tests := []struct {
//		name string
//		want datastore.Entity
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := Provider(); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("Provider() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestResetStatusMonitor(t *testing.T) {
//	type args struct {
//		round int64
//	}
//	tests := []struct {
//		name string
//		args args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//		})
//	}
//}
//
//func TestSetServerChain(t *testing.T) {
//	type args struct {
//		c *Chain
//	}
//	tests := []struct {
//		name string
//		args args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//		})
//	}
//}
//
//func TestSetupEntity(t *testing.T) {
//	type args struct {
//		store datastore.Store
//	}
//	tests := []struct {
//		name string
//		args args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//		})
//	}
//}
//
//func TestSetupStateDB(t *testing.T) {
//	tests := []struct {
//		name string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//		})
//	}
//}
//
//func Test_collectNodes(t *testing.T) {
//	type args struct {
//		mbs []*block.MagicBlock
//	}
//	tests := []struct {
//		name     string
//		args     args
//		wantKeep map[string]struct{}
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if gotKeep := collectNodes(tt.args.mbs...); !reflect.DeepEqual(gotKeep, tt.wantKeep) {
//				t.Errorf("collectNodes() = %v, want %v", gotKeep, tt.wantKeep)
//			}
//		})
//	}
//}
//
//func Test_getNodePath(t *testing.T) {
//	type args struct {
//		path string
//	}
//	tests := []struct {
//		name string
//		args args
//		want util.Path
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := getNodePath(tt.args.path); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("getNodePath() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_mbRoundOffset(t *testing.T) {
//	type args struct {
//		rn int64
//	}
//	tests := []struct {
//		name string
//		args args
//		want int64
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := mbRoundOffset(tt.args.rn); got != tt.want {
//				t.Errorf("mbRoundOffset() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
