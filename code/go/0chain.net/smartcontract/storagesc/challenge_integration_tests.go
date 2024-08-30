//go:build integration_tests
// +build integration_tests

package storagesc

import (
	"time"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
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
