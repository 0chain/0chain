package event

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type DelegatePool struct {
	gorm.Model

	PoolID       string `json:"pool_id"`
	ProviderType int    `json:"provider_type" gorm:"index:idx_dprov_active,priority:2;index:idx_ddel_active,priority:2" `
	ProviderID   string `json:"provider_id" gorm:"index:idx_dprov_active,priority:1"`
	DelegateID   string `json:"delegate_id" gorm:"index:idx_ddel_active,priority:1"`

	Balance      currency.Coin `json:"balance"`
	Reward       currency.Coin `json:"reward"`       // unclaimed reward
	TotalReward  currency.Coin `json:"total_reward"` // total reward paid to pool
	TotalPenalty currency.Coin `json:"total_penalty"`
	Status       int           `json:"status" gorm:"index:idx_dprov_active,priority:3;index:idx_ddel_active,priority:3"`
	RoundCreated int64         `json:"round_created"`
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
		ProviderID:   sp.ProviderID,
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

func (edb *EventDb) updateReward(reward int64, dp DelegatePool) (err error) {

	dpu := dbs.NewDelegatePoolUpdate(dp.PoolID, dp.ProviderID, spenum.Provider(dp.ProviderType))

	if dp.ProviderType == int(spenum.Blobber) && reward < 0 {
		dpu.Updates["total_penalty"], err = currency.MinusInt64(dp.TotalPenalty, reward)
		if err != nil {
			return err
		}
	} else {
		dpu.Updates["reward"], err = currency.AddInt64(dp.Reward, reward)
		if err != nil {
			return err
		}
		dpu.Updates["total_reward"], err = currency.AddInt64(dp.TotalReward, reward)
		if err != nil {
			return err
		}
	}
	if err := edb.updateDelegatePool(*dpu); err != nil {
		return err
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
		ProviderType: int(updates.ProviderType),
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
			PoolID:       dp.PoolID,
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
