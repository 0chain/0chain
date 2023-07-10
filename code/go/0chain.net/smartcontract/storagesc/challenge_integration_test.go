package storagesc

import (
	"math/rand"

	cstate "0chain.net/chaincore/chain/state"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/smartcontract/partitions"
)

// selectBlobberForChallenge select blobber for challenge in random manner
func selectBlobberForChallenge1(
	selection challengeBlobberSelection,
	challengeBlobbersPartition *partitions.Partitions,
	r *rand.Rand,
	balances cstate.StateContextI,
) (string, error) {

	s := crpc.Client().State()
	if s.GenerateChallenge != nil {
		crpc.Client().ChallengeGenerated(s.GenerateChallenge.BlobberID)
		return s.GenerateChallenge.BlobberID, nil
	}
	return selectRandomBlobber(selection, challengeBlobbersPartition, r, balances)
}
