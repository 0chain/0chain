package chain

import (
	"context"
	"net/url"
	"strconv"
	"sync"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
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

// The FetchQueueStat represents numbers of blocks fetch requests to
// miners and to sharders.
type FetchQueueStat struct {
	Miners   int // number of current fetch block requests to miners
	Sharders int // number of current fetch block requests to sharders
}

type BlockFetchReply struct {
	Hash  string       // hash of the block requested, used internally
	Block *block.Block // block, if given
	Err   error        // error on failure
}

// block fetcher internal
type blockFetchRequest struct {
	hash  string // hash of block to fetch
	round int64  // round of the block to fetch (needed to fetch from sharders)

	sharders  bool   // force to fetch from sharders
	sharderID string // sharder ID to fetch from (try first from this sharder)

	replies []chan BlockFetchReply // fetching reply
}

type BlockFetcher struct {
	fetchBlock chan *blockFetchRequest // requests to fetch
	statq      chan FetchQueueStat     // number of current requests
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
	bf.fetchBlock = make(chan *blockFetchRequest, total)
	bf.statq = make(chan FetchQueueStat)
	return
}

// The terminate responds with given error.
func (bf *BlockFetcher) terminate(ctx context.Context, bfr *blockFetchRequest,
	err error) {

	for _, rp := range bfr.replies {
		select {
		case rp <- BlockFetchReply{Hash: bfr.hash, Err: err}:
		case <-ctx.Done():
		}
	}
}

// The respond responds with given block.
func (bf *BlockFetcher) respond(ctx context.Context, bfr *blockFetchRequest,
	b *block.Block) {

	for _, rp := range bfr.replies {
		select {
		case rp <- BlockFetchReply{Hash: bfr.hash, Block: b}:
		case <-ctx.Done():
		}
	}
}

func (bf *BlockFetcher) acquire(ctx context.Context, limit chan struct{}) (
	ok bool) {

	select {
	case <-ctx.Done():
	case limit <- struct{}{}:
		return true
	}
	return // false
}

func (bf *BlockFetcher) release(limit chan struct{}) {
	<-limit // just drain one value
}

// StartBlockFetchWorker used to fetch blocks from other nodes. The BlockFetchWorker
// depends on LFBTickets worker.
func (bf *BlockFetcher) StartBlockFetchWorker(ctx context.Context,
	chainer Chainer) {

	var (
		// configurations
		fm    = config.AsyncBlocksFetchingMaxSimultaneousFromMiners()
		fs    = config.AsyncBlocksFetchingMaxSimultaneousFromSharders()
		total = fm + fs

		// main channels
		quit  = ctx.Done()
		fetch = bf.fetchBlock
		statq = bf.statq

		// internal mapping and replies
		fetching = make(map[string]*blockFetchRequest, cap(bf.fetchBlock))
		got      = make(chan BlockFetchReply)

		// track latest round known by sharders
		tickets = chainer.SubLFBTicket(ctx) // subscribe to new tickets
		tk      *LFBTicket                  // internal
		latest  int64                       // latest given LFB ticket

		// limits
		minersl   = make(chan struct{}, fm)
		shardersl = make(chan struct{}, fs)

		stat FetchQueueStat
	)

	for {

		stat.Miners, stat.Sharders = len(minersl), len(shardersl)

		select {

		// terminate all pending requests and quit when the context is done
		case <-quit:
			// terminate all fetchers with error canceled
			for _, bfr := range fetching {
				bf.terminate(ctx, bfr, context.Canceled)
			}
			return

		// send statistics
		case statq <- stat:

		// update latest round known by sharders
		case tk = <-tickets:
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

			// if force from sharders
			if bfr.sharders {
				if bf.acquire(ctx, shardersl) {
					fetching[bfr.hash] = bfr
					go bf.fetchFromSharders(ctx, bfr, got, chainer, shardersl)
				} else {
					bf.terminate(ctx, bfr, ErrBlockFetchShardersQueueFull)
				}
				continue
			}

			// fetch from miners first
			if bf.acquire(ctx, minersl) {
				fetching[bfr.hash] = bfr
				go bf.fetchFromMiners(ctx, bfr, got, chainer, minersl)
			} else {
				// don't try to fetch from sharder on miners full queue
				// (that's not a reason to fetch from sharders)
				bf.terminate(ctx, bfr, ErrBlockFetchMinersQueueFull)
			}

		case rpl := <-got:
			// process fetching results
			var bfr, ok = fetching[rpl.Hash]
			if !ok {
				panic("BlockFetcher, invalid state: missing block fetch request")
				continue
			}

			// got the correct response
			if rpl.Block != nil {
				delete(fetching, rpl.Hash)
				bf.respond(ctx, bfr, rpl.Block)
				continue
			}

			// got no block, but error

			// already requested from sharders, so, its the end
			if bfr.sharders {
				delete(fetching, rpl.Hash)
				bf.terminate(ctx, bfr, rpl.Err)
				continue
			}

			// if block round > the latest ticket round, then we shouldn't
			// request it from sharders (it can't be on sharders)
			if bfr.round > 0 && bfr.round > latest {
				delete(fetching, rpl.Hash)
				bf.terminate(ctx, bfr, rpl.Err)
				continue
			}

			// try request sharders for the block (set sharders: true to avoid
			// cyclic sharders requests)
			bfr.sharders = true
			if bf.acquire(ctx, shardersl) {
				go bf.fetchFromSharders(ctx, bfr, got, chainer, shardersl)
			} else {
				bf.terminate(ctx, bfr, ErrBlockFetchShardersQueueFull)
			}
		}
	}
}

func (bf *BlockFetcher) gotError(ctx context.Context, got chan BlockFetchReply,
	hash string, err error) {

	select {
	case <-ctx.Done():
	case got <- BlockFetchReply{Hash: hash, Err: err}:
	}

	return
}

func (bf *BlockFetcher) gotBlock(ctx context.Context, got chan BlockFetchReply,
	b *block.Block) {

	select {
	case <-ctx.Done():
	case got <- BlockFetchReply{Hash: b.Hash, Block: b}:
	}

	return
}

func (bf *BlockFetcher) fetchFromMiners(ctx context.Context,
	bfr *blockFetchRequest, got chan BlockFetchReply, chainer Chainer,
	limit chan struct{}) {

	defer bf.release(limit)

	var nb, err = chainer.getNotarizedBlockFromMiners(ctx, bfr.hash)
	if err != nil {
		bf.gotError(ctx, got, bfr.hash, err)
		return
	}

	bf.gotBlock(ctx, got, nb)
}

func (bf *BlockFetcher) fetchFromSharders(ctx context.Context,
	bfr *blockFetchRequest, got chan BlockFetchReply, chainer Chainer,
	limit chan struct{}) {

	defer bf.release(limit)

	var fb, err = chainer.getFinalizedBlockFromSharders(ctx, &LFBTicket{
		LFBHash:   bfr.hash,      //
		Round:     bfr.round,     //
		SharderID: bfr.sharderID, // if set
	})
	if err != nil {
		bf.gotError(ctx, got, bfr.hash, err)
		return
	}

	bf.gotBlock(ctx, got, fb)
}

//
// Common interfaces used by the block fetcher.
//

// FetchedNotarizedBlockHandler - a handler that processes a fetched
// notarized block.
type FetchedNotarizedBlockHandler interface {
	NotarizedBlockFetched(ctx context.Context, b *block.Block)
}

// The Chainer represents Chain.
type Chainer interface {
	// LFB tickets work
	SubLFBTicket(ctx context.Context) (sub chan *LFBTicket)
	GetLatestLFBTicket(ctx context.Context) (tk *LFBTicket)
	// blocks fetching
	getFinalizedBlockFromSharders(ctx context.Context, ticket *LFBTicket) (
		fb *block.Block, err error)
	getNotarizedBlockFromMiners(ctx context.Context, hash string) (
		nb *block.Block, err error)
}

//
// the block fetching functions
//

// The getFinalizedBlockFromSharders - request for a finalized block from all
// sharders from current magic block.
func (c *Chain) getFinalizedBlockFromSharders(ctx context.Context,
	ticket *LFBTicket) (fb *block.Block, err error) {

	var (
		mb       = c.GetCurrentMagicBlock()
		sharders = mb.Sharders
		params   = make(url.Values)

		lctx, cancel = context.WithTimeout(ctx, node.TimeoutLargeMessage)
		fbMutex      sync.Mutex
	)
	defer cancel()

	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var gfb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if gfb.ComputeHash() != ticket.LFBHash {
			Logger.Error("fetch_fb_from_sharders - wrong block hash",
				zap.Int64("round", gfb.Round), zap.String("block", gfb.Hash))
			return nil, common.NewError("fetch_fb_from_sharders",
				"wrong block hash")
		}

		err = c.VerifyNotarization(ctx, gfb, gfb.GetVerificationTickets(),
			gfb.Round)
		if err != nil {
			Logger.Error("fetch_fb_from_sharders - not notarized",
				zap.Int64("round", gfb.Round), zap.String("block", gfb.Hash),
				zap.Error(err))
			return nil, err
		}

		if err = gfb.Validate(ctx); err != nil {
			Logger.Error("fetch_fb_from_sharders - invalid",
				zap.Int64("round", gfb.Round), zap.String("block", gfb.Hash),
				zap.Any("block_obj", gfb), zap.Error(err))
			return nil, err
		}

		fbMutex.Lock()
		defer fbMutex.Unlock()
		fb = gfb

		cancel() // stop
		return   // (nil, nil)
	}

	params.Add("hash", ticket.LFBHash)
	params.Add("round", strconv.FormatInt(ticket.Round, 10))

	// request from ticket sender, or. if the sender is missing,
	// try to fetch from all other sharders from the current MB
	if sh := sharders.GetNode(ticket.SharderID); sh != nil {
		sh.RequestEntityFromNode(lctx, FBRequestor, &params, handler)
	} else {
		sharders.RequestEntityFromAll(lctx, FBRequestor, &params, handler)
	}

	if fb == nil {
		return nil, common.NewError("fetch_fb_from_sharders", "no FB given")
	}

	return // return the first given
}

