package network_byzantine_dkg

import (
	"0chain.net/node"
	"0chain.net/threshold/model"
	"0chain.net/threshold/network"
)

type ProtocolMsg interface{}
type ShareMsg struct {
	m model.Key
	v model.VerificationKey
}
type ComplaintMsg struct {
	against *node.Node
}
type DefendMsg struct {
	against *node.Node
	m       model.Key
	v       model.VerificationKey
}

type ProtocolOutput interface{}
type DisqualifiedOutput []*node.Node
type SuccessOutput *model.Party

type Protocol struct {
	info *network.NodeInfo
	dkg  model.ByzantineDKG
}

func New(info *network.NodeInfo, t int) Protocol {
	return Protocol{
		info: info,
		dkg:  model.NewByzantineDKG(t, len(info.Peers.Nodes)),
	}
}
func (p *Protocol) close()                                 {}
func (p *Protocol) send(to *node.Node, m ProtocolMsg)      {}
func (p *Protocol) receive(from *node.Node, m ProtocolMsg) {}

func Run(info *network.NodeInfo, t int) chan ProtocolOutput {
	New(info, t)
	return make(chan ProtocolOutput, 1)
}
