//go:build !integration_tests
// +build !integration_tests

package storagesc

import (
	"math/rand"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
)

// selectBlobberForChallenge select blobber for challenge in random manner
func selectBlobberForChallenge(
	selection challengeBlobberSelection,
	challengeBlobbersPartition *partitions.Partitions,
	r *rand.Rand,
	balances cstate.StateContextI,
) (string, error) {

	return selectRandomBlobber(selection, challengeBlobbersPartition, r, balances)
}