// The getNotarizedBlockFromMiners - get a notarized block for a round from
// miners. It verifies and validates block. But it never creates corresponding
// Chain round, never adds the block to the round, never adds block to the
// Chain, and never calls NotarizedBlockFetched that should be done after if
// required.
func (c *Chain) getNotarizedBlockFromMiners(ctx context.Context, hash string) (
	b *block.Block, err error) {

	var (
		cround = c.GetCurrentRound()
		params = make(url.Values)
	)

	params.Add("block", hash)

	// force a request to be limited (recreate the context for a the sharders
	// requesting fallback below)
	var (
		lctx, cancel = context.WithTimeout(ctx, node.TimeoutLargeMessage)
		mb           = c.GetCurrentMagicBlock()
		lock         sync.Mutex
	)
	defer cancel() // terminate the context after all anyway

	var handler = func(ctx context.Context, entity datastore.Entity) (
		_ interface{}, err error) {

		Logger.Info("fetch_nb_from_miners",
			zap.String("block", hash),
			zap.Int64("cround", cround),
			zap.Int64("current_round", c.GetCurrentRound()))

		var nb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if nb.ComputeHash() != hash {
			Logger.Error("fetch_nb_from_miners - wrong block hash",
				zap.Int64("round", nb.Round), zap.String("block", nb.Hash))
			return nil, common.NewError("fetch_nb_from_miners",
				"wrong block hash")
		}

		err = c.VerifyNotarization(ctx, nb, nb.GetVerificationTickets(),
			nb.Round)
		if err != nil {
			Logger.Error("fetch_nb_from_miners - not notarized",
				zap.Int64("round", nb.Round), zap.String("block", hash),
				zap.Error(err))
			return nil, err
		}

		if err = nb.Validate(ctx); err != nil {
			Logger.Error("fetch_nb_from_miners - invalid",
				zap.Int64("round", nb.Round), zap.String("block", hash),
				zap.Any("block_obj", nb), zap.Error(err))
			return nil, err
		}

		Logger.Debug("fetch_nb_from_miners -- ok",
			zap.String("block", nb.Hash),
			zap.Int64("round", nb.Round),
			zap.Int("verifictation_tickers", nb.VerificationTicketsSize()))

		cancel() // stop requesting on first block accepted

		// using mutex for the b variable
		lock.Lock()
		defer lock.Unlock()
		b = nb

		return // (nil, nil), don't return the block back
	}

	var n2n = mb.Miners
	n2n.RequestEntity(lctx, MinerNotarizedBlockRequestor, &params, handler)

	if b == nil {
		return nil, common.NewError("get_notarized_block", "no block given")
	}

	return
}

