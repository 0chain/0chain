package chain

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/config"
)

// common fetching errors
var (
	ErrBlockFetchQueueFull = common.NewError("block_fetcher",
		"queue full")
	ErrBlockFetchMinersQueueFull = common.NewError("block_fetcher",
		"miners queue full")
	ErrBlockFetchShardersQueueFull = common.NewError("block_fetcher",
		"sharders queue full")
)

type BlockFetchReply struct {
	Hash  string       // hash of the block requested, used internally
	Block *block.Block // block, if given
	Err   error        // error on failure
}

// block fetcher internal
type blockFetchRequest struct {
	next  string // hash of next block (or empty string) to set previous block
	hash  string // hash of block to fetch
	round int64  // round of the block to fetch (needed to fetch from sharders)

	sharders  bool   // force to fetch from sharders
	sharderID string // sharder ID to fetch from (try first from this sharder)

	replies []chan BlockFetchReply // fetching reply
}

type BlockFetcher struct {
	fetchBlock chan blockFetchRequest //

	fetchFromMiners   chan blockFetchRequest // internal, main fetching channel
	fetchFromSharders chan blockFetchRequest // internal, fallback fetching
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
	bf.fetchBlock = make(chan blockFetchRequest, total)
	bf.fetchFromMiners = make(chan blockFetchRequest, 0)
	bf.fetchFromSharders = make(chan blockFetchRequest, 0)
	return
}

// AsyncFetchBlock downloads block by hash and round from miners or sharders.
func (bf *BlockFetcher) AsyncFetchBlock(next, hash string, rn int64) {
	if bf.isFetching(hash) {
		return
	}
	bf.fetchBlock <- blockFetchRequest{next: next, hash: hash, rn: rn}
}

// The terminate is response with given error.
func (bf *BlockFetcher) terminate(ctx context.Context, bfr *blockFetchRequest,
	err error) {

	for _, rp := range bfr.replies {
		select {
		case rp <- BlockFetchReply{Hash: bfr.hash, Err: err}:
		case <-ctx.Done():
		}
	}
}

// The respond replies with given block.
func (bf *BlockFetcher) respond(ctx context.Context, bfr *blockFetchRequest,
	b *block.Block) {

	for _, rp := range bfr.replies {
		select {
		case rp <- BlockFetchReply{Hash: bfr.hash, Block: b}:
		case <-ctx.Done():
		}
	}
}

// BlockFetchWorker used to fetch blocks from other nodes. The BlockFetchWorker
// depends on LFBTickets worker.
func (bf *BlockFetcher) BlockFetchWorker(ctx context.Context,
	slfbt LFBTicketer) {

	var (
		// configurations
		fm    = config.AsyncBlocksFetchingMaxSimultaneousFromMiners()
		fs    = config.AsyncBlocksFetchingMaxSimultaneousFromSharders()
		total = fm + fs

		// main channels
		quit  = ctx.Done()
		fetch = bf.fetchBlock

		// internal mapping and replies
		fetching = make(map[string]blockFetchRequest, cap(bf.fetchBlock))
		got      = make(chan BlockFetchReply)

		// track latest round known by sharders
		tickets = slfbt.SubLFBTicket(ctx) // can block
		tk      *LFBTicket                // internal
		latest  int64                     // latest given LFB ticket

		// limits
		minersl   = make(chan struct{}, fm)
		shardersl = make(chan struct{}, fs)
	)

	for {
		select {

		// terminate all pending requests and quit when the context is done
		case <-quit:
			// terminate all fetchers with error canceled
			for _, bfr := range fetching {
				bf.terminate(ctx, bfr, context.Canceled)
			}
			return

		// update latest round known by sharders
		case tk = tickets:
			latest = tk.Round // update latest sharders round

		// handle block fetch requests
		case bfr := <-fetch:
			var have, ok = fetching[bfr.hash]
			if ok {
				have.replies = append(have.replies, bfr.replies...)
				continue
			}

			if len(fetching) >= total {
				bf.terminate(ctx, bfr, ErrBlockFetchQueueFull)
				continue
			}

			fetching[bfr.hash] = bfr // add, increasing map length

			// is force from sharders
			if bfr.sharders {
				select {
				case bf.fetchFromSharders <- bfr:
				default:
					// TODO (sfxdx): busy
				}
				continue
			}

		case rpl := <-got:
			//
		}
	}
}

func (bf *BlockFetcher) minersFetchWorker(ctx context.Context) {

	var (
		// configurations
		fm = config.AsyncBlocksFetchingMaxSimultaneousFromMiners()

		// main channels
		quit        = ctx.Done()              //
		fetchMiners = bf.fetchFromMiners      //
		minersl     = make(chan struct{}, fm) // limits
	)

	for {
		select {
		case <-quit:
			return
		case bfr := <-fetchMiners:
			select {
			case minersl <- struct{}{}:
				//
			default:
				bf.terminate(ctx, bfr, ErrBlockFetchMinersQueueFull)
			}
		}
	}

}

func (br *BlockFetcher) shardersFetchWorker(ctx context.Context) {
	//
}

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

// LFBTicketer used to subscribe to new LFB tickets and get latest once.
type LFBTicketer interface {
	SubLFBTicket(ctx context.Context) (sub chan *LFBTicket)
	GetLatestLFBTicket(ctx context.Context) (tk *LFBTicket)
}
