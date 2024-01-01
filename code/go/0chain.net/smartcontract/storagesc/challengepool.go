package storagesc

import (
	"encoding/json"
	"fmt"

	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

//msgp:ignore challengePoolStat
//go:generate msgp -io=false -tests=false -unexported=true -v

// challenge pool is a locked tokens for a duration for an allocation

type challengePool struct {
	*tokenpool.ZcnPool `json:"pool"`
}

func newChallengePool() *challengePool {
	return &challengePool{
		ZcnPool: &tokenpool.ZcnPool{},
	}
}

func challengePoolKey(scKey, allocationID string) datastore.Key {
	return datastore.Key(scKey + ":challengepool:" + allocationID)
}

func (cp *challengePool) Encode() (b []byte) {
	var err error
	if b, err = json.Marshal(cp); err != nil {
		panic(err) // must never happens
	}
	return
}

func (cp *challengePool) Decode(input []byte) (err error) {

	type challengePoolJSON struct {
		Pool json.RawMessage `json:"pool"`
	}

	var challengePoolVal challengePoolJSON
	if err = json.Unmarshal(input, &challengePoolVal); err != nil {
		return
	}

	if len(challengePoolVal.Pool) == 0 {
		return // no data given
	}

	err = cp.ZcnPool.Decode(challengePoolVal.Pool)
	return
}

// save the challenge pool
func (cp *challengePool) save(sscKey string, alloc *StorageAllocation, balances cstate.StateContextI) (err error) {
	cpKey := challengePoolKey(sscKey, alloc.ID)
	r, err := balances.InsertTrieNode(cpKey, cp)
	logging.Logger.Debug("after Save challenge pool", zap.String("root", util.ToHex([]byte(r))))

	//emit challenge pool event
	emitChallengePoolEvent(cpKey, cp.GetBalance(), alloc, balances)

	return
}

func emitChallengePoolEvent(id string, balance currency.Coin, alloc *StorageAllocation, balances cstate.StateContextI) {
	data := event.ChallengePool{
		ID:           id,
		AllocationID: alloc.ID,
		Balance:      int64(balance),
		StartTime:    int64(alloc.StartTime),
		Expiration:   int64(alloc.Expiration),
		Finalized:    alloc.Finalized,
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrUpdateChallengePool, id, data)

	return
}

func (cp *challengePool) moveToValidators(
	reward currency.Coin,
	validators []datastore.Key,
	vSPs []*stakePool,
	balances cstate.StateContextI,
	allocationID string,
) error {
	if len(validators) == 0 || reward == 0 {
		return nil // nothing to move, or nothing to move to
	}

	if cp.ZcnPool.Balance < reward {
		return fmt.Errorf("not enough tokens in challenge pool: %v < %v", cp.Balance, reward)
	}

	oneReward, bal, err := currency.DistributeCoin(reward, int64(len(validators)))
	if err != nil {
		return err
	}

	for i, sp := range vSPs {
		err := sp.DistributeRewards(oneReward, validators[i], spenum.Validator, spenum.ValidationReward, balances, allocationID)
		if err != nil {
			return fmt.Errorf("moving to validator %s: %v",
				validators[i], err)
		}
	}
	if bal > 0 {
		for i := 0; i < int(bal); i++ {
			err := vSPs[i].DistributeRewards(1, validators[i], spenum.Validator, spenum.ValidationReward, balances, allocationID)
			if err != nil {
				return fmt.Errorf("moving to validator %s: %v",
					validators[i], err)
			}
		}
	}

	cp.ZcnPool.Balance -= reward
	return nil
}

func (cp *challengePool) moveToBlobbers(sscKey string, reward currency.Coin,
	blobberId datastore.Key,
	sp *stakePool,
	balances cstate.StateContextI,
	allocationID string,
) error {

	logging.Logger.Info("Jayash moveToBlobbers", zap.Any("reward", reward), zap.Any("blobberId", blobberId), zap.Any("sp", sp), zap.Any("balances", balances), zap.Any("allocationID", allocationID))

	if reward == 0 {
		return nil // nothing to move, or nothing to move to
	}

	if cp.ZcnPool.Balance < reward {
		return fmt.Errorf("not enough tokens in challenge pool: %v < %v", cp.Balance, reward)
	}

	err := sp.DistributeRewards(reward, blobberId, spenum.Blobber, spenum.ChallengePassReward, balances, allocationID)
	if err != nil {
		return fmt.Errorf("can't move tokens to blobber: %v", err)
	}

	cp.ZcnPool.Balance -= reward
	return nil
}

func toChallengePoolStat(cp *event.ChallengePool) *challengePoolStat {
	stat := challengePoolStat{
		ID:         cp.ID,
		Balance:    currency.Coin(cp.Balance),
		StartTime:  common.Timestamp(cp.StartTime),
		Expiration: common.Timestamp(cp.Expiration),
		Finalized:  cp.Finalized,
	}

	return &stat
}

// swagger:model challengePoolStat
type challengePoolStat struct {
	ID         string           `json:"id"`
	Balance    currency.Coin    `json:"balance"`
	StartTime  common.Timestamp `json:"start_time"`
	Expiration common.Timestamp `json:"expiration"`
	Finalized  bool             `json:"finalized"`
}

//
// smart contract methods
//

// getChallengePool of current client
func (ssc *StorageSmartContract) getChallengePool(allocationID datastore.Key,
	balances cstate.StateContextI) (cp *challengePool, err error) {
	cp = newChallengePool()
	err = balances.GetTrieNode(challengePoolKey(ssc.ID, allocationID), cp)
	return
}

// newChallengePool SC function creates new
// challenge pool for a client don't saving it
func (ssc *StorageSmartContract) newChallengePool(allocationID string,
	balances cstate.StateContextI) (
	cp *challengePool, err error) {

	_, err = ssc.getChallengePool(allocationID, balances)
	switch err {
	case util.ErrValueNotPresent:
		cp = newChallengePool()
		cp.TokenPool.ID = challengePoolKey(ssc.ID, allocationID)
		return cp, nil
	case nil:
		return nil, common.NewError("new_challenge_pool_failed", "already exist")
	default:
		return nil, common.NewError("new_challenge_pool_failed", err.Error())
	}
}

// create, fill and Save challenge pool for new allocation
func (ssc *StorageSmartContract) createChallengePool(t *transaction.Transaction,
	alloc *StorageAllocation, balances cstate.StateContextI, conf *Config) (err error) {

	// create related challenge_pool expires with the allocation + challenge
	// completion time
	var cp *challengePool

	cp, err = ssc.newChallengePool(alloc.ID, balances)
	if err != nil {
		return fmt.Errorf("can't create challenge pool: %v", err)
	}

	// don't lock anything here

	// Save the challenge pool
	if err = cp.save(ssc.ID, alloc, balances); err != nil {
		return fmt.Errorf("can't Save challenge pool: %v", err)
	}

	return
}

func (ssc *StorageSmartContract) deleteChallengePool(alloc *StorageAllocation, balances cstate.StateContextI) (err error) {
	if _, err = balances.DeleteTrieNode(challengePoolKey(ssc.ID, alloc.ID)); err != nil {
		return fmt.Errorf("can't delete challenge pool: %v", err)
	}

	return nil
}
