package chain

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"

	"0chain.net/core/logging"
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
		quit = ctx.Done()

		// internal mapping and replies
		fetching = make(map[string]*blockFetchRequest, cap(bf.fetchBlock))
		got      = make(chan BlockFetchReply)

		// track latest round known by sharders
		tickets = chainer.SubLFBTicket() // subscribe to new tickets
		tk      *LFBTicket               // internal
		latest  int64                    // latest given LFB ticket

		// limits
		minersl   = make(chan struct{}, fm)
		shardersl = make(chan struct{}, fs)

		stat FetchQueueStat
	)

	defer chainer.UnsubLFBTicket(tickets)

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
		case bf.statq <- stat:

		// update latest round known by sharders
		case tk = <-tickets:
			latest = tk.Round // update latest sharders round

		// handle block fetch requests
		case bfr := <-bf.fetchBlock:

			var have, ok = fetching[bfr.hash]
			if ok {
				have.replies = append(have.replies, bfr.replies...)
				continue
			}

			if len(fetching) >= total {
				go bf.terminate(ctx, bfr, ErrBlockFetchQueueFull)
				continue
			}

			fetching[bfr.hash] = bfr // add, increasing map length

			// if force from sharders
			if bfr.sharders {
				if bf.acquire(ctx, shardersl) {
					fetching[bfr.hash] = bfr
					go bf.fetchFromSharders(ctx, bfr, got, chainer, shardersl)
				} else {
					go bf.terminate(ctx, bfr, ErrBlockFetchShardersQueueFull)
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
				go bf.terminate(ctx, bfr, ErrBlockFetchMinersQueueFull)
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
				go bf.respond(ctx, bfr, rpl.Block)
				continue
			}

			// got no block, but error

			// already requested from sharders, so, it's the end
			if bfr.sharders {
				delete(fetching, rpl.Hash)
				go bf.terminate(ctx, bfr, rpl.Err)
				continue
			}

			// if block round > the latest ticket round, then we shouldn't
			// request it from sharders (it can't be on sharders)
			if bfr.round > 0 && bfr.round > latest {
				delete(fetching, rpl.Hash)
				go bf.terminate(ctx, bfr, rpl.Err)
				continue
			}

			// try request sharders for the block (set sharders: true to avoid
			// cyclic sharders requests)
			bfr.sharders = true
			if bf.acquire(ctx, shardersl) {
				go bf.fetchFromSharders(ctx, bfr, got, chainer, shardersl)
			} else {
				go bf.terminate(ctx, bfr, ErrBlockFetchShardersQueueFull)
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

	var nb, err = chainer.GetNotarizedBlockFromMiners(ctx, bfr.hash, bfr.round, true)
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

//go: generate
//go:generate mockery -name Chainer --case underscore -inpkg -testonly
// The Chainer represents Chain.
type Chainer interface {
	// LFB tickets work
	SubLFBTicket() (sub chan *LFBTicket)
	UnsubLFBTicket(sub chan *LFBTicket)
	GetLatestLFBTicket(ctx context.Context) (tk *LFBTicket)
	GetLatestFinalizedMagicBlockClone(ctx context.Context) *block.Block
	// blocks fetching
	getFinalizedBlockFromSharders(ctx context.Context, ticket *LFBTicket) (
		fb *block.Block, err error)
	GetNotarizedBlockFromMiners(ctx context.Context, hash string, round int64, withVerification bool) (
		nb *block.Block, err error)
}

//
// the block fetching functions
//

// getFinalizedBlockFromSharders - request for a finalized block from all
// sharders from current magic block.
func (c *Chain) getFinalizedBlockFromSharders(ctx context.Context,
	ticket *LFBTicket) (fb *block.Block, err error) {

	mb := c.getLatestFinalizedMagicBlock(ctx)
	if mb == nil {
		return nil, common.NewError("fetch_nb_from_miners", "could not find magic block")
	}

	sharders := mb.Sharders
	blockC := make(chan *block.Block, sharders.Size())

	lctx, cancel := context.WithTimeout(ctx, node.TimeoutLargeMessage)
	defer cancel()

	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var gfb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if gfb.ComputeHash() != ticket.LFBHash {
			logging.Logger.Error("fetch_fb_from_sharders - wrong block hash",
				zap.Int64("round", gfb.Round), zap.String("block", gfb.Hash))
			return nil, common.NewError("fetch_fb_from_sharders",
				"wrong block hash")
		}

		select {
		case blockC <- gfb:
		case <-ctx.Done():
		}

		return // (nil, nil)
	}

	validateBlock := func(b *block.Block) (*block.Block, error) {
		if err = b.Validate(ctx); err != nil {
			logging.Logger.Error("fetch_fb_from_sharders - invalid",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("block_obj", b), zap.Error(err))
			return nil, err
		}

		err = c.VerifyBlockNotarization(ctx, b)
		if err != nil {
			logging.Logger.Error("fetch_fb_from_sharders - verify notarization failed",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Error(err))
			return nil, err
		}

		b.SetBlockNotarized()
		return b, nil
	}

	params := make(url.Values)
	params.Add("hash", ticket.LFBHash)
	params.Add("round", strconv.FormatInt(ticket.Round, 10))

	// request from ticket sender, or. if the sender is missing,
	// try to fetch from all other sharders from the current MB
	if node.Self.Underlying().GetKey() != ticket.SharderID {
		if sh := sharders.GetNode(ticket.SharderID); sh != nil {
			sh.RequestEntityFromNode(lctx, FBRequestor, &params, handler)
			select {
			case fb = <-blockC:
				return validateBlock(fb)
			default:
				// or continue to request from all other sharders
			}
		}
	}

	doneC := make(chan struct{})
	go func() {
		sharders.RequestEntityFromAll(lctx, FBRequestor, &params, handler)
		close(doneC)
		close(blockC)
	}()

	for {
		select {
		case fb, ok := <-blockC:
			if !ok {
				return nil, common.NewError("fetch_fb_from_sharders", "no FB given")
			}

			b, err := validateBlock(fb)
			switch err {
			case nil:
			case context.Canceled,
				context.DeadlineExceeded:
				return nil, err
			default:
				continue
			}

			// stop requesting on first block accepted
			cancel()
			<-doneC

			return b, nil
		}
	}
}

// getNotarizedBlockFromMiners - get a notarized block for a round from
// miners. It verifies and validates block. But it never creates corresponding
// Chain round, never adds the block to the round, never adds block to the
// Chain, and never calls NotarizedBlockFetched that should be done after if
// required.
func (c *Chain) GetNotarizedBlockFromMiners(ctx context.Context, hash string, round int64, withVerification bool) (
	b *block.Block, err error) {
	params := make(url.Values)
	params.Add("block", hash)
	params.Add("round", strconv.FormatInt(round, 10))

	mb := c.getLatestFinalizedMagicBlock(ctx)
	if mb == nil {
		return nil, errors.New("fetch_nb_from_miners - could not find latest finalized magic block")
	}

	blockC := make(chan *block.Block, mb.Miners.Size())

	lctx, cancel := context.WithTimeout(ctx, node.TimeoutLargeMessage)
	defer cancel() // terminate the context after all anyway

	logging.Logger.Info("fetch_nb_from_miners",
		zap.String("block", hash),
		zap.Int64("current_round", c.GetCurrentRound()))
	var handler = func(_ context.Context, entity datastore.Entity) (
		_ interface{}, err error) {
		var nb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if hash != "" && nb.ComputeHash() != hash {
			logging.Logger.Error("fetch_nb_from_miners - wrong block hash",
				zap.Int64("round", nb.Round), zap.String("block", nb.Hash))
			return nil, common.NewError("fetch_nb_from_miners",
				"wrong block hash")
		}

		select {
		case blockC <- nb:
		case <-ctx.Done():
		}

		return // (nil, nil), don't return the block back
	}

	ts := time.Now()
	doneC := make(chan struct{})
	go func() {
		c.RequestEntityFromMiners(lctx, MinerNotarizedBlockRequestor, &params, handler)
		close(doneC)
		close(blockC)
	}()

	for {
		select {
		case nb, ok := <-blockC:
			if !ok {
				logging.Logger.Debug("fetch_nb_from_miners - no notarized block given",
					zap.Any("duration", time.Since(ts)))
				return nil, common.NewErrorf("fetch_nb_from_miners", "no notarized block given")
			}

			if err = nb.Validate(ctx); err != nil {
				logging.Logger.Error("fetch_nb_from_miners - invalid",
					zap.Int64("round", nb.Round), zap.String("block", hash),
					zap.Any("block_obj", nb), zap.Error(err))
				continue
			}

			if withVerification {
				err = c.VerifyBlockNotarization(ctx, nb)
				switch err {
				case nil:
				case context.Canceled, context.DeadlineExceeded:
					logging.Logger.Error("fetch_nb_from_miners - verify notarization tickets canceled or timeout",
						zap.Int64("round", nb.Round), zap.String("block", hash),
						zap.Any("duration", time.Since(ts)),
						zap.Error(err))
					return nil, err
				default:
					logging.Logger.Error("fetch_nb_from_miners - verify notarization tickets failed",
						zap.Int64("round", nb.Round), zap.String("block", hash),
						zap.Error(err))
					continue
				}
			}

			// cancel further requests
			cancel()
			<-doneC

			logging.Logger.Debug("fetch_nb_from_miners -- ok",
				zap.String("block", nb.Hash),
				zap.Int64("round", nb.Round),
				zap.Int("verification_tickers", nb.VerificationTicketsSize()))
			return nb, nil
		}
	}
}

// RequestEntityFromMiners requests entity from miners in latest finalized magic block
func (c *Chain) RequestEntityFromMiners(ctx context.Context, requestor node.EntityRequestor, params *url.Values, handler datastore.JSONEntityReqResponderF) {
	magicBlock := c.getLatestFinalizedMagicBlock(ctx)
	if magicBlock == nil {
		logging.Logger.Error("can't request entity")
		return
	}
	c.RequestEntityFromMinersOnMB(ctx, magicBlock, requestor, params, handler)
}

// RequestEntityFromSharders requests entity from sharders in latest finalized magic block
func (c *Chain) RequestEntityFromSharders(ctx context.Context, requestor node.EntityRequestor, params *url.Values, handler datastore.JSONEntityReqResponderF) {
	magicBlock := c.getLatestFinalizedMagicBlock(ctx)
	if magicBlock == nil {
		logging.Logger.Error("can't request entity")
		return
	}
	c.RequestEntityFromShardersOnMB(ctx, magicBlock, requestor, params, handler)
}

// RequestEntityFromMinersOnMB requests entity from miners on given magic block
func (c *Chain) RequestEntityFromMinersOnMB(ctx context.Context,
	mb *block.MagicBlock, requestor node.EntityRequestor, params *url.Values, handler datastore.JSONEntityReqResponderF) {
	if mb == nil {
		return
	}

	mb.Miners.RequestEntity(ctx, requestor, params, handler)
}

// RequestEntityFromShardersOnMB requests entity from sharders on given magic block
func (c *Chain) RequestEntityFromShardersOnMB(ctx context.Context,
	mb *block.MagicBlock, requestor node.EntityRequestor, params *url.Values, handler datastore.JSONEntityReqResponderF) {
	if mb == nil {
		return
	}
	mb.Sharders.RequestEntity(ctx, requestor, params, handler)
}

func (c *Chain) getLatestFinalizedMagicBlock(ctx context.Context) (mb *block.MagicBlock) {
	b := c.GetLatestFinalizedMagicBlock(ctx)
	if b == nil {
		return nil
	}

	return b.MagicBlock
}

//
// Access the block fetcher from the Chain. Chain helper methods.
//

func (bf *BlockFetcher) fetch(ctx context.Context,
	bfr *blockFetchRequest) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	case bf.fetchBlock <- bfr:
	}
	return nil
}

