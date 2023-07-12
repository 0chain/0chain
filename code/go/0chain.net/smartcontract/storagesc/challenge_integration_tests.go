//go:build integration_tests
// +build integration_tests

package storagesc

import (
	"math/rand"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/smartcontract/partitions"
)

// selectBlobberForChallenge select blobber for challenge in random manner
func selectBlobberForChallenge(
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

func (sc *StorageSmartContract) challengePassed(
	balances cstate.StateContextI,
	t *transaction.Transaction,
	triggerPeriod int64,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
	maxChallengeCompletionTime time.Duration,
) (string, error) {

	s, err := sc.processChallengePassed(
		balances, t, triggerPeriod,
		validatorsRewarded, cab, maxChallengeCompletionTime,
	)

	m := map[string]interface{}{
		"blobber_id": cab.blobAlloc.BlobberID,
		"status":     0,
	}

	if err == nil {
		m["status"] = 1
	}

	crpc.Client().SendChallengeStatus(m)

	return s, err
}

func (sc *StorageSmartContract) challengeFailed(
	balances cstate.StateContextI,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
	maxChallengeCompletionTime time.Duration,
) (string, error) {

	s, err := sc.processChallengeFailed(
		balances, validatorsRewarded, cab, maxChallengeCompletionTime)

	m := map[string]interface{}{
		"error":      err.Error(),
		"status":     0,
		"response":   s,
		"blobber_id": cab.blobAlloc.BlobberID,
	}

	crpc.Client().SendChallengeStatus(m)
	return s, err
}
