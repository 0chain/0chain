package event

import (
	"0chain.net/chaincore/config"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type AuthorizerAggregate struct {
	model.ImmutableModel

	AuthorizerID string `json:"authorizer_id" gorm:"index:idx_authorizer_aggregate,unique"`
	Round        int64  `json:"round" gorm:"index:idx_authorizer_aggregate,unique"`
	BucketID     int64  `json:"bucket_id"`

	Fee           currency.Coin `json:"fee"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
}

func (a *AuthorizerAggregate) GetTotalStake() currency.Coin {
	return a.TotalStake
}

func (a *AuthorizerAggregate) GetUnstakeTotal() currency.Coin {
	return a.UnstakeTotal
}

func (a *AuthorizerAggregate) GetServiceCharge() float64 {
	return a.ServiceCharge
}

func (a *AuthorizerAggregate) GetTotalRewards() currency.Coin {
	return a.TotalRewards
}

func (a *AuthorizerAggregate) SetTotalStake(value currency.Coin) {
	a.TotalStake = value
}

func (a *AuthorizerAggregate) SetUnstakeTotal(value currency.Coin) {
	a.UnstakeTotal = value
}

func (a *AuthorizerAggregate) SetServiceCharge(value float64) {
	a.ServiceCharge = value
}

func (a *AuthorizerAggregate) SetTotalRewards(value currency.Coin) {
	a.TotalRewards = value
}

func (edb *EventDb) ReplicateAuthorizerAggregate(round int64, authorizerId string) ([]AuthorizerAggregate, error) {
	var snapshots []AuthorizerAggregate
	result := edb.Store.Get().
		Raw("SELECT * FROM authorizer_aggregates WHERE round >= ? AND authorizer_id > ? ORDER BY round, authorizer_id ASC LIMIT 20", round, authorizerId).Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	return snapshots, nil
}

func (edb *EventDb) updateAuthorizerAggregate(round, pageAmount int64, gs *globalSnapshot) {
	currentBucket := round % config.Configuration().ChainConfig.DbSettings().AggregatePeriod

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS authorizer_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM authorizers where bucket_id = ?",
		currentBucket)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	var count int64
	r := edb.Store.Get().Raw("SELECT count(*) FROM authorizer_temp_ids").Scan(&count)
	if r.Error != nil {
		logging.Logger.Error("getting ids count", zap.Error(r.Error))
		return
	}
	if count == 0 {
		return
	}
	pageCount := count / edb.PageLimit()

	for i := int64(0); i <= pageCount; i++ {
		edb.calculateAuthorizerAggregate(gs, round, edb.PageLimit(), i*edb.PageLimit())
	}

}

func (edb *EventDb) calculateAuthorizerAggregate(gs *globalSnapshot, round, limit, offset int64) {

	var ids []string
	r := edb.Store.Get().
		Raw("select id from authorizer_temp_ids ORDER BY ID limit ? offset ?", limit, offset).Scan(&ids)
	if r.Error != nil {
		logging.Logger.Error("getting ids", zap.Error(r.Error))
		return
	}
	logging.Logger.Debug("getting ids", zap.Strings("ids", ids))

	var currentAuthorizers []Authorizer
	result := edb.Store.Get().Model(&Authorizer{}).
		Where("authorizers.id in (select id from authorizer_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Joins("Rewards").
		Find(&currentAuthorizers)
	if result.Error != nil {
		logging.Logger.Error("getting current Authorizers", zap.Error(result.Error))
		return
	}
	logging.Logger.Debug("Authorizer_snapshot", zap.Int("total_current_Authorizers", len(currentAuthorizers)))

	if round <= edb.AggregatePeriod() && len(currentAuthorizers) > 0 {
		if err := edb.addAuthorizerSnapshot(currentAuthorizers); err != nil {
			logging.Logger.Error("saving Authorizers snapshots", zap.Error(err))
		}
	}

	oldAuthorizers, err := edb.getAuthorizerSnapshots(limit, offset)
	if err != nil {
		logging.Logger.Error("getting Authorizer snapshots", zap.Error(err))
		return
	}
	logging.Logger.Debug("Authorizer_snapshot", zap.Int("total_old_Authorizers", len(oldAuthorizers)))

	var aggregates []AuthorizerAggregate
	for _, current := range currentAuthorizers {
		old, found := oldAuthorizers[current.ID]
		if !found {
			continue
		}

		//agg := recalculateProviderFields(old, &current, aggregate)
		aggregate := AuthorizerAggregate{
			Round:        round,
			AuthorizerID: current.ID,
			BucketID:     current.BucketId,
			TotalRewards: (old.TotalRewards + current.Rewards.TotalRewards) / 2,
		}

		recalculateProviderFields(&old, &current, &aggregate)

		aggregate.Fee = (old.Fee + current.Fee) / 2
		aggregates = append(aggregates, aggregate)

		gs.totalTxnFees += aggregate.Fee
		gs.TotalRewards += int64(aggregate.TotalRewards - old.TotalRewards)
	}
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			logging.Logger.Error("saving aggregates", zap.Error(result.Error))
		}
	}
	logging.Logger.Debug("Authorizer_snapshot", zap.Int("aggregates", len(aggregates)))

	if len(currentAuthorizers) > 0 {
		if err := edb.addAuthorizerSnapshot(currentAuthorizers); err != nil {
			logging.Logger.Error("saving Authorizer snapshots", zap.Error(err))
		}
	}

	logging.Logger.Debug("Authorizer_snapshot", zap.Int("current_Authorizers", len(currentAuthorizers)))

	// update global snapshot object

	ttf, err := gs.totalTxnFees.Int64()
	if err != nil {
		logging.Logger.Error("converting write price to coin", zap.Error(err))
		return
	}
	gs.AverageTxnFee = ttf / gs.TransactionsCount
}
