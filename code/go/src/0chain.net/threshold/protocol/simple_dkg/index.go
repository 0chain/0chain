package protocol_simple_dkg

import (
	"context"
	"time"

	"0chain.net/threshold/model"
	"0chain.net/threshold/model/party"
	"0chain.net/threshold/model/simple_dkg"
)

type Timeouts struct {
	retransmit time.Duration
}

type NetMsg struct {
	peer model.PartyId
	msg  model_simple_dkg.KeyShare
}

type Protocol struct {
	dkg      model_simple_dkg.DKG
	timeouts Timeouts
	network  chan NetMsg
	results  chan interface{}
	done     bool
}

func New(t int, n int, timeouts Timeouts, network chan NetMsg) Protocol {
	return Protocol{
		dkg:      model_simple_dkg.New(t, n),
		timeouts: timeouts,
		network:  network,
		results:  make(chan interface{}, 10),
		done:     false,
	}
}

func (p *Protocol) sendShare(to model.PartyId) {
	p.network <- NetMsg{
		peer: to,
		msg:  p.dkg.GetShareFor(to),
	}
}

func (p *Protocol) broadcastShares() {
	i := model.PartyId(0)
	n := model.PartyId(p.dkg.N)
	for ; i < n; i++ {
		if i != model.MyId {
			p.sendShare(i)
		}
	}
}

func (p *Protocol) receiveShare(from model.PartyId, m model_simple_dkg.KeyShare) error {
	return p.dkg.ReceiveShare(from, m)
}

func (p *Protocol) run(ctx context.Context) {
	p.broadcastShares()

	retransmit, cancel := context.WithTimeout(ctx, p.timeouts.retransmit)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			p.results <- ctx.Err()
		case <-retransmit.Done():
			p.broadcastShares()
			retransmit, cancel = context.WithTimeout(ctx, p.timeouts.retransmit)
			continue
		case msg := <-p.network:
			err := p.receiveShare(msg.peer, msg.msg)
			if err != nil {
				p.results <- err
			}
			if p.dkg.IsDone() && !p.done {
				p.results <- model_party.New(&p.dkg)
				p.done = true
			}
			continue
		}
	}
}

func Run(ctx context.Context, t int, n int, timeouts Timeouts, network chan NetMsg) <-chan interface{} {
	p := New(t, n, timeouts, network)
	go p.run(ctx)
	return p.results
}
