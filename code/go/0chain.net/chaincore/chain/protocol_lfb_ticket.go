package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"0chain.net/core/config"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/logging"
)

// compile-time resolution
var _ datastore.Entity = (*LFBTicket)(nil)

// LFBTicketSender represents.
var LFBTicketSender node.EntitySendHandler

// - Setup LFBTicketSender on initialization
// - Register LFB Ticket entity meta data
func setupLFBTicketSender() {
	// 1. Setup LFBTicketSender.
	var options = node.SendOptions{
		Timeout:            node.TimeoutSmallMessage,
		MaxRelayLength:     0,
		CurrentRelayLength: 0,
		Compress:           false,
	}
	LFBTicketSender = node.SendEntityHandler("/v1/block/get/latest_finalized_ticket", &options)
	// 2. Register LFBTicket EntityMetadata implementation.
	datastore.RegisterEntityMetadata("lfb_ticket", new(LFBTicketEntityMetadata))
}

// LFBTicketEntityMetadata implements datastore.EntityMetadata for LFBTicket.
type LFBTicketEntityMetadata struct{}

// GetName returns registered datastore.EntityMetadata name.
func (lfbtem *LFBTicketEntityMetadata) GetName() string {
	return "lfb_ticket"
}

// GetDB returns a stub string.
func (lfbtem *LFBTicketEntityMetadata) GetDB() string {
	return "lfb_ticket.db"
}

// Instance returns new blank LFBTicket.
func (lfbtem *LFBTicketEntityMetadata) Instance() datastore.Entity {
	return new(LFBTicket)
}

// GetStore is a stub.
func (lfbtem *LFBTicketEntityMetadata) GetStore() datastore.Store {
	return nil
}

// GetIDColumnName is a stub.
func (lfbtem *LFBTicketEntityMetadata) GetIDColumnName() string {
	return ""
}

// A LFBTicket represents ticket about LFB of a
// sharder. A sharder broadcasts the ticket to
// all other nodes (including other sharders).
// The ticket signed to protect against forgery.
type LFBTicket struct {
	Round     int64    `json:"round"`      // LFB round
	SharderID string   `json:"sharder_id"` // sender
	LFBHash   string   `json:"lfb_hash"`   // LFB hash
	Sign      string   `json:"sign"`       // ticket signature
	Senders   []string `json:"-"`          // internal
	IsOwn     bool     `json:"-"`          // is own
}

func (lfbt *LFBTicket) addSender(sharder string) { //nolint
	for _, sh := range lfbt.Senders {
		if sharder == sh {
			return // already hae
		}
	}
	lfbt.Senders = append(lfbt.Senders, sharder)
}

func (lfbt *LFBTicket) hashData() string {
	return fmt.Sprintf("%d:%s:%s", lfbt.Round, lfbt.SharderID, lfbt.LFBHash)
}

func (lfbt *LFBTicket) Hash() string {
	return encryption.Hash(lfbt.hashData())
}

func (c *Chain) newLFBTicket(b *block.Block) (ticket *LFBTicket) {
	var selfKey = node.Self.GetKey()
	ticket = new(LFBTicket)
	ticket.Round = b.Round
	ticket.SharderID = selfKey
	ticket.LFBHash = b.Hash
	ticket.Senders = append(ticket.Senders, selfKey) //
	ticket.IsOwn = true                              //
	var err error
	ticket.Sign, err = node.Self.Sign(ticket.Hash())
	if err != nil {
		panic(err) // must not happen
	}
	return
}

func (c *Chain) verifyLFBTicket(lfbt *LFBTicket) bool {
	var sharder = node.GetNode(lfbt.SharderID)
	if sharder == nil {
		return false // unknown or missing node
	}
	var ok, err = sharder.Verify(lfbt.Sign, lfbt.Hash())
	return err == nil && ok
}

// datastore.Entity implementation and stubs

func (lfbt *LFBTicket) GetKey() datastore.Key {
	return lfbt.SharderID + ":" + strconv.FormatInt(lfbt.Round, 10)
}

func (*LFBTicket) GetEntityMetadata() datastore.EntityMetadata {
	return new(LFBTicketEntityMetadata)
}

