package byzantine_dkg

import (
	"0chain.net/node"
	. "0chain.net/threshold/controller"
	. "0chain.net/threshold/model"
	"0chain.net/threshold/model/byzantine_dkg"
	"0chain.net/threshold/model/party"
)

type ProtocolMsg interface{}
type ShareMsg struct {
	m Key
	v VerificationKey
}
type ComplaintMsg struct {
	against *node.Node
}
type DefendMsg struct {
	against *node.Node
	m       Key
	v       VerificationKey
}

type ProtocolOutput interface{}
type DisqualifiedOutput []*node.Node
type SuccessOutput *party.Party

type Protocol struct {
	info *NodeInfo
	dkg  byzantine_dkg.DKG
}

func New(info *NodeInfo, t T) Protocol {
	return Protocol{
		info: info,
		dkg:  byzantine_dkg.New(t, N(len(info.Peers.Nodes))),
	}
}
func (p *Protocol) close()                                 {}
func (p *Protocol) send(to *node.Node, m ProtocolMsg)      {}
func (p *Protocol) receive(from *node.Node, m ProtocolMsg) {}

func Run(info *NodeInfo, t T) chan ProtocolOutput {
	New(info, t)
	return make(chan ProtocolOutput, 1)
}
