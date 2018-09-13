package vrf

import (
	. "0chain.net/threshold/model"
	. "0chain.net/threshold/model/party"
)

type Round uint64
type RandomOutput [4]uint64

type VRF struct{}

func New(p *Party, round Round, prev RandomOutput) VRF {
	return VRF{}
}
func (vrf *VRF) ReceiveShare(i PartyId, share SignatureShare) bool {
	return true
}
func (vrf *VRF) Output() *RandomOutput {
	return nil
}