func (*LFBTicket) SetKey(datastore.Key)                      {}
func (*LFBTicket) GetScore() (int64, error)                  { return 0, nil }
func (*LFBTicket) ComputeProperties() error                  { return nil }
func (*LFBTicket) Validate(context.Context) error            { return nil }
func (*LFBTicket) Read(context.Context, datastore.Key) error { return nil }
func (*LFBTicket) Write(context.Context) error               { return nil }
func (*LFBTicket) Delete(context.Context) error              { return nil }

// sendLFBTicket to all appropriate nodes (by corresponding MB)
func (c *Chain) sendLFBTicket(ctx context.Context, ticket *LFBTicket) {
	logging.Logger.Debug("broadcast LFB ticket", zap.Int64("round", ticket.Round),
		zap.String("hash", ticket.LFBHash))

	var mb = c.GetMagicBlock(ticket.Round)
	if mb == nil {
		logging.Logger.Debug("broadcast LFB ticket: skip due to missing magic block",
			zap.Int64("round", ticket.Round),
			zap.String("hash", ticket.LFBHash))
		return
	}

	mb.Miners.SendAll(ctx, LFBTicketSender(ticket))
	mb.Sharders.SendAll(ctx, LFBTicketSender(ticket))
}

func (c *Chain) asyncSendLFBTicket(ctx context.Context, ticket *LFBTicket) {
	go c.sendLFBTicket(ctx, ticket)
}

// BroadcastLFBTicket sends LFB ticket to all other nodes from
// corresponding Magic Block.
func (c *Chain) BroadcastLFBTicket(ctx context.Context, b *block.Block) {
	if node.Self.Type != node.NodeTypeSharder {
		return
	}
	select {
	case c.broadcastLFBTicket <- b:
	case <-ctx.Done():
	}
}

// SubLFBTicket subscribes for received LFB tickets notifications.
func (c *Chain) SubLFBTicket() (sub chan *LFBTicket) {
	sub = make(chan *LFBTicket, 1)
	select {
	case c.subLFBTicket <- sub:
	case <-c.lfbTickerWorkerIsDone:
	}
	return
}

// UnsubLFBTicket unsubscribes from received LFB tickets notifications.
func (c *Chain) UnsubLFBTicket(sub chan *LFBTicket) {
	select {
	case c.unsubLFBTicket <- sub:
	case <-c.lfbTickerWorkerIsDone:
	}
}

// GetLatestLFBTicket
func (c *Chain) GetLatestLFBTicket(ctx context.Context) (tk *LFBTicket) {
	select {
	case tk = <-c.getLFBTicket:
		logging.Logger.Debug("GetLatestLFBTicket - received",
			zap.Int64("round", tk.Round))
	case <-ctx.Done():
		logging.Logger.Debug("GetLatestLFBTicket - context done")
	}
	return
}

func (c *Chain) BumpLFBTicket(ctx context.Context) {
	// Wait for LoadLatestBlocksFromStore to complete before bumping ticket.
	// This prevents race conditions where workers bump the ticket with network data
	// before the local LFB is properly loaded.
	if !c.IsLFBLoadingComplete() {
		logging.Logger.Debug("BumpLFBTicket - skipping (LFB loading not complete)")
		return
	}

	// Check if local LFB is at round 0 (fresh startup or stale state)
	// If so, DON'T bump the ticket - let the miner start from genesis and sync incrementally
	// Setting ticket high when at genesis causes filtering issues that prevent syncing
	localLFB := c.GetLatestFinalizedBlock()
	if localLFB == nil || localLFB.Round == 0 {
		logging.Logger.Debug("BumpLFBTicket - skipping (local LFB is nil or at round 0, will sync from genesis)")
		return
	}

	// Normal operation - fetch and bump ticket
	list := c.GetLatestFinalizedBlockFromSharder(ctx)

	if len(list) == 0 {
		logging.Logger.Debug("ensure_lfb - no new lfb received")
		return // no LFB given
	}

	rcvd := list[0].Block // the highest received LFB
	c.BumpTicket(ctx, rcvd)
}

// bump the ticket if necessary
func (c *Chain) BumpTicket(ctx context.Context, lfb *block.Block) {
	if lfb == nil {
		return
	}
	var tk = c.GetLatestLFBTicket(ctx) // is the worker starts
	if tk == nil || tk.Round < lfb.Round {
		logging.Logger.Debug("bumpLFBTicket", zap.Int64("lfb_round", lfb.Round))
		c.AddReceivedLFBTicket(ctx, &LFBTicket{Round: lfb.Round})
	}
}

