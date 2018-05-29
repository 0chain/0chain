package round

import (
	"0chain.net/block"
	"0chain.net/datastore"
)

/*Round - data structure for the round */
type Round struct {
	Number int64
	Role   int

	// For generator, this is the block the miner is generating till a consensus is reached
	// For a verifier, this is the block that is currently the best block received for verification.
	// Once a consensus is reached and finalized, this is the finalized block of the given round
	Block *block.Block

	// All blocks in a given round
	blocks map[datastore.Key]*block.Block
}

/*RoleGenerator - block genreator role */
var RoleGenerator = 1

/*RoleVerifier - block verifier role */
var RoleVerifier = 2

/*AddBlock - adds a block to the round. Assumes non-concurrent update */
func (r *Round) AddBlock(b *block.Block) {
	r.blocks[b.Hash] = b
}
