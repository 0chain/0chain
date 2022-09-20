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

func (edb *EventDb) bulkUpdateRewards(providerID string, providerType int, rewards []rewardInfo) error {
	n := len(rewards)
	sql := fmt.Sprintf(`
	UPDATE delegate_pools
		SET reward = reward + data_table.reward_in, total_reward = total_reward + data_table.total_reward_in
	FROM (
		SELECT
		UNNEST(ARRAY[%s]) as provider_id,
		UNNEST(ARRAY[%s]) as provider_type,
		UNNEST(ARRAY[%s]) as pool_id,
		UNNEST(ARRAY[%s]) as reward_in,
		UNNEST(ARRAY[%s]) as total_reward_in
	) AS data_table
	WHERE (delegate_pools.provider_id = data_table.provider_id)
		AND (delegate_pools.provider_type = data_table.provider_type)
		AND (delegate_pools.pool_id = data_table.pool_id)
		AND (delegate_pools.status != ?)`,
		placeholders(n),
		placeholders(n, "integer"),
		placeholders(n),
		placeholders(n, "integer"),
		placeholders(n, "integer"),
	)

	vs := append(makeBulkRewardsValues(providerID, providerType, rewards), spenum.Deleted)
	return edb.Store.Get().Exec(sql, vs...).Error
}

func makeBulkRewardsValues(providerID string, providerType int, rewardInfos []rewardInfo) []interface{} {
	var (
		n             = len(rewardInfos)
		providerIDs   = make([]interface{}, n)
		providerTypes = make([]interface{}, n)
		pools         = make([]interface{}, n)
		rewards       = make([]interface{}, n)
		totalRewards  = make([]interface{}, n)
	)

	for i, r := range rewardInfos {
		providerIDs[i] = providerID
		providerTypes[i] = providerType
		pools[i] = r.pool
		rewards[i] = r.value
		totalRewards[i] = r.value
	}

	return append(append(append(append(providerIDs, providerTypes...), pools...), rewards...), totalRewards...)
}

func (edb *EventDb) bulkUpdatePenalty(providerID string, providerType int, penalties []rewardInfo) error {
	var (
		n  = len(penalties)
		vs = append(makeBulkPenaltyValues(providerID, providerType, penalties), spenum.Deleted)
	)

	sql := fmt.Sprintf(`
	UPDATE delegate_pools
		SET total_penalty = total_penalty + data_table.total_penalty_in
	FROM (
		SELECT
		UNNEST(ARRAY[%s]) as provider_id,
		UNNEST(ARRAY[%s]) as provider_type,
		UNNEST(ARRAY[%s]) as pool_id,
		UNNEST(ARRAY[%s]) as total_penalty_in
	) AS data_table
	WHERE (delegate_pools.provider_id = data_table.provider_id)
		AND (delegate_pools.provider_type = data_table.provider_type)
		AND (delegate_pools.pool_id = data_table.pool_id)
		AND (delegate_pools.status != ?)`,
		placeholders(n),
		placeholders(n, "integer"),
		placeholders(n),
		placeholders(n, "integer"),
	)

	return edb.Store.Get().Exec(sql, vs...).Error
}

func makeBulkPenaltyValues(providerID string, providerType int, penaltyInfos []rewardInfo) []interface{} {
	var (
		n             = len(penaltyInfos)
		providerIDs   = make([]interface{}, n)
		providerTypes = make([]interface{}, n)
		pools         = make([]interface{}, n)
		penalties     = make([]interface{}, n)
	)

	for i, r := range penaltyInfos {
		providerIDs[i] = providerID
		providerTypes[i] = providerType
		pools[i] = r.pool
		penalties[i] = r.value
	}

	return append(append(append(providerIDs, providerTypes...), pools...), penalties...)
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
