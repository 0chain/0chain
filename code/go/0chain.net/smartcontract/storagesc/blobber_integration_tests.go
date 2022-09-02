//go:build integration_tests
// +build integration_tests

package storagesc

import (
	"strings"

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

func afterAddChallenge(challengeID string, validatorsIDs []string) {

}

func beforeEmitAddChallenge(challenge *StorageChallengeResponse) {
	var (
		client = crpc.Client()
		state  = client.State()
	)

	if state.AdversarialValidator != nil && state.AdversarialValidator.PassAllChallenges && containsString(challenge.ValidatorIDs, state.AdversarialValidator.ID) {
		// any challenge adulteration produces an invalid challenge
		challenge.AllocationRoot = strings.ReplaceAll(challenge.AllocationRoot, "1", "0")
		challenge.AllocationRoot = strings.ReplaceAll(challenge.AllocationRoot, "a", "b")
	}
}

func containsString(arr []string, str string) bool {
	for _, s := range arr {
		if s == str {
			return true
		}
	}

	return false
}
