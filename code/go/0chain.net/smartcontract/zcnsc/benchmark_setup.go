package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
)

func AddMockGlobalNode(balances cstate.StateContextI) {
	gn := &GlobalNode{
		ID: ADDRESS,
	}
	_, _ = balances.InsertTrieNode(gn.GetKey(), gn)
}

func AddMockUserNodes(
	clients []string,
	balances cstate.StateContextI,
) {
	for _, client := range clients {
		un := &UserNode{
			ID:   client,
		}
		_, _ = balances.InsertTrieNode(un.GetKey(ADDRESS), un)
	}
}

// TODO: Add authorizer nodes
// TODO: Add config - where?
