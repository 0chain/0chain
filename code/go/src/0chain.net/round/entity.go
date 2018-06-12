package round

import (
	"context"
	"fmt"
	"math/rand"
	"sort"

	"0chain.net/block"
	"0chain.net/datastore"
	"0chain.net/node"
)

/*Round - data structure for the round */
type Round struct {
	datastore.NOIDField
	Number int64 `json:"number"`

	RandomSeed int64 `json:"round_random_seed"`

	SelfRandomFunctionValue int64 `json:"-"`

	// For generator, this is the block the miner is generating till a notraization is received
	// For a verifier, this is the block that is currently the best block received for verification.
	// Once a notraization is received and finalized, this is the finalized block of the given round
	Block *block.Block `json:"-"`

	perm []int

	// All blocks in a given round
	blocks map[datastore.Key]*block.Block

	blocksToVerifyChannel chan *block.Block

	verificationCancelf context.CancelFunc

	verificationComplete bool
	finalized            bool
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

/*AddBlock - adds a block to the round. Assumes non-concurrent update */
func (r *Round) AddBlock(b *block.Block) {
	if r.verificationComplete {
		return
	}
	if r.Number != b.Round {
		return
	}
	if r.Number == 0 {
		r.Block = b
		r.blocks[b.Hash] = b
		return
	}
	b.RoundRandomSeed = r.RandomSeed
	bNode := node.GetNode(b.MinerID)
	//TODO: view change in the middle of a round will throw off the SetIndex
	b.RoundRank = r.GetRank(bNode.SetIndex)
	r.blocksToVerifyChannel <- b
}

/*IsVerificationComplete - indicates if the verification process for the round is complete */
func (r *Round) IsVerificationComplete() bool {
	return r.verificationComplete
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
	r.blocks = make(map[datastore.Key]*block.Block)
	r.blocksToVerifyChannel = make(chan *block.Block, 200)
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

/*GetBlocksByRank - return the currently stored blocks in the order of best rank for the round */
func (r *Round) GetBlocksByRank(blocks []*block.Block) []*block.Block {
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].RoundRank < blocks[j].RoundRank })
	return blocks
}

/*GetBlocksToVerifyChannel - a channel where all the blocks requiring verification are put into */
func (r *Round) GetBlocksToVerifyChannel() chan *block.Block {
	return r.blocksToVerifyChannel
}

/*CollectionFunc - function to start collecting blocks
* TODO: clean up with better design?
* As we can't have circular dependency between miner and round packages, this is the workaround
 */
type CollectionFunc func(ctx context.Context, r *Round)

/*CollectionBlocks - an interface that starts collecting and verifying blocks */
type CollectBlocks interface {
	CollectionBlocksForVerification(ctx context.Context, r *Round)
}

/*StartVerificationBlockCollection - WARNING: Doesn't support concurrent calling */
func (r *Round) StartVerificationBlockCollection(ctx context.Context, collectionf CollectionFunc) {
	if r.verificationCancelf != nil {
		return
	}
	lctx, cancelf := context.WithCancel(ctx)
	r.verificationCancelf = cancelf
	go collectionf(lctx, r)
}

/*CancelVerification - Cancel verification of blocks */
func (r *Round) CancelVerification() {
	if r.verificationComplete {
		return
	}
	r.verificationComplete = true
	if r.verificationCancelf != nil {
		r.verificationCancelf()
	}
}
