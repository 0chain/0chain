package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/chain"
)

/*GetBlockBySummary - get a block */
func (sc *Chain) GetBlockBySummary(ctx context.Context, bs *block.BlockSummary) (*block.Block, error) {
	//Try to get the block from the cache
	b, err := chain.GetServerChain().GetBlock(ctx, bs.Hash)
	if err != nil {
		//TODO: based on round random seed, check whether this sharder should have the block or not before fetching from the store
		b, err = sc.GetBlockFromStoreBySummary(bs)
		if err != nil {
			//We are able to send partial information
			return nil, err
		}
	}
	return b, nil
}
