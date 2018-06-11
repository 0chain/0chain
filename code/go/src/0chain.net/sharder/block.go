package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/blockstore"
)

/*StoreBlock - store the block to persistence storage */
func StoreBlock(ctx context.Context, b *block.Block) error {
	err := b.Validate(ctx)
	if err != nil {
		return err
	}
	return blockstore.GetStore().Write(b)
}
