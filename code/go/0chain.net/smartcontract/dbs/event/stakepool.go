package event

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/state"
	"gorm.io/gorm"
)

type StakePool struct {
	gorm.Model

	ProviderId   string `json:"provider_id"`
	ProviderType int    `json:"provider_type"`
	Balance      int64  `json:"balance"`
	Unstake      int64  `json:"unstake"`

	TotalOffers  int64 `json:"total_offers"`
	TotalUnStake int64 `json:"total_un_stake"`

	Reward int64 `json:"reward"`

	TotalRewards int64 `json:"total_rewards"`
	TotalPenalty int64 `json:"total_penalty"`

	MinStake        state.Balance `json:"min_stake"`
	MaxStake        state.Balance `json:"max_stake"`
	MaxNumDelegates int           `json:"num_delegates"`
	ServiceCharge   float64       `json:"service_charge"`
}

func (edb *EventDb) deleteStakePool(id string) error {

	result := edb.Store.Get().
		Where(&StakePool{ProviderId: id}).
		Delete(&StakePool{})

	return result.Error
}

func (edb *EventDb) getStakePool(id string, pType int) (StakePool, error) {
	var sp StakePool
	result := edb.Store.Get().
		Model(&StakePool{}).
		Where(&StakePool{ProviderId: id, ProviderType: pType}).
		Find(&sp)
	return sp, result.Error
}

func (edb *EventDb) overwriteStakePool(sp StakePool) error {
	result := edb.Store.Get().
		Model(&StakePool{}).
		Where(&StakePool{ProviderId: sp.ProviderId}).
		Updates(&sp)
	return result.Error
}

func (edb *EventDb) addOrOverwriteStakePool(sp StakePool) error {
	exists, err := sp.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteStakePool(sp)
	}

	result := edb.Store.Get().Create(&sp)
	return result.Error
}

func (wm *StakePool) exists(edb *EventDb) (bool, error) {
	var sp StakePool

	result := edb.Get().
		Model(&StakePool{}).
		Where(&StakePool{ProviderId: sp.ProviderId}).
		Take(&sp)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for write marker txn: %v, error %v",
			sp.ProviderId, result.Error)
	}
	return true, nil
}
