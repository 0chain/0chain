package chain

import (
	"context"
	"fmt"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"go.uber.org/zap"
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

/*GenesisBlockHash - block of 0chain.net main chain */
var GenesisBlockHash = "ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae4" //TODO

/*GenerateGenesisBlock - Create the genesis block for the chain */
func (c *Chain) GenerateGenesisBlock() (*round.Round, *block.Block) {
	if c.ID != config.GetMainChainID() {
		// TODO
		return nil, nil
	}
	gb := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	gb.Hash = GenesisBlockHash
	gb.Round = 0
	gr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
	gr.Number = 0
	gr.AddNotarizedBlock(gb)
	return gr, gb
}

/*ComputeFinalizedBlock - compute the block that has been finalized. It should be the one in the prior round */
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

/*AddBlock - adds a block to the cache */
func (c *Chain) AddBlock(b *block.Block) {
	c.Blocks[b.Hash] = b
	if b.Round == 0 {
		c.LatestFinalizedBlock = b // Genesis block is always finalized
		c.CurrentMagicBlock = b    // Genesis block is always a magic block
	} else if b.PrevBlock == nil {
		pb, ok := c.Blocks[b.PrevHash]
		if ok {
			b.PrevBlock = pb
		} else {
			Logger.Error("Prev block not present", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("prev_block", b.PrevHash))
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
