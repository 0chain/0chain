package controller

import (
	"0chain.net/node"
	. "0chain.net/threshold/model"
)

type NodeInfo struct {
	Peers   *node.Pool
	PeerIds map[string]PartyId
	myId    PartyId
}

func NewNodeInfo(peers *node.Pool) NodeInfo {
	peerIds := make(map[string]PartyId, len(peers.Nodes)+1)
	for i := 0; i < len(peers.Nodes); i++ {
		peerIds[peers.Nodes[i].Host] = PartyId(i + 1)
	}
	return NodeInfo{
		Peers:   peers,
		PeerIds: peerIds,
		myId:    0,
	}
}
