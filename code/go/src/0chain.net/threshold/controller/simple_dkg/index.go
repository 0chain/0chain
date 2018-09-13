package simple_dkg

import (
	"context"
	"time"

	"0chain.net/node"
	. "0chain.net/threshold/controller"
	. "0chain.net/threshold/model"
	"0chain.net/threshold/model/party"
	"0chain.net/threshold/model/simple_dkg"
)

type Timeouts struct {
	retransmit time.Duration
}

type ShareMsg struct {
	m Key
	v VerificationKey
}

type Output interface{}
type Canceled struct{}
type IncorrectShare *node.Node
type Success party.Party

type Protocol struct {
	net      *NodeInfo
	dkg      simple_dkg.DKG
	timeouts Timeouts
	output   chan Output
}

func newProtocol(net *NodeInfo, t T, timeouts Timeouts) Protocol {
	return Protocol{
		net:      net,
		dkg:      simple_dkg.New(t, N(len(net.Peers.Nodes))),
		timeouts: timeouts,
		output:   make(chan Output, 10),
	}
}
func (p *Protocol) newShareMsg(to *node.Node) ShareMsg {
	i := p.net.PeerIds[to.Host]
	m, v := p.dkg.GetShareFor(i)
	return ShareMsg{
		m: m,
		v: v,
	}
}
func (p *Protocol) transmitAll() {
	for _, peer := range p.net.Peers.Nodes {
		m := p.newShareMsg(peer)
		_ = m
		// TODO: Send m to peer.
	}
}
func (p *Protocol) receive(from *node.Node, m ShareMsg) {
	i := p.net.PeerIds[from.Host]
	p.dkg.ReceiveShare(i, m.m, m.v)
}

func (p *Protocol) run(ctx context.Context) {
	retransmit, _ := context.WithTimeout(ctx, p.timeouts.retransmit)
	p.transmitAll()
	for {
		select {
		case <-ctx.Done():
			p.output <- Canceled{}
			return
		case <-retransmit.Done():
			p.transmitAll()
			continue
			// TODO: Receive share from a peer. Optinally quit.
		}
	}
}

func Run(ctx context.Context, net *NodeInfo, t T, timeouts Timeouts) <-chan Output {
	p := newProtocol(net, t, timeouts)
	go p.run(ctx)
	return p.output
}
