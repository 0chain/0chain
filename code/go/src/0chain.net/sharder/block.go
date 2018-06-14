package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/blockstore"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/persistencestore"
	"go.uber.org/zap"
)

/*StoreBlock - store the block to persistence storage */
func StoreBlock(ctx context.Context, b *block.Block) error {
	err := b.Validate(ctx)
	if err != nil {
		Logger.Error("block validation", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
		return err
	}
	err = blockstore.GetStore().Write(b)
	if err != nil {
		Logger.Error("block save", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	} else {
		Logger.Info("saved block", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Any("prev_hash", b.PrevHash))
	}

	// TODO: Store the block summary and transaction summary information
	bs := datastore.GetEntityMetadata("block_summary").Instance().(*block.BlockSummary)
	bs.Hash = b.Hash
	bs.RoundRandomSeed = b.RoundRandomSeed
	bs.PrevHash = b.PrevHash
	bs.Round = b.Round
	ctx = persistencestore.WithEntityConnection(ctx, bs.GetEntityMetadata())
	store := persistencestore.GetStorageProvider()
	err = store.Write(ctx, bs)
	if err != nil {
		Logger.Error("can't write to DB", zap.Any("Error", err.Error()))
	}
	return err
}
