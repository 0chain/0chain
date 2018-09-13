package simple_dkg

import (
	"0chain.net/node"
	. "0chain.net/threshold/controller"
	. "0chain.net/threshold/model"
	"0chain.net/threshold/model/party"
	"0chain.net/threshold/model/simple_dkg"
)

type ShareMsg struct {
	m Key
	v VerificationKey
}

type Protocol struct {
	info *NodeInfo
	dkg  simple_dkg.DKG
}

func New(info *NodeInfo, t T) Protocol {
	return Protocol{
		info: info,
		dkg:  simple_dkg.New(t, N(len(info.Peers.Nodes))),
	}
}
func (p *Protocol) close() {}
func (p *Protocol) GetShareMsgFor(to *node.Node) ShareMsg {
	i := p.info.PeerIds[to.Host]
	m, v := p.dkg.GetShareFor(i)
	return ShareMsg{
		m: m,
		v: v,
	}
}
func (p *Protocol) receive(from *node.Node, m ShareMsg) {
	i := p.info.PeerIds[from.Host]
	p.dkg.ReceiveShare(i, m.m, m.v)
}

func Run(info *NodeInfo, t T) chan party.Party {
	New(info, t)
	return make(chan party.Party, 1)
}