//
// Access the block fetcher from the Chain. Chain helper methods.
//

func (bf *BlockFetcher) fetch(ctx context.Context,
	bfr *blockFetchRequest) {

	select {
	case <-ctx.Done():
	case bf.fetchBlock <- bfr:
	}
}

// GetNotarizedBlock - get a notarized block for a round.
func (c *Chain) GetNotarizedBlock(ctx context.Context, hash string, rn int64) (
	nb *block.Block) {

	var bfr = new(blockFetchRequest)
	bfr.hash = hash
	bfr.round = rn

	var reply = make(chan BlockFetchReply, 1)
	bfr.replies = append(bfr.replies, reply)

	c.blockFetcher.fetch(ctx, bfr)

	var (
		cround = c.GetCurrentRound()

		rpl BlockFetchReply
	)

	select {
	case <-ctx.Done():
		return //
	case rpl = <-reply:
	}

	if rpl.Err != nil {
		Logger.Error("get notarized block - error",
			zap.Int64("cround", cround), zap.Int64("round", rn),
			zap.String("block", hash), zap.Error(rpl.Err))
		return // nil
	}

	// the block validated and its notarization verified
	nb = rpl.Block

	var r = c.GetRound(nb.Round)
	if r == nil {
		Logger.Info("get notarized block - no round, creating...",
			zap.Int64("round", nb.Round), zap.String("block", nb.Hash),
			zap.Int64("cround", cround))

		r = c.RoundF.CreateRoundF(nb.Round).(*round.Round)
		c.AddRound(r)
	}

	Logger.Info("got notarized block", zap.String("block", nb.Hash),
		zap.Int64("round", nb.Round),
		zap.Int("verifictation_tickers", nb.VerificationTicketsSize()))

	var b *block.Block
	b = c.AddBlock(nb)
	// This is a notarized block. So, use this method to sync round info
	// with the notarized block.
	b, r = c.AddNotarizedBlockToRound(r, nb)
	b, _ = r.AddNotarizedBlock(b)

	if b == nb {
		go c.fetchedNotarizedBlockHandler.NotarizedBlockFetched(ctx, nb)
	}

	return b
}