type BlockConsensus struct {
	*block.Block
	Consensus int
}

// GetLatestFinalizedBlockFromSharder - request for latest finalized block from
// all the sharders.
func (c *Chain) GetLatestFinalizedBlockFromSharder(ctx context.Context) (
	fbs []*BlockConsensus) {

	// Get magic block for sharders list
	// First try LFMB, fall back to latest magic block (which always has genesis)
	// This ensures we can query sharders even on fresh startup before LFMB is set
	var mb *block.MagicBlock
	lfmb := c.GetLatestFinalizedMagicBlock(ctx)
	if lfmb != nil && lfmb.MagicBlock != nil {
		mb = lfmb.MagicBlock
	} else {
		// Fallback: use GetLatestMagicBlock which always returns at least genesis
		mb = c.GetLatestMagicBlock()
	}

	if mb == nil {
		return
	}

	shardersN2NURLs := mb.Sharders.N2NURLs()
	if len(shardersN2NURLs) == 0 {
		return
	}

	fbs = make([]*BlockConsensus, 0, len(shardersN2NURLs))
	fbc := make(chan *block.Block, len(shardersN2NURLs))

	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var fb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if fb.Round == 0 {
			return
		}

		if err = fb.Validate(ctx); err != nil {
			logging.Logger.Error("lfb from sharder - invalid",
				zap.Int64("round", fb.Round), zap.String("block", fb.Hash),
				zap.Error(err))
			return
		}
		select {
		case fbc <- fb:
		default:
		}

		return fb, nil
	}

	// Use RequestEntityFromShardersOnMB directly with our magic block
	// to avoid the LFMB channel lookup in RequestEntityFromSharders
	c.RequestEntityFromShardersOnMB(ctx, mb, MinerLatestFinalizedBlockRequestor, nil, handler)
	close(fbc)

	_, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for fb := range fbc {
		// increase consensus
		for i, b := range fbs {
			if b.Hash == fb.Hash {
				fbs[i].Consensus++
				continue
			}
		}

		lfbtk := c.GetLatestLFBTicket(ctx)
		localLFB := c.GetLatestFinalizedBlock()
		localLFBRound := int64(0)
		if localLFB != nil {
			localLFBRound = localLFB.Round
		}

		// Only filter by ticket if:
		// 1. We have a valid ticket with round > 0
		// 2. Sharder's LFB is less than our ticket
		// 3. AND sharder's LFB is also less than our actual LFB
		// This prevents rejecting sharder LFBs that are ahead of us but behind a stale ticket
		if lfbtk != nil && lfbtk.Round > 0 && fb.Round < lfbtk.Round && fb.Round < localLFBRound {
			logging.Logger.Debug("lfb from sharder - round too old",
				zap.Int64("round", fb.Round), zap.String("block", fb.Hash),
				zap.Int64("current_round", c.GetCurrentRound()),
				zap.Int64("local_lfb_round", localLFBRound),
				zap.Int64("ticket_round", lfbtk.Round),
			)
			continue
		}

		// add new block
		fbs = append(fbs, &BlockConsensus{
			Block:     fb,
			Consensus: 1,
		})
	}

	// highest (the first sorting order), most popular (the second order)
	sort.Slice(fbs, func(i int, j int) bool {
		if fbs[i].Round == fbs[j].Round {
			return fbs[i].Consensus > fbs[j].Consensus
		}

		return fbs[i].Round > fbs[j].Round
	})

	return
}

