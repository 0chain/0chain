package chain

import (
	"context"
	"fmt"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"go.uber.org/zap"
)

const DELTA = 200 * time.Millisecond
const FINALIZATION_TIME = 2 * DELTA

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
	ClientID      datastore.Key `json:"client_id"`                 // Client who created this chain
	ParentChainID datastore.Key `json:"parent_chain_id,omitempty"` // Chain from which this chain is forked off

	Decimals  int8  `json:"decimals"`   // Number of decimals allowed for the token on this chain
	BlockSize int32 `json:"block_size"` // Number of transactions in a block

	/*Miners - this is the pool of miners */
	Miners *node.Pool `json:"-"`

	/*Sharders - this is the pool of sharders */
	Sharders *node.Pool `json:"-"`

	/*Blobbers - this is the pool of blobbers */
	Blobbers *node.Pool `json:"-"`

	GenesisBlockHash string `json:"genesis_block_hash"`

	/* This is a cache of blocks that may include speculative blocks */
	Blocks               map[datastore.Key]*block.Block `json:"-"`
	LatestFinalizedBlock *block.Block                   `json:"latest_finalized_block,omitempty"` // Latest block on the chain the program is aware of
	CurrentRound         int64
	CurrentMagicBlock    *block.Block
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
	if datastore.IsEmpty(c.ClientID) {
		return common.InvalidRequest("client id is required")
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

/*Provider - entity provider for chain object */
func Provider() datastore.Entity {
	c := &Chain{}
	c.Version = "1.0"
	c.InitializeCreationDate()
	c.Miners = node.NewPool(node.NodeTypeMiner)
	c.Sharders = node.NewPool(node.NodeTypeSharder)
	c.Blobbers = node.NewPool(node.NodeTypeBlobber)
	c.Blocks = make(map[string]*block.Block)
	return c
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	chainEntityMetadata = datastore.MetadataProvider()
	chainEntityMetadata.Name = "chain"
	chainEntityMetadata.Provider = Provider
	chainEntityMetadata.Store = store
	datastore.RegisterEntityMetadata("chain", chainEntityMetadata)
}

/*GenerateGenesisBlock - Create the genesis block for the chain */
func (c *Chain) GenerateGenesisBlock(hash string) (*round.Round, *block.Block) {
	c.GenesisBlockHash = hash
	gb := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	gb.Hash = hash
	gb.Round = 0
	gr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
	gr.Number = 0
	gr.AddNotarizedBlock(gb)
	return gr, gb
}

/*ComputeFinalizedBlock - compute the block that has been finalized. It should be the one in the prior round
TODO: This logic needs refinement when the sharders start saving only partial set of blocks they are responsible for
*/
func (c *Chain) ComputeFinalizedBlock(ctx context.Context, r *round.Round) *block.Block {
	tips := r.GetNotarizedBlocks()
	for true {
		ntips := make([]*block.Block, 0, 1)
		for _, b := range tips {
			if b.Hash == c.LatestFinalizedBlock.Hash {
				break
			}
			found := false
			for _, nb := range ntips {
				if b.PrevHash == nb.Hash {
					found = true
					break
				}
			}
			if found {
				continue
			}
			ntips = append(ntips, b.PrevBlock)
		}
		tips = ntips
		if len(tips) == 1 {
			break
		}
	}
	if len(tips) != 1 {
		return nil
	}
	fb := tips[0]
	if fb.Round == r.Number {
		return nil
	}
	return fb
}

/*FinalizeRound - starting from the given round work backwards and identify the round that can be
  assumed to be finalized as only one chain has survived.
  Note: It is that round and prior that actually get finalized.
*/
func (c *Chain) FinalizeRound(ctx context.Context, r *round.Round, bsh BlockStateHandler) {
	if r.IsFinalizing() || r.IsFinalized() {
		return
	}
	r.Finalizing()
	var finzalizeTimer = time.NewTimer(FINALIZATION_TIME)
	select {
	case <-finzalizeTimer.C:
		break
	}
	fb := c.ComputeFinalizedBlock(ctx, r)
	if fb == nil {
		Logger.Debug("finalization - no decisive block to finalize yet", zap.Any("round", r.Number))
		return
	}
	lfbHash := c.LatestFinalizedBlock.Hash
	c.LatestFinalizedBlock = fb
	frchain := make([]*block.Block, 0, 1)
	for b := fb; b != nil && b.Hash != lfbHash; b = b.PrevBlock {
		frchain = append(frchain, b)
	}
	deadBlocks := make([]*block.Block, 0, 1)
	for idx := range frchain {
		fb = frchain[len(frchain)-1-idx]
		Logger.Debug("finalizing round", zap.Any("round", r.Number), zap.Any("finalized_round", fb.Round), zap.Any("hash", fb.Hash))
		bsh.UpdateFinalizedBlock(ctx, fb)
		frb := c.GetRoundBlocks(fb.Round)
		for _, b := range frb {
			if b.Hash != fb.Hash {
				deadBlocks = append(deadBlocks, b)
			}
		}
	}
	// Prune all the dead blocks
	go func() {
		for _, b := range deadBlocks {
			c.DeleteBlock(ctx, b)
		}
		Logger.Debug("finalize round", zap.Any("round", r.Number), zap.Any("block_cache_size", len(c.Blocks)))
	}()
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
func (c *Chain) AddBlock(b *block.Block) {
	if b.Round <= c.LatestFinalizedBlock.Round {
		return
	}
	c.Blocks[b.Hash] = b
	if b.PrevBlock == nil {
		pb, ok := c.Blocks[b.PrevHash]
		if ok {
			b.PrevBlock = pb
		} else {
			Logger.Debug("previous block not present", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("prev_block", b.PrevHash))
		}
	}
}

/*GetBlock - returns a known block for a given hash from the cache */
func (c *Chain) GetBlock(ctx context.Context, hash string) (*block.Block, error) {
	b, ok := c.Blocks[datastore.ToKey(hash)]
	if ok {
		return b, nil
	}
	/*
		b = block.Provider().(*block.Block)
		err := b.Read(ctx, datastore.ToKey(hash))
		if err != nil {
			return b, nil
		}*/
	return nil, common.NewError(datastore.EntityNotFound, fmt.Sprintf("Block with hash (%v) not found", hash))
}

/*DeleteBlock - delete a block from the cache */
func (c *Chain) DeleteBlock(ctx context.Context, b *block.Block) {
	delete(c.Blocks, b.Hash)
}

/*GetRoundBlocks - get the blocks for a given round */
func (c *Chain) GetRoundBlocks(round int64) []*block.Block {
	blocks := make([]*block.Block, 0, 1)
	for _, blk := range c.Blocks {
		if blk.Round == round {
			blocks = append(blocks, blk)
		}
	}
	return blocks
}

/*VerifyTicket - verify the ticket */
func (c *Chain) VerifyTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) error {
	sender := c.Miners.GetNode(bvt.VerifierID)
	if sender == nil {
		return common.InvalidRequest("Verifier unknown or not authorized at this time")
	}

	if ok, _ := sender.Verify(bvt.Signature, b.Hash); !ok {
		return common.InvalidRequest("Couldn't verify the signature")
	}
	return nil
}

/*VerifyNotarization - verify that the notarization is correct */
func (c *Chain) VerifyNotarization(ctx context.Context, b *block.Block, bvt []*block.VerificationTicket) error {
	if b.Round != 0 && bvt == nil {
		return common.NewError("no_verification_tickets", "No verification tickets for this block")
	}
	// TODO: Logic similar to ReachedNotarization to check the count satisfies (refactor)

	for _, vt := range bvt {
		if err := c.VerifyTicket(ctx, b, vt); err != nil {
			return err
		}
	}
	return nil
}

/*IsBlockNotarized - Does the given number of signatures means eligible for notraization?
TODO: For now, we just assume more than 50% */
func (c *Chain) IsBlockNotarized(ctx context.Context, b *block.Block) bool {
	numSignatures := b.GetVerificationTicketsCount()
	if 3*numSignatures >= 2*c.Miners.Size() {
		return true
	}
	return false
}

/*ValidateMagicBlock - validate the block for a given round has the right magic block */
func (c *Chain) ValidateMagicBlock(ctx context.Context, b *block.Block) bool {
	//TODO: This needs to take the round number into account and go backwards as needed to validate
	return b.MagicBlockHash == c.CurrentMagicBlock.Hash
}
