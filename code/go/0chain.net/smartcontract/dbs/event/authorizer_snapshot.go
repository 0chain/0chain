package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model AuthorizerSnapshot
type AuthorizerSnapshot struct {
	AuthorizerID string `json:"id" gorm:"index"`
	BucketId     int64  `json:"bucket_id"`
	Round        int64  `json:"round"`

	Fee           currency.Coin `json:"fee"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	TotalMint     currency.Coin `json:"total_mint"`
	TotalBurn     currency.Coin `json:"total_burn"`
	ServiceCharge float64       `json:"service_charge"`
	CreationRound int64         `json:"creation_round" gorm:"index"`
	IsKilled      bool          `json:"is_killed"`
	IsShutdown    bool          `json:"is_shutdown"`
}

func (a *AuthorizerSnapshot) IsOffline() bool {
	return a.IsKilled || a.IsShutdown
}

func (a *AuthorizerSnapshot) GetTotalStake() currency.Coin {
	return a.TotalStake
}

func (a *AuthorizerSnapshot) GetServiceCharge() float64 {
	return a.ServiceCharge
}

func (a *AuthorizerSnapshot) GetTotalRewards() currency.Coin {
	return a.TotalRewards
}

func (a *AuthorizerSnapshot) SetTotalStake(value currency.Coin) {
	a.TotalStake = value
}

func (a *AuthorizerSnapshot) SetServiceCharge(value float64) {
	a.ServiceCharge = value
}

func (a *AuthorizerSnapshot) SetTotalRewards(value currency.Coin) {
	a.TotalRewards = value
}

func (edb *EventDb) getAuthorizerSnapshots(limit, offset int64) (map[string]AuthorizerSnapshot, error) {
	var snapshots []AuthorizerSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM authorizer_snapshots WHERE authorizer_id in (select id from authorizer_old_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	var mapSnapshots = make(map[string]AuthorizerSnapshot, len(snapshots))
	logging.Logger.Debug("get_authorizer_snapshot", zap.Int("snapshots selected", len(snapshots)))
	logging.Logger.Debug("get_authorizer_snapshot", zap.Int64("snapshots rows selected", result.RowsAffected))

	for _, snapshot := range snapshots {
		mapSnapshots[snapshot.AuthorizerID] = snapshot
	}

	result = edb.Store.Get().Where("authorizer_id IN (select id from authorizer_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).Delete(&AuthorizerSnapshot{})
	logging.Logger.Debug("get_authorizer_snapshot", zap.Int64("deleted rows", result.RowsAffected))
	return mapSnapshots, result.Error
}

func (edb *EventDb) addAuthorizerSnapshot(authorizers []Authorizer, round int64) error {
	var snapshots []AuthorizerSnapshot
	for _, authorizer := range authorizers {
		snapshots = append(snapshots, AuthorizerSnapshot{
			AuthorizerID:  authorizer.ID,
			Round:         round,
			BucketId:      authorizer.BucketId,
			Fee:           authorizer.Fee,
			TotalStake:    authorizer.TotalStake,
			ServiceCharge: authorizer.ServiceCharge,
			CreationRound: authorizer.CreationRound,
			TotalRewards:  authorizer.Rewards.TotalRewards,
			TotalMint:     authorizer.TotalMint,
			TotalBurn:     authorizer.TotalBurn,
			IsKilled:      authorizer.IsKilled,
			IsShutdown:    authorizer.IsShutdown,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
