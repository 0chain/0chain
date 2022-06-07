//go:build integration_tests
// +build integration_tests

package utils

import (
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/conductor/config/cases"
)

// Split nodes list by given IsGoodOrBad.
func Split(s *crpc.State, igb crpc.IsGoodOrBad, nodes []*node.Node) (
	good, bad []*node.Node) {

	for _, n := range nodes {
		if igb.IsBad(s, n.GetKey()) {
			bad = append(bad, n)
		} else if igb.IsGood(s, n.GetKey()) {
			good = append(good, n)
		}
	}
	return
}

// Filter return IsBy nodes only.
func Filter(s *crpc.State, ib crpc.IsBy, nodes []*node.Node) (
	rest []*node.Node) {

	for _, n := range nodes {
		if ib.IsBy(s, n.GetKey()) {
			rest = append(rest, n)
		}
	}
	return
}

// IsSpammer checks whether a node is a spammer.
// A list of spammers names with format "miner-x" are passed, then the x is extracted and compared with the node index.
func IsSpammer(spammers []cases.NodeTypeTypeRank, roundNum int64) (isSpammer bool) {
	isSpammer = false

	nodeType, typeRank := chain.GetNodeTypeAndTypeRank(roundNum)

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

	nodeType, typeRank := chain.GetNodeTypeAndTypeRank(roundNum)

	return state.RoundHasFinalizedConfig.SpammingReceiver.NodeType == nodeType && state.RoundHasFinalizedConfig.SpammingReceiver.TypeRank == typeRank
}
