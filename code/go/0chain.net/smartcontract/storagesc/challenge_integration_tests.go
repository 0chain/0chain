//go:build integration_tests
// +build integration_tests

package storagesc

import (
	"math/rand"
	"time"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/smartcontract/partitions"
)

var curTime = time.Now()

func (sc *StorageSmartContract) generateChallenge(
	t *transaction.Transaction,
	b *block.Block,
	input []byte,
	conf *Config,
	balances cstate.StateContextI,
) (err error) {
	err = sc.genChal(t, b, input, conf, balances)
	return
}

// selectBlobberForChallenge select blobber for challenge in random manner
func selectBlobberForChallenge(
	selection challengeBlobberSelection,
	challengeBlobbersPartition *partitions.Partitions,
	r *rand.Rand,
	balances cstate.StateContextI,
	conf *Config,
) (string, error) {

	s := crpc.Client().State()
	if s.GenerateChallenge != nil {
		return s.GenerateChallenge.BlobberID, nil
	}
	return selectRandomBlobber(selection, challengeBlobbersPartition, r, balances, conf)
}

func (sc *StorageSmartContract) challengePassed(
	balances cstate.StateContextI,
	t *transaction.Transaction,
	triggerPeriod int64,
	validatorsRewarded int,
	cab *challengeAllocBlobberPassResult,
) (string, error) {

	s, err := sc.processChallengePassed(
		balances, t, triggerPeriod,
		validatorsRewarded, cab)

	return s, err
}

func (sc *StorageSmartContract) challengeFailed(
	balances cstate.StateContextI,
	cab *challengeAllocBlobberPassResult,
) (string, error) {

	s, err := sc.processChallengeFailed(
		balances, cab)

	return s, err
}
