package event

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type DelegatePool struct {
	gorm.Model

	PoolID       string `json:"pool_id"`
	ProviderType int    `json:"provider_type"`
	ProviderID   string `json:"provider_id"`
	DelegateID   string `json:"delegate_id"`

	Balance      int64 `json:"balance"`
	Reward       int64 `json:"reward"`       // unclaimed reward
	TotalReward  int64 `json:"total_reward"` // total reward paid to pool
	TotalPenalty int64 `json:"total_penalty"`
	Status       int   `json:"status"`
	RoundCreated int64 `json:"round_created"`
}

func (edb *EventDb) overwriteDelegatePool(sp DelegatePool) error {
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{
			PoolID:       sp.PoolID,
			ProviderType: sp.ProviderType,
		}).Updates(map[string]interface{}{
		"delegate_id":   sp.DelegateID,
		"provider_type": sp.ProviderType,
		"provider_id":   sp.ProviderID,
		"pool_id":       sp.PoolID,
		"balance":       sp.Balance,
		"reward":        sp.Reward,
		"total_reward":  sp.TotalReward,
		"total_penalty": sp.TotalPenalty,
		"status":        sp.Status,
		"round_created": sp.RoundCreated,
	})
	return result.Error
}

func (sp *DelegatePool) exists(edb *EventDb) (bool, error) {
	var dp DelegatePool
	result := edb.Store.Get().Model(&DelegatePool{}).Where(&DelegatePool{
		ProviderType: sp.ProviderType,
		PoolID:       sp.PoolID,
	}).Take(&dp)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if result.Error != nil {
		return false, fmt.Errorf("failed to check Curator existence %v, error %v",
			dp, result.Error)
	}
	return true, nil
}

func (edb *EventDb) updateReward(reward int64, dp DelegatePool) error {
	dpu := dbs.NewDelegatePoolUpdate(dp.PoolID, dp.ProviderID, dp.ProviderType)

	if dp.ProviderType == int(spenum.Blobber) && reward < 0 {
		dpu.Updates["total_penalty"] = dp.TotalPenalty - reward
	} else {
		dpu.Updates["reward"] = dp.Reward + reward
		dpu.Updates["total_reward"] = dp.TotalReward + reward
	}
	if err := edb.updateDelegatePool(*dpu); err != nil {
		return nil
	}
	return nil
}

func (edb *EventDb) GetDelegatePools(id string, pType int) ([]DelegatePool, error) {
	var dps []DelegatePool
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{
			ProviderType: pType,
			ProviderID:   id,
		}).
		Not(&DelegatePool{Status: int(spenum.Deleted)}).
		Find(&dps)
	if result.Error != nil {
		return nil, fmt.Errorf("error getting delegate pools, %v", result.Error)
	}
	return dps, nil
}

func (edb *EventDb) GetUserDelegatePools(userId string, pType int) ([]DelegatePool, error) {
	var dps []DelegatePool
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{
			ProviderType: pType,
			DelegateID:   userId,
		}).
		Not(&DelegatePool{Status: int(spenum.Deleted)}).
		Find(&dps)
	if result.Error != nil {
		return nil, fmt.Errorf("error getting delegate pools, %v", result.Error)
	}
	return dps, nil
}

func (edb *EventDb) updateDelegatePool(updates dbs.DelegatePoolUpdate) error {
	var dp = DelegatePool{
		ProviderID:   updates.ProviderId,
		ProviderType: updates.ProviderType,
		PoolID:       updates.PoolId,
	}
	exists, err := dp.exists(edb)

	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("stakepool %v not in database cannot update",
			dp.ProviderID)
	}

	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{
			ProviderType: dp.ProviderType,
			ProviderID:   dp.ProviderID,
		}).
		Updates(updates.Updates)
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
