package round

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"sort"
	"sync"

	"0chain.net/ememorystore"
	. "0chain.net/logging"
	"0chain.net/node"

	"0chain.net/block"
	"0chain.net/datastore"
)

const (
	RoundShareVRF                  = 0
	RoundVRFComplete               = iota
	RoundGenerating                = iota
	RoundGenerated                 = iota
	RoundCollectingBlockProposals  = iota
	RoundStateVerificationTimedOut = iota
	RoundStateFinalizing           = iota
	RoundStateFinalized            = iota
)

/*Round - data structure for the round */
type Round struct {
	datastore.NOIDField
	Number        int64 `json:"number"`
	RandomSeed    int64 `json:"round_random_seed"`
	hasRandomSeed bool

	SelfRandomFunctionValue int64 `json:"-"`

	// For generator, this is the block the miner is generating till a notraization is received
	// For a verifier, this is the block that is currently the best block received for verification.
	// Once a round is finalized, this is the finalized block of the given round
	Block     *block.Block `json:"-"`
	BlockHash string       `json:"block_hash"`
	VRFOutput string       `json:"vrf_output"` //TODO: VRFOutput == rbooutput?
	minerPerm []int
	state     int

	proposedBlocks  []*block.Block
	notarizedBlocks []*block.Block
	Mutex           sync.RWMutex

	shares map[string]*VRFShare
}

//NewRound - Create a new round object
func NewRound(round int64) *Round {
	r := datastore.GetEntityMetadata("round").Instance().(*Round)
	r.Number = round
	return r
}

var roundEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (r *Round) GetEntityMetadata() datastore.EntityMetadata {
	return roundEntityMetadata
}

/*GetKey - returns the round number as the key */
func (r *Round) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", r.GetRoundNumber()))
}

//GetRoundNumber - returns the round number
func (r *Round) GetRoundNumber() int64 {
	return r.Number
}

//SetRandomSeed - set the random seed of the round
func (r *Round) SetRandomSeed(seed int64) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.hasRandomSeed {
		return
	}
	r.RandomSeed = seed
	r.hasRandomSeed = true
	r.minerPerm = nil
}

//GetRandomSeed - returns the random seed of the round
func (r *Round) GetRandomSeed() int64 {
	return r.RandomSeed
}

// SetVRFOutput --sets the VRFOutput
func (r *Round) SetVRFOutput(rboutput string) {
	r.VRFOutput = rboutput
}

// GetVRFOutput --gets the VRFOutput
func (r *Round) GetVRFOutput() string {
	return r.VRFOutput
}

/*AddNotarizedBlock - this will be concurrent as notarization is recognized by verifying as well as notarization message from others */
func (r *Round) AddNotarizedBlock(b *block.Block) (*block.Block, bool) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	b, _ = r.addProposedBlock(b)
	for _, blk := range r.notarizedBlocks {
		if blk.Hash == b.Hash {
			if blk != b {
				blk.MergeVerificationTickets(b.VerificationTickets)
			}
			return blk, false
		}
	}
	b.SetBlockNotarized()
	if r.Block == nil || r.Block.RoundRank > b.RoundRank {
		r.Block = b
	}
	rnb := append(r.notarizedBlocks, b)
	sort.Slice(rnb, func(i int, j int) bool { return rnb[i].ChainWeight > rnb[j].ChainWeight })
	r.notarizedBlocks = rnb
	return b, true
}

/*GetNotarizedBlocks - return all the notarized blocks associated with this round */
func (r *Round) GetNotarizedBlocks() []*block.Block {
	return r.notarizedBlocks
}

/*AddProposedBlock - this will be concurrent as notarization is recognized by verifying as well as notarization message from others */
func (r *Round) AddProposedBlock(b *block.Block) (*block.Block, bool) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.addProposedBlock(b)
}

func (r *Round) addProposedBlock(b *block.Block) (*block.Block, bool) {
	for _, blk := range r.proposedBlocks {
		if blk.Hash == b.Hash {
			return blk, false
		}
	}
	r.proposedBlocks = append(r.proposedBlocks, b)
	sort.SliceStable(r.proposedBlocks, func(i, j int) bool { return r.proposedBlocks[i].RoundRank < r.proposedBlocks[j].RoundRank })
	return b, true
}

/*GetProposedBlocks - return all the blocks that have been proposed for this round */
func (r *Round) GetProposedBlocks() []*block.Block {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.proposedBlocks
}

/*GetHeaviestNotarizedBlock - get the heaviest notarized block that we have in this round */
func (r *Round) GetHeaviestNotarizedBlock() *block.Block {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	rnb := r.notarizedBlocks
	if len(rnb) == 0 {
		return nil
	}
	return rnb[0]
}

/*GetBlocksByRank - return the currently stored blocks in the order of best rank for the round */
func (r *Round) GetBlocksByRank(blocks []*block.Block) []*block.Block {
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].RoundRank < blocks[j].RoundRank })
	return blocks
}

