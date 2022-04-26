//go:build integration_tests
// +build integration_tests

package storagesc

import (
	"0chain.net/chaincore/node"

	crpc "0chain.net/conductor/conductrpc"
)

func afterInsertBlobber(id string) {
	var (
		client = crpc.Client()
		state  = client.State()
		abe    crpc.AddBlobberEvent
	)
	abe.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	abe.Blobber = state.Name(crpc.NodeID(id))
	if err := client.AddBlobber(&abe); err != nil {
		panic(err)
	}
	return
}