func (c *Chain) GetNotarizedBlockForce(ctx context.Context, hash string, rn int64) (*block.Block, error) {
	for true {
		select {
		case <-ctx.Done():
			break
		default:
		}

		notarizedBlock, err := c.GetNotarizedBlock(ctx, hash, rn)
		if err != nil {
			continue
		}
		return notarizedBlock, nil
	}
	return nil, context.DeadlineExceeded
}

// GetNotarizedBlock - get a notarized block for a round.
func (c *Chain) GetNotarizedBlock(ctx context.Context, hash string, rn int64) (*block.Block, error) {

	var bfr = new(blockFetchRequest)
	bfr.hash = hash
	bfr.round = rn

	var reply = make(chan BlockFetchReply, 1)
	bfr.replies = append(bfr.replies, reply)

	cctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if err := c.blockFetcher.fetch(cctx, bfr); err != nil {
		return nil, common.NewErrorf("get_notarized_block",
			"push to block fetch channel failed, round: %d, err: %v", bfr.round, err)
	}

	var (
		cround = c.GetCurrentRound()

		rpl BlockFetchReply
	)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case rpl = <-reply:
	}

	switch rpl.Err {
	case nil:
	case context.Canceled:
		return nil, context.Canceled
	default:
		return nil, rpl.Err
	}

	// the block validated and its notarization verified
	nb := rpl.Block

	var r = c.GetRound(nb.Round)
	if r == nil {
		logging.Logger.Info("get notarized block - no round, creating...",
			zap.Int64("round", nb.Round), zap.String("block", nb.Hash),
			zap.Int64("cround", cround))

		r = c.RoundF.CreateRoundF(nb.Round)
	}

	logging.Logger.Info("got notarized block", zap.String("block", nb.Hash),
		zap.Int64("round", nb.Round),
		zap.Int("verification_tickers", nb.VerificationTicketsSize()))

	var b *block.Block
	// This is a notarized block. So, use this method to sync round info
	// with the notarized block.
	var err error
	b, r, err = c.AddNotarizedBlockToRound(r, nb)
	if err != nil {
		logging.Logger.Error("get notarized block failed",
			zap.Int64("cround", cround), zap.Int64("round", rn),
			zap.String("block", hash), zap.Error(err))
		return nil, err
	}

	// Add the round if chain does not have it
	if c.GetRound(nb.Round) == nil {
		c.AddRound(r)
	}

	if b == nb {
		go c.fetchedNotarizedBlockHandler.NotarizedBlockFetched(ctx, nb)
	}

	return b, nil
}

