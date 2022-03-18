package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
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

func (lfbt *LFBTicket) addSender(sharder string) {
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
func (*LFBTicket) GetScore() int64                           { return 0 }
func (*LFBTicket) ComputeProperties()                        {}
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
	return
}

// GetLatestLFBTicket
func (c *Chain) GetLatestLFBTicket(ctx context.Context) (tk *LFBTicket) {
	select {
	case tk = <-c.getLFBTicket:
	case <-ctx.Done():
	}
	return
}

func (c *Chain) sendLFBTicketEventToSubscribers(
	subs map[chan *LFBTicket]struct{}, ticket *LFBTicket) {

	logging.Logger.Debug("[send LFB-ticket event to subscribers]",
		zap.Int("subs", len(subs)), zap.Int64("round", ticket.Round))
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
		latest = c.newLFBTicket(on)                 //
		subs   = make(map[chan *LFBTicket]struct{}) //

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

		// request current
		case c.getLFBTicket <- latest:
			// request latest LFB Ticket generated or received at any time

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
				logging.Logger.Debug("update lfb ticket -  ticket.Round <= latest.Round",
					zap.Int64("ticket.Round", ticket.Round),
					zap.Int64("latest.Round", latest.Round))
				continue // not updated
			}

			// for self updating case (kick itself)
			if ticket.Sign == "" {
				latest = ticket
				// send for all subscribers
				c.sendLFBTicketEventToSubscribers(subs, ticket)
				continue // don't need a block for the blank kick ticket
			}

			// only if updated, only for sharders
			// (don't rebroadcast without a block verification)

			if isSharder {
				if _, err := c.GetBlock(ctx, ticket.LFBHash); err != nil {
					c.AsyncFetchFinalizedBlockFromSharders(ctx, ticket,
						c.afterFetcher)
					continue // if haven't the block, then don't update the latest
				}
			}

			// send for all subscribers
			c.sendLFBTicketEventToSubscribers(subs, ticket)

			// update latest
			latest = ticket //
			logging.Logger.Debug("update lfb ticket", zap.Int64("round", latest.Round))

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

			if b.Round <= latest.Round {
				logging.Logger.Debug("update lfb ticket - b.Round <= latest.Round",
					zap.Int64("b.Round", b.Round),
					zap.Int64("latest.Round", latest.Round))
				continue // not updated
			}

			ticket = c.newLFBTicket(b)

			// send newer tickets
			c.asyncSendLFBTicket(ctx, ticket)

			// send for all subscribers, if any
			c.sendLFBTicketEventToSubscribers(subs, ticket)

			latest = ticket // update the latest
			logging.Logger.Debug("update lfb ticket", zap.Int64("round", latest.Round))

		// rebroadcast after some timeout
		case <-rebroadcast.C:
			// send newer tickets
			c.asyncSendLFBTicket(ctx, latest)

		// subscribe / unsubscribe for new *received* LFB Tickets
		case sub := <-c.subLFBTicket:
			subs[sub] = struct{}{}
		case unsub := <-c.unsubLFBTicket:
			delete(subs, unsub)

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

// LFBTicketHandler handles LFB tickets.
func LFBTicketHandler(ctx context.Context, r *http.Request) (
	resp interface{}, err error) {

	var dec = json.NewDecoder(r.Body)
	defer r.Body.Close()

	var ticket LFBTicket
	if err = dec.Decode(&ticket); err != nil {
		logging.Logger.Debug("handling LFB ticket", zap.String("from", r.RemoteAddr),
			zap.Error(err))
		return // (nil, err)
	}

	var chain = GetServerChain()
	if !chain.verifyLFBTicket(&ticket) {
		logging.Logger.Debug("handling LFB ticket", zap.String("err", "can't verify"),
			zap.Int64("round", ticket.Round))
		return nil, common.NewError("lfb_ticket_handler", "can't verify")
	}

	logging.Logger.Debug("handle LFB ticket", zap.String("sharder", r.RemoteAddr),
		zap.Int64("round", ticket.Round))
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
			logging.Logger.Debug("update LFMB", zap.Int64("round", lfmb.Round))
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
