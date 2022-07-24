package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/stakepool/spenum"
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
func (cp *challengePool) save(sscKey, allocationID string,
	balances cstate.StateContextI) (err error) {

	r, err := balances.InsertTrieNode(challengePoolKey(sscKey, allocationID), cp)
	logging.Logger.Debug("after save challenge pool", zap.String("root", r))

	return
}

func (cp *challengePool) moveToValidators(sscKey string, reward currency.Coin,
	validators []datastore.Key,
	vSPs []*stakePool,
	balances cstate.StateContextI,
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
		err := sp.DistributeRewards(oneReward, validators[i], spenum.Validator, balances)
		if err != nil {
			return fmt.Errorf("moving to validator %s: %v",
				validators[i], err)
		}
	}
	if bal > 0 {
		for i := 0; i < int(bal); i++ {
			err := vSPs[i].DistributeRewards(1, validators[i], spenum.Validator, balances)
			if err != nil {
				return fmt.Errorf("moving to validator %s: %v",
					validators[i], err)
			}
		}
	}

	cp.ZcnPool.Balance -= reward
	return nil
}

func (cp *challengePool) stat(alloc *StorageAllocation) (
	stat *challengePoolStat) {

	stat = new(challengePoolStat)

	stat.ID = cp.ID
	stat.Balance = cp.Balance
	stat.StartTime = alloc.StartTime
	stat.Expiration = alloc.Until()
	stat.Finalized = alloc.Finalized

	return
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
	creationDate, expiresAt common.Timestamp, balances cstate.StateContextI) (
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

// create, fill and save challenge pool for new allocation
func (ssc *StorageSmartContract) createChallengePool(t *transaction.Transaction,
	alloc *StorageAllocation, balances cstate.StateContextI) (err error) {

	// create related challenge_pool expires with the allocation + challenge
	// completion time
	var cp *challengePool
	cp, err = ssc.newChallengePool(alloc.ID, t.CreationDate, alloc.Until(),
		balances)
	if err != nil {
		return fmt.Errorf("can't create challenge pool: %v", err)
	}

	// don't lock anything here

	// save the challenge pool
	if err = cp.save(ssc.ID, alloc.ID, balances); err != nil {
		return fmt.Errorf("can't save challenge pool: %v", err)
	}

	return
}
