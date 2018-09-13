package vrf

import (
	"0chain.net/node"
	. "0chain.net/threshold/controller"
	"0chain.net/threshold/model/party"
	"0chain.net/threshold/model/vrf"
)

type Protocol struct {
	info *NodeInfo
	vrf  vrf.VRF
}

func New(info *NodeInfo, vrf vrf.VRF) Protocol {
	return Protocol{
		info: info,
		vrf:  vrf,
	}
}
func (p *Protocol) close()                                          {}
func (p *Protocol) send(to *node.Node, m party.SignatureShare)      {}
func (p *Protocol) receive(from *node.Node, m party.SignatureShare) {}

func Run(info *NodeInfo, v vrf.VRF) chan vrf.RandomOutput {
	New(info, v)
	return make(chan vrf.RandomOutput, 1)
}