// GetLatestFinalizedBlockFromSharderNoFilter - request for latest finalized block from
// all the sharders WITHOUT filtering by LFB ticket round. This is used as a fallback
// when the miner's RocksDB is empty and needs to bootstrap from sharders.
func (c *Chain) GetLatestFinalizedBlockFromSharderNoFilter(ctx context.Context) (
	fbs []*BlockConsensus) {

	// Use GetLatestMagicBlock() instead of GetLatestFinalizedMagicBlockBrief()
	// because on fresh startup, LFMB may not be set yet in the worker,
	// but GetLatestMagicBlock() always returns at least the genesis magic block.
	mb := c.GetLatestMagicBlock()
	if mb == nil {
		logging.Logger.Warn("GetLatestFinalizedBlockFromSharderNoFilter - no magic block")
		return
	}

	shardersN2NURLs := mb.Sharders.N2NURLs()
	if len(shardersN2NURLs) == 0 {
		logging.Logger.Warn("GetLatestFinalizedBlockFromSharderNoFilter - no sharders in magic block")
		return
	}

	fbs = make([]*BlockConsensus, 0, len(shardersN2NURLs))
	fbc := make(chan *block.Block, len(shardersN2NURLs))

	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var fb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if fb.Round == 0 {
			return
		}

		if err = fb.Validate(ctx); err != nil {
			logging.Logger.Error("lfb from sharder (no filter) - invalid",
				zap.Int64("round", fb.Round), zap.String("block", fb.Hash),
				zap.Error(err))
			return
		}
		select {
		case fbc <- fb:
		default:
		}

		return fb, nil
	}

	// Use RequestEntityFromShardersOnMB directly with our magic block
	// to avoid the LFMB channel lookup in RequestEntityFromSharders
	c.RequestEntityFromShardersOnMB(ctx, mb, MinerLatestFinalizedBlockRequestor, nil, handler)
	close(fbc)

	for fb := range fbc {
		// increase consensus
		found := false
		for i, b := range fbs {
			if b.Hash == fb.Hash {
				fbs[i].Consensus++
				found = true
				break
			}
		}

		if found {
			continue
		}

		// add new block (NO filtering by ticket round)
		fbs = append(fbs, &BlockConsensus{
			Block:     fb,
			Consensus: 1,
		})
	}

	// highest (the first sorting order), most popular (the second order)
	sort.Slice(fbs, func(i int, j int) bool {
		if fbs[i].Round == fbs[j].Round {
			return fbs[i].Consensus > fbs[j].Consensus
		}

		return fbs[i].Round > fbs[j].Round
	})

	logging.Logger.Debug("GetLatestFinalizedBlockFromSharderNoFilter - received blocks",
		zap.Int("count", len(fbs)))

	return
}

func (c *Chain) sendLFBTicketEventToSubscribers(
	subs map[chan *LFBTicket]struct{}, ticket *LFBTicket) {

	for s := range subs {
		select {
		case s <- ticket: // the sending must be non-blocking
		default:
			logging.Logger.Debug("[send LFB-ticket event to subscribers] ignore one")
		}
	}
}

