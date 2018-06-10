package sharder

import (
	"0chain.net/block"
	"0chain.net/blockstore"
)

/*StoreBlock - store the block to persistence storage */
func StoreBlock(b *block.Block) {
	blockstore.GetStore().Write(b)
}
