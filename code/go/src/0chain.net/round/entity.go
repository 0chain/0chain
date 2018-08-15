package round

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"

	"0chain.net/ememorystore"

	"0chain.net/block"
	"0chain.net/datastore"
)

const (
	RoundGenerating               = 0
	RoundGenerated                = 1
	RoundCollectingBlockProposals = 2
	RoundStateFinalizing          = 3
	RoundStateFinalized           = 4
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

	minerPerm   []int
	sharderPerm []int
	state       int

	notarizedBlocks      []*block.Block
	notarizedBlocksMutex *sync.Mutex
}

var roundEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (r *Round) GetEntityMetadata() datastore.EntityMetadata {
	return roundEntityMetadata
}

/*GetKey - returns the round number as the key */
func (r *Round) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", r.Number))
}

/*AddNotarizedBlock - this will be concurrent as notarization is recognized by verifying as well as notarization message from others */
func (r *Round) AddNotarizedBlock(b *block.Block) bool {
	r.notarizedBlocksMutex.Lock()
	defer r.notarizedBlocksMutex.Unlock()
	for _, blk := range r.notarizedBlocks {
		if blk.Hash == b.Hash {
			return false
		}
	}
	r.notarizedBlocks = append(r.notarizedBlocks, b)
	return true
}

/*GetNotarizedBlocks - return all the notarized blocks associated with this round */
func (r *Round) GetNotarizedBlocks() []*block.Block {
	rnb := r.notarizedBlocks
	sort.Slice(rnb, func(i int, j int) bool { return rnb[i].ChainWeight > rnb[j].ChainWeight })
	return r.notarizedBlocks
}

/*SetFinalizing - the round is being finalized */
func (r *Round) SetFinalizing() bool {
	r.notarizedBlocksMutex.Lock()
	defer r.notarizedBlocksMutex.Unlock()
	if r.IsFinalized() || r.IsFinalizing() {
		return false
	}
	r.state = RoundStateFinalizing
	return true
}

/*IsFinalizing - is the round finalizing */
func (r *Round) IsFinalizing() bool {
	return r.state == RoundStateFinalizing
}

/*Finalize - finalize the round */
func (r *Round) Finalize(b *block.Block) {
	r.state = RoundStateFinalized
	r.Block = b
	r.BlockHash = b.Hash
}

/*IsFinalized - indicates if the round is finalized */
func (r *Round) IsFinalized() bool {
	return r.state == RoundStateFinalized || r.Number == 0
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	r := &Round{}
	r.notarizedBlocks = make([]*block.Block, 0, 1)
	r.notarizedBlocksMutex = &sync.Mutex{}
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

//SetupRoundSummaryDB - setup the round summary db
func SetupRoundSummaryDB() {
	db, err := ememorystore.CreateDB("data/rocksdb/roundsummary")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("roundsummarydb", db)
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

/*ComputeRanks - Compute random order of n elements given the random see of the round
NOTE: The permutation is deterministic using a PRNG that uses a starting seed. The starting seed itself
      is crytgraphically generated random number and is not known till the threshold signature is reached.
*/
func (r *Round) ComputeRanks(m int, s int) {
	r.minerPerm = rand.New(rand.NewSource(r.RandomSeed)).Perm(m)
	r.sharderPerm = rand.New(rand.NewSource(r.RandomSeed)).Perm(s)
}

/*GetMinerRank - get the rank of element at the elementIdx position based on the permutation of the round */
func (r *Round) GetMinerRank(elementIdx int) int {
	return r.minerPerm[elementIdx]
}

/*GetSharderRank - get the rank of element at the elementIdx position based on the permutation of the round */
func (r *Round) GetSharderRank(elementIdx int) int {
	return r.sharderPerm[elementIdx]
}