// StartLFBTicketWorker should work in a goroutine. It process received
// and generated LFB tickets. It works until context done.
func (c *Chain) StartLFBTicketWorker(ctx context.Context, on *block.Block) {

	var (
		// configurations (resend the latest by timer)
		rebroadcastTimeout = config.GetReBroadcastLFBTicketTimeout()
		rebroadcast        = time.NewTimer(rebroadcastTimeout)
		isSharder          = node.Self.Type == node.NodeTypeSharder

		// internals
		latest          = c.newLFBTicket(on) //
		localBumpTicket = latest
		subs            = make(map[chan *LFBTicket]struct{}) //

		// loop locals
		ticket *LFBTicket
		b      *block.Block
	)

	defer close(c.lfbTickerWorkerIsDone)
	defer rebroadcast.Stop()

	// don't broadcast if miner
	if !isSharder {
		rebroadcast.Stop()
		select {
		case <-rebroadcast.C:
		default:
		}
	}

	for {
		if isSharder {
			rebroadcast.Reset(rebroadcastTimeout)
		}

		select {

		// request current - return received ticket (or local if higher) for sync decisions
		case c.getLFBTicket <- latest:
			logging.Logger.Debug("getLFBTicket - sent",
				zap.Int64("latest.Round", latest.Round))
			// returns the highest known LFB ticket (received or local) for sync logic

		// a received LFB
		case ticket = <-c.updateLFBTicket:

			// drain all in the channel, choosing the latest one
			// (https://play.golang.org/p/PrLs7KaUgGF)
			var prev = ticket
			for len(c.updateLFBTicket) > 0 {
				ticket = <-c.updateLFBTicket
				if ticket.Round > prev.Round {
					prev = ticket
				}
			}

			ticket = prev // the latest in the channel

			if ticket.Round <= latest.Round {
				logging.Logger.Debug("update lfb ticket - SKIPPING (ticket.Round <= latest.Round)",
					zap.Int64("ticket.Round", ticket.Round),
					zap.Int64("latest.Round", latest.Round),
					zap.String("ticket.Sign", ticket.Sign))
				continue // not updated
			}
			logging.Logger.Info("update lfb ticket - ACCEPTING (ticket.Round > latest.Round)",
				zap.Int64("ticket.Round", ticket.Round),
				zap.Int64("latest.Round", latest.Round),
				zap.String("ticket.Sign", ticket.Sign))

			// Received tickets can be any round - we accept them to learn the network state.
			// We don't cap or rebroadcast received tickets; we only broadcast our own local LFB ticket.
			// Trigger sync if ticket is ahead of local LFB.
			lfb := c.GetLatestFinalizedBlock()
			if lfb != nil && ticket.Round > lfb.Round {
				logging.Logger.Info("update lfb ticket - received ticket ahead, triggering sync",
					zap.Int64("ticket_round", ticket.Round),
					zap.Int64("lfb_round", lfb.Round))
				c.NotifyBlockSync()
			}

			// for self updating case (kick itself) - blank ticket from BumpTicket
			if ticket.Sign == "" {
				// If ticket is ahead of local LFB, trigger sync to catch up
				lfb := c.GetLatestFinalizedBlock()
				if lfb != nil && ticket.Round > lfb.Round {
					logging.Logger.Info("update lfb ticket - ticket ahead of LFB, syncing",
						zap.Int64("ticket.Round", ticket.Round),
						zap.Int64("lfb.Round", lfb.Round))
					c.NotifyBlockSync()
				}
				oldLatest := latest.Round
				latest = ticket
				logging.Logger.Info("update lfb ticket - UPDATED latest (blank ticket)",
					zap.Int64("old_latest", oldLatest),
					zap.Int64("new_latest", latest.Round))
				// send for all subscribers
				c.sendLFBTicketEventToSubscribers(subs, ticket)
				continue // don't need a block for the blank kick ticket
			}

			// send for all subscribers
			c.sendLFBTicketEventToSubscribers(subs, ticket)

			// update latest
			oldLatest := latest.Round
			latest = ticket //
			logging.Logger.Info("update lfb ticket - UPDATED latest (signed ticket)",
				zap.Int64("old_latest", oldLatest),
				zap.Int64("new_latest", latest.Round))

			// don't broadcast a received LFB ticket, since its already
			// broadcasted by its sender

		// broadcast about new LFB
		case b = <-c.broadcastLFBTicket:
			// drain all pending blocks in the broadcastLFBTicket channel
			// (https://play.golang.org/p/PrLs7KaUgGF)
			var prev = b
			for len(c.broadcastLFBTicket) > 0 {
				b = <-c.broadcastLFBTicket
				if b.Round > prev.Round {
					prev = b
				}
			}

			b = prev // use latest, regardless order in the channel

			if b.Round <= localBumpTicket.Round {
				logging.Logger.Debug("update lfb ticket - b.Round <= localBumpTicket.Round",
					zap.Int64("b.Round", b.Round),
					zap.Int64("localLatestBump.Round", localBumpTicket.Round))
				continue // not updated
			}

			ticket = c.newLFBTicket(b)

			// send newer tickets
			c.asyncSendLFBTicket(ctx, ticket)

			// send for all subscribers, if any
			c.sendLFBTicketEventToSubscribers(subs, ticket)

			if localBumpTicket.Round < ticket.Round {
				localBumpTicket = ticket // update the latest
				if ticket.Round > latest.Round {
					oldLatest := latest.Round
					latest = ticket
					logging.Logger.Info("update lfb ticket - UPDATED latest (broadcast)",
						zap.Int64("old_latest", oldLatest),
						zap.Int64("new_latest", latest.Round))
				}
				logging.Logger.Debug("update lfb ticket", zap.Int64("round", latest.Round))
			}

		// rebroadcast after some timeout
		case <-rebroadcast.C:
			// Only rebroadcast our own ticket (localBumpTicket), not received tickets (latest).
			// A node should only advertise LFB rounds it actually has blocks for.
			c.asyncSendLFBTicket(ctx, localBumpTicket)

		// subscribe / unsubscribe for new *received* LFB Tickets
		case sub := <-c.subLFBTicket:
			subs[sub] = struct{}{}
		case unsub := <-c.unsubLFBTicket:
			delete(subs, unsub)

		// reset ticket to lower round (startup rollback)
		case b = <-c.resetLFBTicket:
			ticket = c.newLFBTicket(b)
			latest = ticket
			localBumpTicket = ticket
			logging.Logger.Info("reset lfb ticket",
				zap.Int64("round", ticket.Round),
				zap.String("hash", ticket.LFBHash))
			// send for all subscribers
			c.sendLFBTicketEventToSubscribers(subs, ticket)

		case <-ctx.Done():
			return
		}
	}

}

