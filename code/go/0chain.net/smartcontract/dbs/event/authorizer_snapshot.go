package event

import (
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// swagger:model AuthorizerSnapshot
type AuthorizerSnapshot struct {
	AuthorizerID string `json:"id" gorm:"index"`
	Round        int64  `json:"round"`

	Fee           currency.Coin `json:"fee"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	ServiceCharge float64       `json:"service_charge"`
}

// nolint
func (edb *EventDb) getAuthorizerSnapshots(limit, offset int64) (map[string]AuthorizerSnapshot, error) {
	var snapshots []AuthorizerSnapshot
	result := edb.Store.Get().
		Raw("SELECT * FROM authorizer_snapshots WHERE authorizer_id in (select id from authorizer_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
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

// nolint
func (edb *EventDb) addAuthorizerSnapshot(authorizers []Authorizer) error {
	var snapshots []AuthorizerSnapshot
	for _, authorizer := range authorizers {
		snapshots = append(snapshots, AuthorizerSnapshot{
			AuthorizerID:  authorizer.ID,
			UnstakeTotal:  authorizer.UnstakeTotal,
			Fee:           authorizer.Fee,
			TotalStake:    authorizer.TotalStake,
			ServiceCharge: authorizer.ServiceCharge,
		})
	}

	return edb.Store.Get().Create(&snapshots).Error
}
