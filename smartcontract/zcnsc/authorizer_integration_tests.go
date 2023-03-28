//go:build integration_tests
// +build integration_tests

package zcnsc

import (
	"0chain.net/chaincore/node"

	crpc "0chain.net/conductor/conductrpc"
)

func afterInsertAuthorizer(id string) {
	var (
		client = crpc.Client()
		state  = client.State()
		abe    crpc.AddAuthorizerEvent
	)
	abe.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	abe.Authorizer = state.Name(crpc.NodeID(id))
	if err := client.AddAuthorizer(&abe); err != nil {
		panic(err)
	}
	return
}
