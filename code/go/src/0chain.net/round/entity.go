package round

import (
	"fmt"
	"math/rand"
	"sync"

	"0chain.net/block"
	"0chain.net/datastore"
)

/*Round - data structure for the round */
type Round struct {
	datastore.NOIDField
	Number     int64 `json:"number"`
	RandomSeed int64 `json:"round_random_seed"`

	SelfRandomFunctionValue int64 `json:"-"`

	// For generator, this is the block the miner is generating till a notraization is received
	// For a verifier, this is the block that is currently the best block received for verification.
	// Once a notraization is received and finalized, this is the finalized block of the given round
	Block *block.Block `json:"-"`

	perm      []int
	finalized bool

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
	for _, blk := range r.notarizedBlocks {
		if blk.Hash == b.Hash {
			return false
		}
	}
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

/*Finalize - finalize the round */
func (r *Round) Finalize() {
	r.finalized = true
}

/*IsFinalized - indicates if the round is finalized */
func (r *Round) IsFinalized() bool {
	return r.finalized
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	r := &Round{}
	r.notarizedBlocks = make([]*block.Block, 0, 1)
	r.notarizedBlocksMutex = &sync.Mutex{}
	return r
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	roundEntityMetadata = datastore.MetadataProvider()
	roundEntityMetadata.Name = "round"
	roundEntityMetadata.Provider = Provider
	roundEntityMetadata.IDColumnName = "number"
	datastore.RegisterEntityMetadata("round", roundEntityMetadata)
}

/*ComputeRanks - Compute random order of n elements given the random see of the round */
func (r *Round) ComputeRanks(n int) {
	r.perm = rand.New(rand.NewSource(r.RandomSeed)).Perm(n)
}

/*GetRank - get the rank of element at the elementIdx position based on the permutation of the round */
func (r *Round) GetRank(elementIdx int) int {
	return r.perm[elementIdx]
}
