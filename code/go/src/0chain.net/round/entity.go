package round

import (
	"fmt"
	"math/rand"

	"0chain.net/block"
	"0chain.net/datastore"
)

/*Round - data structure for the round */
type Round struct {
	datastore.NOIDField
	Number int64 `json:"number"`

	RandomSeed int64 `json:"round_random_seed"`

	SelfRandomFunctionValue int64 `json:"-"`

	// For generator, this is the block the miner is generating till a consensus is reached
	// For a verifier, this is the block that is currently the best block received for verification.
	// Once a consensus is reached and finalized, this is the finalized block of the given round
	Block *block.Block `json:"-"`

	perm []int

	// All blocks in a given round
	blocks map[datastore.Key]*block.Block
}

var roundEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (r *Round) GetEntityMetadata() datastore.EntityMetadata {
	return roundEntityMetadata
}

/*GetEntityName - implementing the interface */
func (r *Round) GetEntityName() string {
	return "round"
}

/*GetKey - returns the round number as the key */
func (r *Round) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", r.Number))
}

/*AddBlock - adds a block to the round. Assumes non-concurrent update */
func (r *Round) AddBlock(b *block.Block) {
	r.blocks[b.Hash] = b
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	r := &Round{}
	r.blocks = make(map[datastore.Key]*block.Block)
	return r
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	roundEntityMetadata = &datastore.EntityMetadataImpl{Name: "round", Provider: Provider}
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
