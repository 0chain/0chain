//go:build integration_tests
// +build integration_tests

package chain

import (
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/config/cases"
)

// IsSpammer checks whether a node is a spammer.
// A list of spammers names with format "miner-x" are passed, then the x is extracted and compared with the node index.
func IsSpammer(spammers []cases.NodeTypeTypeRank, roundNum int64) (isSpammer bool) {
	isSpammer = false

	nodeType, typeRank := GetNodeTypeAndTypeRank(roundNum)

	for _, spammer := range spammers {
		if spammer.NodeType == nodeType && spammer.TypeRank == typeRank {
			return true
		}
	}

	return
}

// IsSpamReceiver checks whether a node is the spam receiver set in the round_has_finalized configuration.
func IsSpamReceiver(state *crpc.State, roundNum int64) (isSpamReceiver bool) {
	isSpamReceiver = false

	nodeType, typeRank := GetNodeTypeAndTypeRank(roundNum)

	return state.RoundHasFinalized.SpammingReceiver.NodeType == nodeType && state.RoundHasFinalized.SpammingReceiver.TypeRank == typeRank
}
