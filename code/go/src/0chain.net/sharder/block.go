package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/ememorystore"
	"0chain.net/node"
)

/*GetBlockBySummary - get a block */
func (sc *Chain) GetBlockBySummary(ctx context.Context, bs *block.BlockSummary) (*block.Block, error) {
	//Try to get the block from the cache
	b, err := sc.GetBlock(ctx, bs.Hash)
	if err != nil {
		bi, err := GetSharderChain().BlockTxnCache.Get(bs.Hash)
		if err != nil {
			db := &block.Block{}
			db.Hash = bs.Hash
			db.Round = bs.Round
			if sc.IsBlockSharder(db, node.Self.Node) {
				b, err = sc.GetBlockFromStoreBySummary(bs)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, common.NewError("block_not_available", "Block not available")
			}
		} else {
			b = bi.(*block.Block)
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

/*GetBlockFromHash - given the block hash, get the block */
func (sc *Chain) GetBlockFromHash(ctx context.Context, hash string, roundNum int64) (*block.Block, error) {
	b, err := sc.GetBlock(ctx, hash)
	if err != nil {
		b, err = sc.GetBlockFromStore(hash, roundNum)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

/*StoreBlockSummary - store the block to ememory/rocksdb */
func (sc *Chain) StoreBlockSummary(ctx context.Context, b *block.Block) error {
	bs := b.GetSummary()
	bSummaryEntityMetadata := bs.GetEntityMetadata()
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	defer ememorystore.Close(bctx)
	err := bs.Write(bctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(bctx, bSummaryEntityMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}