type AfterBlockFetchFunc func(b *block.Block)

func (c *Chain) AsyncFetchFinalizedBlockFromSharders(ctx context.Context,
	ticket *LFBTicket, afterFetcher AfterFetcher) {

	var bfr = new(blockFetchRequest)
	bfr.hash = ticket.LFBHash        //
	bfr.round = ticket.Round         //
	bfr.sharders = true              // force to fetch from sharders
	bfr.sharderID = ticket.SharderID // request from this sharder, if given

	var reply = make(chan BlockFetchReply, 1)
	bfr.replies = append(bfr.replies, reply)

	cctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if err := c.blockFetcher.fetch(cctx, bfr); err != nil {
		logging.Logger.Error("async fetch fb from sharders - push to block fetcher failed",
			zap.Int64("round", bfr.round),
			zap.Error(err))
		return
	}

	var rpl BlockFetchReply

	select {
	case <-ctx.Done():
		return //
	case rpl = <-reply:
	}

	if rpl.Err != nil {
		logging.Logger.Error("async fetch fb from sharders - error",
			zap.Int64("round", bfr.round), zap.String("block", bfr.hash),
			zap.Error(rpl.Err))
		return // nil
	}

	// the block validated and its notarization verified
	var fb = rpl.Block

	// after fetch hook (if any)
	if afterFetcher != nil {
		var err error
		if err = afterFetcher.AfterFetch(ctx, fb); err != nil {
			logging.Logger.Error("async fetch fb from sharders - rejected by "+
				"the 'after fetch' hook", zap.Int64("round", bfr.round),
				zap.String("block", bfr.hash), zap.Error(err))
			return // nil
		}
	}

	// After the AfterFetch the following process can be terminated by an error
	// thus, we can set LFB inside the AfterFetch.

	var r = c.GetRound(fb.Round)
	if r == nil {
		logging.Logger.Info("async fetch fb from sharders - no round, creating...",
			zap.Int64("round", fb.Round), zap.String("block", fb.Hash))

		r = c.RoundF.CreateRoundF(fb.Round)
	}

	logging.Logger.Info("async fetch fb from sharders", zap.String("block", fb.Hash),
		zap.Int64("round", fb.Round))

	// This is a notarized block. So, use this method to sync round info
	// with the notarized block.
	var err error
	_, r, err = c.AddNotarizedBlockToRound(r, fb)
	if err != nil {
		logging.Logger.Error("async fetch fb from sharders failed",
			zap.Int64("round", bfr.round), zap.String("block", bfr.hash),
			zap.Error(err))
		return
	}

	//  Add the round to chain if does not in the chain yet
	if c.GetRound(fb.Round) == nil {
		c.AddRound(r)
	}
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
