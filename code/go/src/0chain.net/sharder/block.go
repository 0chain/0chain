package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/datastore"
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

/*GetBlockSummary - given a block hash, get the block summary */
func GetBlockSummary(ctx context.Context, hash string) (*block.BlockSummary, error) {
	blockSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	blockSummary := blockSummaryEntityMetadata.Instance().(*block.BlockSummary)
	err := blockSummaryEntityMetadata.GetStore().Read(ctx, datastore.ToKey(hash), blockSummary)
	if err != nil {
		return nil, err
	}
	return blockSummary, nil
}
