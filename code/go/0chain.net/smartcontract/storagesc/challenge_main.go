//go:build !integration_tests
// +build !integration_tests

package storagesc

import (
	"math/rand"
	"time"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/partitions"
)

func (sc *StorageSmartContract) generateChallenge(
	t *transaction.Transaction,
	b *block.Block,
	input []byte,
	conf *Config,
	balances cstate.StateContextI,
) (err error) {
	return sc.genChal(t, b, input, conf, balances)
}

// selectBlobberForChallenge select blobber for challenge in random manner
func selectBlobberForChallenge(
	selection challengeBlobberSelection,
	challengeBlobbersPartition *partitions.Partitions,
	r *rand.Rand,
	balances cstate.StateContextI,
) (string, error) {

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

	return sc.processChallengePassed(
		balances, t, triggerPeriod,
		validatorsRewarded, cab, maxChallengeCompletionTime,
	)
}

func (sc *StorageSmartContract) challengeFailed(
	balances cstate.StateContextI,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
	maxChallengeCompletionTime time.Duration,
) (string, error) {

	return sc.processChallengeFailed(
		balances, validatorsRewarded, cab, maxChallengeCompletionTime)
}
