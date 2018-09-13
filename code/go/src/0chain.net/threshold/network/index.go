package network

import (
	"0chain.net/node"
	"0chain.net/threshold/model"
)

type NodeInfo struct {
	Peers    *node.Pool
	HostToId map[string]model.PartyId
	IdToHost map[model.PartyId]string
	myId     model.PartyId
}

func NewNodeInfo(peers *node.Pool) NodeInfo {
	numNodes := len(peers.Nodes) + 1

	hostToId := make(map[string]model.PartyId, numNodes)
	idToHost := make(map[model.PartyId]string, numNodes)

	for i := 0; i < len(peers.Nodes); i++ {
		host := peers.Nodes[i].Host
		id := model.PartyId(i + 1)

		hostToId[host] = id
		idToHost[id] = host
	}

	return NodeInfo{
		Peers:    peers,
		HostToId: hostToId,
		IdToHost: idToHost,
	}
}