func (c *Chain) AsyncFetchFinalizedBlockFromSharders(ctx context.Context,
	ticket *LFBTicket) {

	var bfr = new(blockFetchRequest)
	bfr.hash = ticket.LFBHash        //
	bfr.round = ticket.Round         //
	bfr.sharders = true              // force to fetch from sharders
	bfr.sharderID = ticket.SharderID // request from this sharder, if given

	var reply = make(chan BlockFetchReply, 1)
	bfr.replies = append(bfr.replies, reply)

	c.blockFetcher.fetch(ctx, bfr)

	var rpl BlockFetchReply

	select {
	case <-ctx.Done():
		return //
	case rpl = <-reply:
	}

	if rpl.Err != nil {
		Logger.Error("async fetch fb from sharders - error",
			zap.Int64("round", bfr.round), zap.String("block", bfr.hash),
			zap.Error(rpl.Err))
		return // nil
	}

	// the block validated and its notarization verified
	var fb = rpl.Block

	// after work

	var r = c.GetRound(fb.Round)
	if r == nil {
		Logger.Info("async fetch fb from sharders - no round, creating...",
			zap.Int64("round", fb.Round), zap.String("block", fb.Hash))

		r = c.RoundF.CreateRoundF(fb.Round).(*round.Round)
		c.AddRound(r)
	}

	Logger.Info("async fetch fb from sharders", zap.String("block", fb.Hash),
		zap.Int64("round", fb.Round))

	var b *block.Block
	b = c.AddBlock(fb)
	// This is a notarized block. So, use this method to sync round info
	// with the notarized block.
	b, r = c.AddNotarizedBlockToRound(r, fb)
	b, _ = r.AddNotarizedBlock(b)
}

// FetchStat returns numbers of current block
// fetch requests to miners and to sharders.
func (c *Chain) FetchStat(ctx context.Context) (fqs FetchQueueStat) {
	select {
	case <-ctx.Done():
	case fqs = <-c.blockFetcher.statq:
	}
	return
}
