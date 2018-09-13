package network_byzantine_dkg

import (
	"0chain.net/node"
	"0chain.net/threshold/model"
	"0chain.net/threshold/model/byzantine_dkg"
	"0chain.net/threshold/model/party"
	. "0chain.net/threshold/network"
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
type SuccessOutput *model_party.Party

type Protocol struct {
	info *NodeInfo
	dkg  model_byzantine_dkg.DKG
}

func New(info *NodeInfo, t model.T) Protocol {
	return Protocol{
		info: info,
		dkg:  model_byzantine_dkg.New(t, model.N(len(info.Peers.Nodes))),
	}
}
func (p *Protocol) close()                                 {}
func (p *Protocol) send(to *node.Node, m ProtocolMsg)      {}
func (p *Protocol) receive(from *node.Node, m ProtocolMsg) {}

func Run(info *NodeInfo, t model.T) chan ProtocolOutput {
	New(info, t)
	return make(chan ProtocolOutput, 1)
}
