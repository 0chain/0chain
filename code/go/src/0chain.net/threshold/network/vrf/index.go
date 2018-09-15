package network_vrf

import (
	"0chain.net/node"
	"0chain.net/threshold/model"
	"0chain.net/threshold/network"
)

type Protocol struct {
	info *network.NodeInfo
	vrf  model.VRF
}

func New(info *network.NodeInfo, vrf model.VRF) Protocol {
	return Protocol{
		info: info,
		vrf:  vrf,
	}
}

func (p *Protocol) close() {
}

func (p *Protocol) send(to *node.Node, m model.SignatureShare) {
}

func (p *Protocol) receive(from *node.Node, m model.SignatureShare) {
}

func Run(info *network.NodeInfo, vrf model.VRF) chan model.RandomOutput {
	New(info, vrf)
	return make(chan model.RandomOutput, 1)
}
