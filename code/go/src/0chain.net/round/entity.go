package round

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"

	"0chain.net/ememorystore"
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
	Number     int64 `json:"number"`
	RandomSeed int64 `json:"round_random_seed"`

	SelfRandomFunctionValue int64 `json:"-"`

	// For generator, this is the block the miner is generating till a notraization is received
	// For a verifier, this is the block that is currently the best block received for verification.
	// Once a round is finalized, this is the finalized block of the given round
	Block     *block.Block `json:"-"`
	BlockHash string       `json:"block_hash"`

	minerPerm []int
	state     int

	notarizedBlocks []*block.Block
	Mutex           *sync.Mutex

	shares map[string]*VRFShare
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
	r.RandomSeed = seed
	r.SetState(RoundVRFComplete)
}

//GetRandomSeed - returns the random seed of the round
func (r *Round) GetRandomSeed() int64 {
	return r.RandomSeed
}

/*AddNotarizedBlock - this will be concurrent as notarization is recognized by verifying as well as notarization message from others */
func (r *Round) AddNotarizedBlock(b *block.Block) (*block.Block, bool) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	for _, blk := range r.notarizedBlocks {
		if blk.Hash == b.Hash {
			if blk != b {
				blk.MergeVerificationTickets(b.VerificationTickets)
			}
			return blk, false
		}
	}
	b.SetBlockNotarized()
	r.notarizedBlocks = append(r.notarizedBlocks, b)
	return b, true
}

/*GetNotarizedBlocks - return all the notarized blocks associated with this round */
func (r *Round) GetNotarizedBlocks() []*block.Block {
	return r.notarizedBlocks
}

/*GetBestNotarizedBlock - get the best notarized block that we have */
func (r *Round) GetBestNotarizedBlock() *block.Block {
	rnb := r.notarizedBlocks
	if len(rnb) == 0 {
		return nil
	}
	if len(rnb) == 1 {
		return rnb[0]
	}
	sort.Slice(rnb, func(i int, j int) bool { return rnb[i].ChainWeight > rnb[j].ChainWeight })
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
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.isFinalizing()
}

func (r *Round) isFinalizing() bool {
	return r.state == RoundStateFinalizing
}

/*IsFinalized - indicates if the round is finalized */
func (r *Round) IsFinalized() bool {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.isFinalized()
}

func (r *Round) isFinalized() bool {
	return r.state == RoundStateFinalized || r.GetRoundNumber() == 0
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	r := &Round{}
	r.notarizedBlocks = make([]*block.Block, 0, 1)
	r.Mutex = &sync.Mutex{}
	r.shares = make(map[string]*VRFShare)
	return r
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
func (r *Round) ComputeMinerRanks(m int) {
	r.minerPerm = rand.New(rand.NewSource(r.RandomSeed)).Perm(m)
}

/*GetMinerRank - get the rank of element at the elementIdx position based on the permutation of the round */
func (r *Round) GetMinerRank(miner *node.Node) int {
	return r.minerPerm[miner.SetIndex]
}

/*GetMinersByRank - get the rnaks of the miners */
func (r *Round) GetMinersByRank(miners *node.Pool) []*node.Node {
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

//AddVRFShare - implement interface
func (r *Round) AddVRFShare(share *VRFShare) bool {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if _, ok := r.shares[share.party.GetKey()]; ok {
		return false
	}
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