/*GetBestRankedNotarizedBlock - get the best ranked notarized block for this round */
func (r *Round) GetBestRankedNotarizedBlock() *block.Block {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	rnb := r.notarizedBlocks
	if len(rnb) == 0 {
		return nil
	}
	if len(rnb) == 1 {
		return rnb[0]
	}
	rnb = r.GetBlocksByRank(rnb)
	return rnb[0]
}

/*Finalize - finalize the round */
func (r *Round) Finalize(b *block.Block) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.setState(RoundStateFinalized)
	r.Block = b
	r.BlockHash = b.Hash
}

/*SetFinalizing - the round is being finalized */
func (r *Round) SetFinalizing() bool {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.isFinalized() || r.isFinalizing() {
		return false
	}
	r.setState(RoundStateFinalizing)
	return true
}

/*IsFinalizing - is the round finalizing */
func (r *Round) IsFinalizing() bool {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return r.isFinalizing()
}

func (r *Round) isFinalizing() bool {
	return r.state == RoundStateFinalizing
}

/*IsFinalized - indicates if the round is finalized */
func (r *Round) IsFinalized() bool {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return r.isFinalized()
}

func (r *Round) isFinalized() bool {
	return r.state == RoundStateFinalized || r.GetRoundNumber() == 0
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	r := &Round{}
	r.initialize()
	return r
}

func (r *Round) initialize() {
	r.notarizedBlocks = make([]*block.Block, 0, 1)
	r.proposedBlocks = make([]*block.Block, 0, 3)
	r.shares = make(map[string]*VRFShare)
}

/*Read - read round entity from store */
func (r *Round) Read(ctx context.Context, key datastore.Key) error {
	return r.GetEntityMetadata().GetStore().Read(ctx, key, r)
}

/*Write - write round entity to store */
func (r *Round) Write(ctx context.Context) error {
	return r.GetEntityMetadata().GetStore().Write(ctx, r)
}

/*Delete - delete round entity from store */
func (r *Round) Delete(ctx context.Context) error {
	return r.GetEntityMetadata().GetStore().Delete(ctx, r)
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	roundEntityMetadata = datastore.MetadataProvider()
	roundEntityMetadata.Name = "round"
	roundEntityMetadata.DB = "roundsummarydb"
	roundEntityMetadata.Provider = Provider
	roundEntityMetadata.Store = store
	roundEntityMetadata.IDColumnName = "number"
	datastore.RegisterEntityMetadata("round", roundEntityMetadata)
}

//SetupRoundSummaryDB - setup the round summary db
func SetupRoundSummaryDB() {
	db, err := ememorystore.CreateDB("data/rocksdb/roundsummary")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("roundsummarydb", db)
}

/*ComputeMinerRanks - Compute random order of n elements given the random see of the round
NOTE: The permutation is deterministic using a PRNG that uses a starting seed. The starting seed itself
      is crytgraphically generated random number and is not known till the threshold signature is reached.
*/
func (r *Round) ComputeMinerRanks(miners *node.Pool) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.minerPerm = rand.New(rand.NewSource(r.RandomSeed)).Perm(miners.Size())
}

/*GetMinerRank - get the rank of element at the elementIdx position based on the permutation of the round */
func (r *Round) GetMinerRank(miner *node.Node) int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	if r.minerPerm == nil {
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		Logger.DPanic(fmt.Sprintf("miner ranks not computed yet: %v", r.GetState()))
	}
	return r.minerPerm[miner.SetIndex]
}

/*GetMinersByRank - get the rnaks of the miners */
func (r *Round) GetMinersByRank(miners *node.Pool) []*node.Node {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	nodes := miners.Nodes
	rminers := make([]*node.Node, len(nodes))
	for _, nd := range nodes {
		rminers[r.minerPerm[nd.SetIndex]] = nd
	}
	return rminers
}

//Clear - implement interface
func (r *Round) Clear() {
}

//Restart - restart the round
func (r *Round) Restart() {
	r.initialize()
	r.Block = nil
	r.SetState(RoundShareVRF)
}

//AddVRFShare - implement interface
func (r *Round) AddVRFShare(share *VRFShare) bool {

	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if _, ok := r.shares[share.party.GetKey()]; ok {
		return false
	}

	Logger.Info("Adding Shares from minerId: " + share.party.GetKey())
	r.setState(RoundShareVRF)
	r.shares[share.party.GetKey()] = share
	return true
}

//GetVRFShares - implement interface
func (r *Round) GetVRFShares() map[string]*VRFShare {
	return r.shares
}

//GetState - get the state of the round
func (r *Round) GetState() int {
	return r.state
}

//SetState - set the state of the round
func (r *Round) SetState(state int) {
	r.setState(state)
}

func (r *Round) setState(state int) {
	if state > r.state {
		r.state = state
	}
}

//HasRandomSeed - implement interface
func (r *Round) HasRandomSeed() bool {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.hasRandomSeed
}
