package event

import (
	"errors"
	"fmt"

	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
	"gorm.io/gorm"
)

type DelegatePool struct {
	gorm.Model

	PoolID       string          `json:"pool_id" gorm:"uniqueIndex:ppp;index:idx_ddel_active"`
	ProviderType spenum.Provider `json:"provider_type" gorm:"uniqueIndex:ppp;index:idx_dprov_active,priority:2;index:idx_ddel_active,priority:2" `
	ProviderID   string          `json:"provider_id" gorm:"uniqueIndex:ppp;index:idx_dprov_active,priority:1;index:idx_ddel_active,priority:2"`
	DelegateID   string          `json:"delegate_id" gorm:"index:idx_ddel_active,priority:2;index:idx_del_id"` //todo think of changing priority for idx_ddel_active

	Balance      currency.Coin `json:"balance"`
	Reward       currency.Coin `json:"reward"`       // unclaimed reward
	TotalReward  currency.Coin `json:"total_reward"` // total reward paid to pool
	TotalPenalty currency.Coin `json:"total_penalty"`
	Status       int           `json:"status" gorm:"index:idx_dprov_active,priority:3;index:idx_ddel_active,priority:3"`
	RoundCreated int64         `json:"round_created"`
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

func (edb *EventDb) GetDelegatePools(id string, pType spenum.Provider) ([]DelegatePool, error) {
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

func (edb *EventDb) GetUserTotalLocked(id string) (int64, error) {
	res := int64(0)
	err := edb.Store.Get().Table("delegate_pools").Select("coalesce(sum(balance),0)").
		Where("delegate_id = ?", id).Row().Scan(&res)
	return res, err
}

func (edb *EventDb) GetUserDelegatePools(userId string, pType spenum.Provider) ([]DelegatePool, error) {
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
			PoolID:       dp.PoolID,
		}).
		Updates(updates.Updates)
	return result.Error
}

func mergeAddDelegatePoolsEvents() *eventsMergerImpl[DelegatePool] {
	return newEventsMerger[DelegatePool](TagAddOrOverwriteDelegatePool, withUniqueEventOverwrite())
}

func (edb *EventDb) addOrOverwriteDelegatePools(dps []DelegatePool) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider_id"}, {Name: "provider_type"}, {Name: "pool_id"}},
		UpdateAll: true,
	}).Create(&dps).Error
}
