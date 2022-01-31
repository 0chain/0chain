package event

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type DelegatePool struct {
	gorm.Model

	PoolId string `json:"pool_id"`

	// foreign keys todo: when user(ID) created, enable it
	DelegateId   string `json:"delegate_id"`
	ProviderId   string `json:"provider_id"`
	ProviderType int    `json:"provider_type"`

	Reward       int64 `json:"reward"`
	TotalReward  int64 `json:"total_reward"`
	TotalPenalty int64 `json:"total_penalty"`
	Status       int   `json:"status"`
	RoundCreated int64 `json:"round_created"`
}

func (edb *EventDb) addDelegatePoolReward(reward int64, id string) error {
	dp, err := edb.getDelegateByPoolId(id)
	if err != nil {
		return err
	}
	sp, err := edb.getStakePool(dp.ProviderId, dp.ProviderType)
	if reward > 0 {
		dp.TotalReward += reward
		sp.TotalRewards += reward
	} else {
		dp.TotalPenalty -= reward
		sp.TotalPenalty -= reward
	}
	if err := edb.addOrOverwriteStakePool(sp); err != nil {
		return err
	}
	if err := edb.addOrOverwriteDelegatePool(dp); err != nil {
		return err
	}

	return nil
}

func (edb *EventDb) getDelegateByPoolId(poolId string) (DelegatePool, error) {
	var dp DelegatePool
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{PoolId: poolId}).
		Find(&dp)
	return dp, result.Error
}

func (edb *EventDb) deleteDelegatePool(id string) error {
	result := edb.Store.Get().
		Where(&DelegatePool{PoolId: id}).
		Delete(&DelegatePool{})

	return result.Error
}

func (edb *EventDb) getDelegatesByBlobber(blobberId string) (*[]DelegatePool, error) {
	var dps []DelegatePool
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{ProviderId: blobberId}).
		Find(&dps)
	return &dps, result.Error
}

func (edb *EventDb) getDelegatesByUser(delegateId string) (*[]DelegatePool, error) {
	var dps []DelegatePool
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{DelegateId: delegateId}).
		Find(&dps)
	return &dps, result.Error
}

func (edb *EventDb) overwriteDelegatePool(dp DelegatePool) error {
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{PoolId: dp.PoolId}).
		Updates(&dp)
	return result.Error
}

func (edb *EventDb) addOrOverwriteDelegatePool(dp DelegatePool) error {
	exists, err := dp.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteDelegatePool(dp)
	}

	result := edb.Store.Get().Create(&dp)
	return result.Error
}

func (wm *DelegatePool) exists(edb *EventDb) (bool, error) {
	var dp DelegatePool

	result := edb.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{PoolId: dp.PoolId}).
		Take(&dp)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for write marker txn: %v, error %v",
			dp.PoolId, result.Error)
	}
	return true, nil
}
