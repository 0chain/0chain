package sharder

import (
	"context"
	"strconv"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/persistencestore"
	. "github.com/0chain/common/core/logging"

	"go.uber.org/zap"
)

// BlockSummaries -
type BlockSummaries struct {
	datastore.IDField
	BSummaryList []*block.BlockSummary `json:"block_summaries"`
}

var blockSummariesEntityMetadata *datastore.EntityMetadataImpl

// NewBlockSummaries - create a new BlockSummaries entity
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
			if sc.IsBlockSharder(db, node.Self.Underlying()) {
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
		Logger.Error("get block summary", zap.Error(err))
		return nil, err
	}
	if len(blockSummary.Hash) < 64 {
		Logger.Error("get block summary - hash of block in summary is less than 64", zap.Any("block_summary", blockSummary))
	}
	Logger.Debug("get block summary", zap.Any("block_summary", blockSummary))
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
func (sc *Chain) StoreBlockSummaryFromBlock(b *block.Block) error {
	bs := b.GetSummary()
	bSummaryEntityMetadata := bs.GetEntityMetadata()
	bctx := ememorystore.WithEntityConnection(common.GetRootContext(), bSummaryEntityMetadata)
	defer ememorystore.Close(bctx)
	if len(bs.Hash) < 64 {
		Logger.Error("Writing block summary - block hash less than 64", zap.Any("hash", bs.Hash))
	}
	err := bs.Write(bctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(bctx, bSummaryEntityMetadata)
	return con.Commit()
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

/*StoreMagicBlockMapFromBlock - stores magic block number mapped to the block hash */
func (sc *Chain) StoreMagicBlockMapFromBlock(mbm *block.MagicBlockMap) error {
	mbMapEntityMetadata := mbm.GetEntityMetadata()
	mctx := persistencestore.WithEntityConnection(common.GetRootContext(), mbMapEntityMetadata)
	defer persistencestore.Close(mctx)
	if len(mbm.Hash) < 64 {
		Logger.Error("Writing block summary - block hash less than 64", zap.Any("hash", mbm.Hash), zap.Any("magic_block_number", mbm.ID))
	}
	return mbMapEntityMetadata.GetStore().Write(mctx, mbm)
}

/*GetMagicBlockMap - given a magic block number, get the magic block map */
func (sc *Chain) GetMagicBlockMap(ctx context.Context, magicBlockNumber string) (*block.MagicBlockMap, error) {
	magicBlockMapEntityMetadata := datastore.GetEntityMetadata("magic_block_map")
	magicBlockMap := magicBlockMapEntityMetadata.Instance().(*block.MagicBlockMap)
	mctx := persistencestore.WithEntityConnection(ctx, magicBlockMapEntityMetadata)
	defer persistencestore.Close(mctx)
	err := magicBlockMapEntityMetadata.GetStore().Read(mctx, datastore.ToKey(magicBlockNumber), magicBlockMap)
	if err != nil {
		return nil, err
	}
	return magicBlockMap, nil
}

// GetHighestMagicBlockMap returns highest stored MB map. The highest means with
// greatest MB number. It works with Cassandra only.
func (sc *Chain) GetHighestMagicBlockMap(ctx context.Context) (
	mbm *block.MagicBlockMap, err error) {

	var mbmemd = datastore.GetEntityMetadata("magic_block_map")
	mbm = mbmemd.Instance().(*block.MagicBlockMap)

	var mctx = persistencestore.WithEntityConnection(ctx, mbmemd)
	defer persistencestore.Close(mctx)

	const query = `SELECT MAX(id) FROM zerochain.magic_block_map;`

	var (
		cql    = persistencestore.GetCon(mctx)
		number int64
	)

	if err = cql.Query(query).Scan(&number); err != nil {
		return nil, common.NewErrorf("get_highest_mbm",
			"scanning CQL result: %v", err)
	}

	var mbn = strconv.FormatInt(number, 10)
	err = mbmemd.GetStore().Read(mctx, datastore.ToKey(mbn), mbm)
	if err != nil {
		return nil, common.NewErrorf("get_highest_mbm",
			"getting latest MB map: %v", err)
	}

	return
}
