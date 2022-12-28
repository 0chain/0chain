package event

import (
	"0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthorizerAggregate struct {
	gorm.Model

	AuthorizerID string `json:"authorizer_id" gorm:"index:idx_authorizer_aggregate,unique"`
	Round        int64  `json:"round" gorm:"index:idx_authorizer_aggregate,unique"`

	Fee           currency.Coin `json:"fee"`
	UnstakeTotal  currency.Coin `json:"unstake_total"`
	TotalStake    currency.Coin `json:"total_stake"`
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

func (a *AuthorizerAggregate) SetTotalStake(value currency.Coin) {
	a.TotalStake = value
}

func (a *AuthorizerAggregate) SetUnstakeTotal(value currency.Coin) {
	a.UnstakeTotal = value
}

func (a *AuthorizerAggregate) SetServiceCharge(value float64) {
	a.ServiceCharge = value
}

func (edb *EventDb) ReplicateAuthorizerAggregate(p common.Pagination) ([]AuthorizerAggregate, error) {
	var snapshots []AuthorizerAggregate

	queryBuilder := edb.Store.Get().
		Model(&AuthorizerAggregate{}).Offset(p.Offset).Limit(p.Limit)

	queryBuilder.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   false,
	})

	result := queryBuilder.Scan(&snapshots)
	if result.Error != nil {
		return nil, result.Error
	}

	return snapshots, nil
}

func (edb *EventDb) updateAuthorizerAggregate(round, pageAmount int64, gs *globalSnapshot) {
	count, err := edb.GetAuthorizerCount()
	if err != nil {
		logging.Logger.Error("update_authorizer_aggregates", zap.Error(err))
		return
	}
	size, currentPageNumber, subpageCount := paginate(round, pageAmount, count, edb.PageLimit())

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS authorizer_temp_ids "+
		"ON COMMIT DROP AS SELECT id as id FROM authorizers ORDER BY (id, creation_round) LIMIT ? OFFSET ?",
		size, size*currentPageNumber)
	if exec.Error != nil {
		logging.Logger.Error("error creating temp table", zap.Error(exec.Error))
		return
	}

	for i := 0; i < subpageCount; i++ {
		edb.calculateAuthorizerAggregate(gs, round, edb.PageLimit(), int64(i)*edb.PageLimit())
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
	result := edb.Store.Get().
		Raw("SELECT * FROM Authorizers WHERE id in (select id from authorizer_temp_ids ORDER BY ID limit ? offset ?)", limit, offset).
		Scan(&currentAuthorizers)
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
		}

		recalculateProviderFields(&old, &current, &aggregate)

		aggregate.Fee = (old.Fee + current.Fee) / 2
		aggregates = append(aggregates, aggregate)

		gs.totalTxnFees += aggregate.Fee
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
