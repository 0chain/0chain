package protocol_byzantine_dkg

import (
	"context"
	"time"

	"0chain.net/threshold/model"
	"0chain.net/threshold/model/byzantine_dkg"
	"0chain.net/threshold/model/party"
)

type Timeouts struct {
	retransmit time.Duration
}

type ShareMsg struct {
	m model.Key
	v model.VerificationKey
}
type ComplaintMsg struct {
	against model.PartyId
}
type DefendMsg struct {
	defending model.PartyId
	m         model.Key
	v         model.VerificationKey
}

type NetMsg struct {
	peer model.PartyId
	msg  interface{}
}

type Protocol struct {
	dkg      model_byzantine_dkg.DKG
	timeouts Timeouts
	network  chan NetMsg
	results  chan interface{}
	done     bool
}

func New(t model.T, n model.N, timeouts Timeouts, network chan NetMsg) Protocol {
	return Protocol{
		dkg:      model_byzantine_dkg.New(t, n),
		timeouts: timeouts,
		network:  network,
		results:  make(chan interface{}, 10),
		done:     false,
	}
}

func (p *Protocol) sendShare(to model.PartyId) {
	m, v := p.dkg.Simple.GetShareFor(to)
	p.network <- NetMsg{
		peer: to,
		msg: ShareMsg{
			m: m,
			v: v,
		},
	}
}

func (p *Protocol) sendComplaint(against, to model.PartyId) {
	p.network <- NetMsg{
		peer: to,
		msg: ComplaintMsg{
			against: against,
		},
	}
}

func (p *Protocol) sendDefendMsg(defending model.PartyId, to model.PartyId) {
	p.dkg.Simple.ReceiveShare()
	p.network <- NetMsg{
		peer: to,
		msg: DefendMsg{
			against: against,
			m:       m,
			v:       v,
		},
	}
}

func (p *Protocol) broadcastShares() {
	i := model.PartyId(0)
	n := model.PartyId(p.dkg.Simple.N)
	for ; i < n; i++ {
		if i != model.MyId {
			p.sendShare(i)
		}
	}
}

func (p *Protocol) receiveShare(from model.PartyId, m ShareMsg) error {
	return p.dkg.Simple.ReceiveShare(from, m.m, m.v)
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
			if p.dkg.Simple.IsDone() && !p.done {
				p.results <- model_party.New(&p.dkg.Simple)
				p.done = true
			}
			continue
		}
	}
}

func Run(ctx context.Context, t model.T, n model.N, timeouts Timeouts, network chan NetMsg) <-chan interface{} {
	p := New(t, n, timeouts, network)
	go p.run(ctx)
	return p.results
}
