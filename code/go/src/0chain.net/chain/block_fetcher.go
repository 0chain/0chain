package chain

import (
	"context"

	"0chain.net/block"
	"0chain.net/cache"
)

//BlockFetcher - to fetch blocks from other nodes
type BlockFetcher struct {
	fblocks           cache.Cache
	missingBlocks     chan string
	missingLinkBlocks chan *block.Block
}

//NewBlockFetcher - create a block fetcher object
func NewBlockFetcher() *BlockFetcher {
	bf := &BlockFetcher{}
	bf.fblocks = cache.NewLRUCache(100)
	bf.missingLinkBlocks = make(chan *block.Block, 128)
	bf.missingBlocks = make(chan string, 128)
	return bf
}

//AsyncFetchPreviousBlock - fetch previous block asynchronously
func (bf *BlockFetcher) AsyncFetchPreviousBlock(b *block.Block) {
	if bf.IsFetching(b.PrevHash) {
		return
	}
	bf.missingLinkBlocks <- b
}

//AsyncFetchBlock - fetch the block asynchronously
func (bf *BlockFetcher) AsyncFetchBlock(hash string) {
	if bf.IsFetching(hash) {
		return
	}
	bf.missingBlocks <- hash
}

//IsFetching - is the block being fetched (determined by cache)
func (bf *BlockFetcher) IsFetching(hash string) bool {
	_, err := bf.fblocks.Get(hash)
	return err == nil
}

//FetchPreviousBlock - fetch the previous block
func (bf *BlockFetcher) FetchPreviousBlock(ctx context.Context, c *Chain, b *block.Block) {
	if !bf.IsFetching(b.PrevHash) {
		bf.fblocks.Add(b.PrevHash, true)
		go c.GetPreviousBlock(ctx, b)
	}
}

//FetchBlock - fetch the block
func (bf *BlockFetcher) FetchBlock(ctx context.Context, c *Chain, hash string) {
	if !c.blockFetcher.IsFetching(hash) {
		c.blockFetcher.fblocks.Add(hash, true)
		go c.GetNotarizedBlock(hash)
	}
}

//FetchedNotarizedBlockHandler - a handler that processes a fetched notarized block
type FetchedNotarizedBlockHandler interface {
	NotarizedBlockFetched(ctx context.Context, b *block.Block)
}
