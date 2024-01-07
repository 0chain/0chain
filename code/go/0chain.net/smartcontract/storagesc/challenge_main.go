//go:build !integration_tests
// +build !integration_tests

package storagesc

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	cstate "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/partitions"
	"math/rand"
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
	conf *Config,
) (string, error) {

	return selectRandomBlobber(selection, challengeBlobbersPartition, r, balances, conf)
}

func (sc *StorageSmartContract) challengePassed(
	balances cstate.StateContextI,
	t *transaction.Transaction,
	triggerPeriod int64,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
) (string, error) {

	return sc.processChallengePassed(
		balances, t, triggerPeriod,
		validatorsRewarded, cab,
	)
}

func (sc *StorageSmartContract) challengeFailed(
	balances cstate.StateContextI,
	cab *challengeAllocBlobberPassResult,
) (string, error) {
	return sc.processChallengeFailed(
		balances, cab)
}
