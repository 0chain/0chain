package network_simple_dkg

import (
	"context"
	"time"

	"0chain.net/node"
	"0chain.net/threshold/model"
	"0chain.net/threshold/model/party"
	"0chain.net/threshold/model/simple_dkg"
	. "0chain.net/threshold/network"
)

type Timeouts struct {
	retransmit time.Duration
}

type Msg interface{}
type ShareMsg struct {
	m model.Key
	v model.VerificationKey
}

type NetOutput struct {
	to node.Node
	m  Msg
}
type NetInput struct {
	from node.Node
	m    Msg
}

type Result interface{}
type Canceled struct{}
type IncorrectShare *node.Node
type Success model_party.Party

type Protocol struct {
	net      *NodeInfo
	dkg      model_simple_dkg.DKG
	timeouts Timeouts

	netOutput chan NetOutput
	netInput  chan NetInput

	results chan Result
}

func newProtocol(net *NodeInfo, t model.T, timeouts Timeouts) Protocol {
	return Protocol{
		net:      net,
		dkg:      model_simple_dkg.New(t, model.N(len(net.Peers.Nodes))),
		timeouts: timeouts,
		results:  make(chan Result, 10),
	}
}
func (p *Protocol) newShareMsg(to *node.Node) ShareMsg {
	i := p.net.HostToId[to.Host]
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
	i := p.net.HostToId[from.Host]
	p.dkg.ReceiveShare(i, m.m, m.v)
}

func (p *Protocol) run(ctx context.Context) {
	retransmit, _ := context.WithTimeout(ctx, p.timeouts.retransmit)
	p.transmitAll()
	for {
		select {
		case <-ctx.Done():
			p.results <- Canceled{}
			return
		case <-retransmit.Done():
			retransmit, _ = context.WithTimeout(ctx, p.timeouts.retransmit)
			p.transmitAll()
			continue
			// TODO: Receive share from a peer. Optinally quit.
		}
	}
}

func Run(ctx context.Context, net *NodeInfo, t model.T, timeouts Timeouts) <-chan Result {
	p := newProtocol(net, t, timeouts)
	go p.run(ctx)
	return p.results
}
