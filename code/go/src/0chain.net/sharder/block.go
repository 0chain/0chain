package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/blockstore"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

/*StoreBlock - store the block to persistence storage */
func StoreBlock(ctx context.Context, b *block.Block) error {
	err := b.Validate(ctx)
	if err != nil {
		Logger.Info("block validation failed", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
		return err
	}
	err = blockstore.GetStore().Write(b)
	if err != nil {
		Logger.Info("block save failed", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Error(err))
	} else {
		Logger.Info("saved block", zap.Any("round", b.Round), zap.Any("hash", b.Hash), zap.Any("prev_hash", b.PrevHash))
	}

	// TODO: Store the block summary and transaction summary information
	return err
}
