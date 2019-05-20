package sharder

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	. "0chain.net/core/logging"

	"go.uber.org/zap"
)

type BlockSummaries struct {
	datastore.IDField
	BSummaryList []*block.BlockSummary `json:block_summaries`
}

var blockSummariesEntityMetadata *datastore.EntityMetadataImpl

/*NewBlockSummaries - create a new BlockSummaries entity */
func NewBlockSummaries() *BlockSummaries {
	bs := datastore.GetEntityMetadata("block_summaries").Instance().(*BlockSummaries)
	return bs
}

/*BlockSummariesProvider - a block summaries instance provider */
func BlockSummariesProvider() datastore.Entity {
	bs := &BlockSummaries{}
	return bs
}

/*GetEntityMetadata - implement interface */
func (bs *BlockSummaries) GetEntityMetadata() datastore.EntityMetadata {
	return blockSummariesEntityMetadata
}

/*SetupBlockSummaries - setup the block summaries entity */
func SetupBlockSummaries() {
	blockSummariesEntityMetadata = datastore.MetadataProvider()
	blockSummariesEntityMetadata.Name = "block_summaries"
	blockSummariesEntityMetadata.Provider = BlockSummariesProvider
	blockSummariesEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("block_summaries", blockSummariesEntityMetadata)
}

/*GetBlockBySummary - get a block */
func (sc *Chain) GetBlockBySummary(ctx context.Context, bs *block.BlockSummary) (*block.Block, error) {
	if len(bs.Hash) < 64 {
		Logger.Error("Hash from block summary is less than 64", zap.Any("block_summary", bs))
	}
	//Try to get the block from the cache
	b, err := sc.GetBlock(ctx, bs.Hash)
	if err != nil {
		bi, err := GetSharderChain().BlockTxnCache.Get(bs.Hash)
		if err != nil {
			db := datastore.GetEntityMetadata("block").Instance().(*block.Block)
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
func (sc *Chain) GetBlockSummary(ctx context.Context, hash string) (*block.BlockSummary, error) {
	blockSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	blockSummary := blockSummaryEntityMetadata.Instance().(*block.BlockSummary)
	err := blockSummaryEntityMetadata.GetStore().Read(ctx, datastore.ToKey(hash), blockSummary)
	if err != nil {
		return nil, err
	}
	if len(blockSummary.Hash) < 64 {
		Logger.Error("Reading block summary - hash of block in summary is less than 64", zap.Any("block_summary", blockSummary))
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

/*StoreBlockSummaryFromBlock - gets block summary from block and stores it to ememory/rocksdb */
func (sc *Chain) StoreBlockSummaryFromBlock(ctx context.Context, b *block.Block) error {
	bs := b.GetSummary()
	bSummaryEntityMetadata := bs.GetEntityMetadata()
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	defer ememorystore.Close(bctx)
	if len(bs.Hash) < 64 {
		Logger.Error("Writing block summary - block hash less than 64", zap.Any("hash", bs.Hash))
	}
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

/*StoreBlockSummary - stores block summary to ememory/rocksdb */
func (sc *Chain) StoreBlockSummary(ctx context.Context, bs *block.BlockSummary) error {
	bSummaryEntityMetadata := bs.GetEntityMetadata()
	bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
	defer ememorystore.Close(bctx)
	if len(bs.Hash) < 64 {
		Logger.Error("Writing block summary - block hash less than 64", zap.Any("hash", bs.Hash))
	}
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
