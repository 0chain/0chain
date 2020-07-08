package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

// compile-time resolution
var _ datastore.Entity = ((*LFBTicket)(nil))

// LFBTicketSender represents.
var LFBTicketSender node.EntitySendHandler

// - Setup LFBTicketSender on initialization
// - Register LFB Ticket entity meta data
func init() {
	// 1. Setup LFBTicketSender.
	var options = node.SendOptions{
		Timeout:            node.TimeoutSmallMessage,
		MaxRelayLength:     0,
		CurrentRelayLength: 0,
		Compress:           false,
	}
	LFBTicketSender = node.SendEntityHandler(
		"/v1/block/get/latest_finalized_ticket",
		&options,
	)
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
	if err != nil {
		println("INVALID LFB TICKET SIGNATURE, ERR", err.Error())
	} else if !ok {
		println("INVALID LFB TICKET SIGNATURE, FALSE")
	}
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
func (c *Chain) sendLFBTicket(ticket *LFBTicket) {

	Logger.Debug("broadcast LFB ticket", zap.Int64("round", ticket.Round),
		zap.String("hash", ticket.LFBHash))

	var mb = c.GetMagicBlock(ticket.Round)
	mb.Miners.SendAll(LFBTicketSender(ticket))
	mb.Sharders.SendAll(LFBTicketSender(ticket))
	return
}

// BloadcastLFBTicket sends LFB ticket to all other nodes from
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
func (c *Chain) SubLFBTicket(ctx context.Context) (sub chan *LFBTicket) {
	sub = make(chan *LFBTicket)
	select {
	case c.subLFBTicket <- sub:
	case <-ctx.Done():
	}
	return
}

// UnsubLFBTicket unsubscribes from received LFB tickets notifications.
func (c *Chain) UnsubLFBTicket(ctx context.Context, sub chan *LFBTicket) {
	select {
	case c.unsubLFBTicket <- sub:
	case <-ctx.Done():
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

// StartLFBTicketWorker should work in a goroutine. It process received
// and generated LFB tickets. It works until context done.
func (c *Chain) StartLFBTicketWorker(ctx context.Context, on *block.Block) {
	println("StartLFBTicketWorker", on.Round)

	var (
		// configurations (resend the latest by timer)
		rebroadcastTimeout = config.GetReBroadcastLFBTicketTimeout()
		rebroadcast        = time.NewTimer(rebroadcastTimeout)
		isSharder          = (node.Self.Type == node.NodeTypeSharder)

		// internals
		latest = c.newLFBTicket(on)                 //
		subs   = make(map[chan *LFBTicket]struct{}) //

		// loop locals
		ticket *LFBTicket
		b      *block.Block
	)

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
			println("latest given", latest.Round)

		// a received LFB
		case ticket = <-c.updateLFBTicket:
			println("update received", ticket.Round)

			if _, err := c.getBlock(ctx, ticket.LFBHash); err != nil {
				println("update received: fetch")
				c.AsyncFetchNotarizedBlock(ticket.LFBHash)
				continue // if haven't the block, then don't update the latest
			}

			if ticket.Round <= latest.Round {
				println("update received: not a new")
				continue // not updated
			}

			println("update received: a new")

			// only if updated

			// send for all subscribers
			for s := range subs {
				select {
				case s <- ticket:
				case <-ctx.Done():
					return
				}
			}

			// update latest
			latest = ticket //

		// broadcast about new LFB
		case b = <-c.broadcastLFBTicket:
			println("broadcast", b.Round)
			ticket = c.newLFBTicket(b)
			c.sendLFBTicket(ticket)
			if ticket.Round > latest.Round {
				println("broadcast", b.Round, "set")
				latest = ticket // update
			}

			// add to list

		// rebroadcast after some timeout
		case <-rebroadcast.C:
			c.sendLFBTicket(latest)
			println("re-broadcast", latest.Round)

		// subscribe / unsubscribe for new *received* LFB Tickets
		case sub := <-c.subLFBTicket:
			println("subscribe")
			subs[sub] = struct{}{}
		case unsub := <-c.unsubLFBTicket:
			println("unsubscribe")
			delete(subs, unsub)

		case <-ctx.Done():
			println("done")
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

func LFBTicketHandler(ctx context.Context, r *http.Request) (
	resp interface{}, err error) {

	Logger.Debug("handle LFB ticket", zap.String("sharder", r.RemoteAddr))

	var dec = json.NewDecoder(r.Body)
	defer r.Body.Close()

	var ticket LFBTicket
	if err = dec.Decode(&ticket); err != nil {
		Logger.Error("handling LFB ticket", zap.String("from", r.RemoteAddr),
			zap.Error(err))
		return // (nil, err)
	}

	var chain = GetServerChain()
	if !chain.verifyLFBTicket(&ticket) {
		Logger.Error("handling LFB ticket", zap.String("err", "can't verify"),
			zap.Int64("round", ticket.Round))
		return nil, common.NewError("lfb_ticket_handler", "can't verify")
	}
	chain.AddReceivedLFBTicket(ctx, &ticket)
	return // (nil, nil)
}
