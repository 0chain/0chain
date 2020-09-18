package chain

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/core/cache"
	"0chain.net/core/config"
)

// block fetcher internal
type hashRound struct {
	next  string // hash of next block (or empty string) to set previous block
	hash  string // hash of block to fetch
	round int64  // round of the block to fetch (needed to fetch from sharders)
}

type BlockFetcher struct {
	cache             cache.Cache    // are fetching now
	fetchBlock        chan hashRound //
	fetchFromSharders chan hashRound // internal, fallback fetching
}

func NewBlockFetcher() (bf *BlockFetcher) {

	// short hands
	var (
		fm    = config.AsyncBlocksFetchingMaxSimultaneousFromMiners()
		fs    = config.AsyncBlocksFetchingMaxSimultaneousFromSharders()
		total = fm + fs
	)

	// the block fetcher
	bf = new(BlockFetcher)
	bf.cache = cache.NewLRUCache(total)
	bf.fetchBlock = make(chan hashRound, fm)
	bf.fetchFromSharders = make(chan hashRound, fs)
	return
}

func (bf *BlockFetcher) isFetching(hash string) bool {
	_, err := bf.cache.Get(hash)
	return err == nil
}

func (bf *BlockFetcher) AsyncFetchBlock(next, hash string, rn int64) {
	if bf.isFetching(hash) {
		return
	}
	bf.fetchBlock <- hashRound{next: next, hash: hash, rn: rn}
}

func (bf *BlockFetcher) BlockFetchWorker(ctx context.Context) {

	//

	for {
		//
	}
}

/*

// AsyncFetchPreviousBlock - fetch previous block asynchronously.
func (bf *BlockFetcher) AsyncFetchPreviousBlock(b *block.Block) {
	if bf.IsFetching(b.PrevHash) {
		return
	}
	bf.missingLinkBlocks <- b
}

// AsyncFetchBlock - fetch the block asynchronously.
func (bf *BlockFetcher) AsyncFetchBlock(hash string, round int64) {
	if bf.IsFetching(hash) {
		return
	}
	bf.missingBlocks <- hashRound{hash: hash, round: round}
}

// FetchPreviousBlock - fetch the previous block.
func (bf *BlockFetcher) FetchPreviousBlock(ctx context.Context, c *Chain, b *block.Block) {
	if !bf.IsFetching(b.PrevHash) {
		bf.fblocks.Add(b.PrevHash, true)
		go c.GetPreviousBlock(ctx, b)
	}
}

// FetchBlock - fetch the block.
func (bf *BlockFetcher) FetchBlock(ctx context.Context, c *Chain, hash string,
	round int64) {

	if !c.blockFetcher.IsFetching(hash) {
		c.blockFetcher.fblocks.Add(hash, true)
		go c.GetNotarizedBlock(ctx, hash, round)
	}
}

*/

// FetchedNotarizedBlockHandler - a handler that processes a fetched
// notarized block.
type FetchedNotarizedBlockHandler interface {
	NotarizedBlockFetched(ctx context.Context, b *block.Block)
}

// ViewChanger represents node makes view change where a block
// with new magic block finalized.
type ViewChanger interface {
	ViewChange(ctx context.Context, lfb *block.Block) (err error)
}
