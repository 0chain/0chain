//go:build integration_tests
// +build integration_tests

package storagesc

import (
	"errors"
	"math/rand"
	"time"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/smartcontract/partitions"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

var numChalGen int

func (sc *StorageSmartContract) generateChallenge(
	t *transaction.Transaction,
	b *block.Block,
	input []byte,
	conf *Config,
	balances cstate.StateContextI,
) (err error) {

	s := crpc.Client().State()
	if s.StopChallengeGeneration != nil && *s.StopChallengeGeneration {
		numChalGen = 0
		logging.Logger.Info("Challenge generation has been stopped")
		return errors.New("challenge generation stopped by conductor")
	}

	if s.GenerateChallenge != nil {
		if s.BlobberCommittedWM != nil && !*s.BlobberCommittedWM {
			logging.Logger.Info("Selected blobber has not committed WM yet")
			return errors.New("challenge generation stopped by conductor because selected blobber has not committed any writemarker")
		}

		if numChalGen >= s.GenerateChallenge.TotalChallenges {
			logging.Logger.Info("Challenge generation execeed total challenge to generate",
				zap.Any("numChalGen", numChalGen), zap.Any("Total Challenges", s.GenerateChallenge.TotalChallenges))
			return errors.New("challenge generation stopped by conductor because total challenges required is already generated")
		}
	}

	numChalGen++
	return sc.genChal(t, b, input, conf, balances)
}

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