// AddReceivedLFBTicket used to update LFB ticket from a received one.
func (c *Chain) AddReceivedLFBTicket(ctx context.Context, ticket *LFBTicket) {
	select {
	case c.updateLFBTicket <- ticket:
	case <-ctx.Done():
	}
}

// ResetLFBTicket forces the LFB ticket to a specific block, even if lower than current.
// Used during startup when LFB rolls back due to invalid notarization.
func (c *Chain) ResetLFBTicket(ctx context.Context, b *block.Block) {
	select {
	case c.resetLFBTicket <- b:
	case <-ctx.Done():
	}
}

// LFBTicketHandler handles LFB tickets.
func LFBTicketHandler(ctx context.Context, r *http.Request) (
	resp interface{}, err error) {

	var chain = GetServerChain()

	// Reject network tickets until LoadLatestBlocksFromStore completes.
	// This prevents the ticket from being set too high before the local LFB is loaded,
	// which would cause the node to reject valid blocks during sync.
	if !chain.IsLFBLoadingComplete() {
		logging.Logger.Debug("handling LFB ticket - rejecting (LFB loading not complete)")
		return nil, common.NewError("lfb_ticket_handler", "node still loading")
	}

	var dec = json.NewDecoder(r.Body)
	defer r.Body.Close()

	var ticket LFBTicket
	if err = dec.Decode(&ticket); err != nil {
		logging.Logger.Debug("handling LFB ticket", zap.String("from", r.RemoteAddr),
			zap.Error(err))
		return // (nil, err)
	}

	if !chain.verifyLFBTicket(&ticket) {
		logging.Logger.Debug("handling LFB ticket", zap.String("err", "can't verify"),
			zap.Int64("round", ticket.Round))
		return nil, common.NewError("lfb_ticket_handler", "can't verify")
	}

	// Accept all signed tickets from network - they represent the sender's actual LFB state.
	// No cap on ticket round - nodes that are behind need to learn where the network is.
	// The rebroadcast logic uses localBumpTicket (not received tickets) so we won't
	// rebroadcast tickets we haven't synced to.
	chain.AddReceivedLFBTicket(ctx, &ticket)
	return // (nil, nil)
}

// StartLFMBWorker starts the worker for getting latest finalized magic block
func (c *Chain) StartLFMBWorker(ctx context.Context) {
	var (
		lfmb  *block.Block
		clone *block.Block
	)

	for {
		select {
		case c.getLFMB <- lfmb:
		case c.getLFMBClone <- clone:
		case v := <-c.updateLFMB:
			lfmb = v.block
			clone = v.clone
			logging.Logger.Debug("receive update LFMB",
				zap.Int64("round", clone.Round),
				zap.Int("miners", clone.Miners.Size()),
				zap.Int("sharders", clone.Sharders.Size()))
			v.reply <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Chain) updateLatestFinalizedMagicBlock(ctx context.Context, lfmb *block.Block) {
	v := &updateLFMBWithReply{
		block: lfmb,
		clone: lfmb.Clone(),
		reply: make(chan struct{}, 1),
	}
	select {
	case c.updateLFMB <- v:
		<-v.reply
	case <-ctx.Done():
		logging.Logger.Debug("update LFMB missed")
	}
}

// IsBlockSyncing checks if the miner is syncing blocks
func (c *Chain) IsBlockSyncing() bool {
	var (
		lfb          = c.GetLatestFinalizedBlock()
		lfbTkt       = c.GetLatestLFBTicket(context.Background())
		aheadN       = int64(3)
		currentRound = c.GetCurrentRound()
	)

	if currentRound < lfbTkt.Round ||
		lfb.Round+aheadN < lfbTkt.Round ||
		lfb.Round+int64(config.GetLFBTicketAhead()) < currentRound {
		return true
	}

	return false
}
