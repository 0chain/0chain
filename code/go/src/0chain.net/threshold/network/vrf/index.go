package network_vrf

import (
	"0chain.net/node"
	"0chain.net/threshold/model/party"
	"0chain.net/threshold/model/vrf"
	. "0chain.net/threshold/network"
)

type Protocol struct {
	info *NodeInfo
	vrf  model_vrf.VRF
}

func New(info *NodeInfo, vrf model_vrf.VRF) Protocol {
	return Protocol{
		info: info,
		vrf:  vrf,
	}
}

func (p *Protocol) close() {
}

func (p *Protocol) send(to *node.Node, m model_party.SignatureShare) {
}

func (p *Protocol) receive(from *node.Node, m model_party.SignatureShare) {
}

func Run(info *NodeInfo, vrf model_vrf.VRF) chan model_vrf.RandomOutput {
	New(info, vrf)
	return make(chan model_vrf.RandomOutput, 1)
}
