package chain

import (
	"context"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/node"
	"0chain.net/round"
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
	datastore.CreationDateField
	ClientID      datastore.Key `json:"client_id"`                 // Client who created this chain
	ParentChainID datastore.Key `json:"parent_chain_id,omitempty"` // Chain from which this chain is forked off
	Decimals      int8          `json:"decimals"`                  // Number of decimals allowed for the token on this chain

	/*Miners - this is the pool of miners */
	Miners *node.Pool `json:"-"`

	/*Sharders - this is the pool of sharders */
	Sharders *node.Pool `json:"-"`

	/*Blobbers - this is the pool of blobbers */
	Blobbers *node.Pool `json:"-"`

	RoundsChannel        chan *round.Round `json:"-"`
	LatestFinalizedBlock *block.Block      `json:"latest_finalized_block,omitempty"` // Latest block on the chain the program is aware of

}

/*GetEntityName - implementing the interface */
func (c *Chain) GetEntityName() string {
	return "chain"
}

/*Validate - implementing the interface */
func (c *Chain) Validate(ctx context.Context) error {
	if c.ID == "" {
		return common.InvalidRequest("chain id is required")
	}
	if c.ClientID == "" {
		return common.InvalidRequest("client id is required")
	}
	return nil
}

/*Read - datastore read */
func (c *Chain) Read(ctx context.Context, key datastore.Key) error {
	return datastore.Read(ctx, key, c)
}

/*Write - datastore read */
func (c *Chain) Write(ctx context.Context) error {
	return datastore.Write(ctx, c)
}

/*Delete - datastore read */
func (c *Chain) Delete(ctx context.Context) error {
	return datastore.Delete(ctx, c)
}

/*Provider - entity provider for chain object */
func Provider() interface{} {
	c := &Chain{}
	c.RoundsChannel = make(chan *round.Round)
	c.InitializeCreationDate()
	c.Miners = node.NewPool(node.NodeTypeMiner)
	c.Sharders = node.NewPool(node.NodeTypeSharder)
	c.Blobbers = node.NewPool(node.NodeTypeBlobber)
	return c
}

/*SetupEntity - setup the entity */
func SetupEntity() {
	datastore.RegisterEntityProvider("chain", Provider)
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (c *Chain) UpdateFinalizedBlock(lfb *block.Block) {
	if lfb.Hash == c.LatestFinalizedBlock.Hash {
		return
	}
	ctx := datastore.WithConnection(context.Background())
	for b := lfb; b != nil && b != c.LatestFinalizedBlock; b = b.GetPreviousBlock() {
		b.Finalize(ctx)
	}
}

/*GetRoundsChannel - a channel that provides the round messages */
func (c *Chain) GetRoundsChannel() chan *round.Round {
	return c.RoundsChannel
}
