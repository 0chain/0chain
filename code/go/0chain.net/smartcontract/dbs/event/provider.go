package event

import (
	"fmt"
	"math/big"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ProviderModel = map[spenum.Provider]interface{}{
	spenum.Miner		: &Miner{},
	spenum.Sharder		: &Sharder{},
	spenum.Authorizer	: &Authorizer{},
	spenum.Blobber		: &Blobber{},
	spenum.Validator	: &Validator{},
}

type Provider struct {
	ID             string `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	BucketId       int64           `gorm:"not null"`
	DelegateWallet string          `json:"delegate_wallet"`
	MinStake       currency.Coin   `json:"min_stake"`
	MaxStake       currency.Coin   `json:"max_stake"`
	NumDelegates   int             `json:"num_delegates"`
	ServiceCharge  float64         `json:"service_charge"`
	UnstakeTotal   currency.Coin   `json:"unstake_total"`
	TotalStake     currency.Coin   `json:"total_stake"`
	Rewards        ProviderRewards `json:"rewards" gorm:"foreignKey:ProviderID"`
	Downtime	   uint64		   `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
}

type ProviderAggregate interface {
	GetTotalStake() currency.Coin
	GetUnstakeTotal() currency.Coin
	GetServiceCharge() float64
	SetTotalStake(value currency.Coin)
	SetUnstakeTotal(value currency.Coin)
	SetServiceCharge(value float64)
}

func recalculateProviderFields(prev, curr, result ProviderAggregate) {
	result.SetTotalStake((curr.GetTotalStake() + prev.GetTotalStake()) / 2)
	result.SetUnstakeTotal((curr.GetUnstakeTotal() + prev.GetUnstakeTotal()) / 2)
	result.SetServiceCharge((curr.GetServiceCharge() + prev.GetServiceCharge()) / 2)
}

func (p *Provider) BeforeCreate(tx *gorm.DB) (err error) {
	intID := new(big.Int)
	intID.SetString(p.ID, 16)

	period := config.Configuration().ChainConfig.DbSettings().AggregatePeriod
	p.BucketId = 0
	if period != 0 {
		p.BucketId = big.NewInt(0).Mod(intID, big.NewInt(period)).Int64()
	}
	return
}

func (edb *EventDb) updateProvidersHealthCheck(updates []dbs.DbHealthCheck, tableName ProviderTable) error {
	logging.Logger.Info("Running update provider health check with data: ", zap.Any("updates", updates), zap.String("tableName", string(tableName)))
	updateExpr := map[string]interface{}{
		"last_health_check": gorm.Expr("excluded.last_health_check"),
		"downtime": gorm.Expr(fmt.Sprintf("%v.downtime + excluded.downtime", tableName)),
		"bucket_id": gorm.Expr(fmt.Sprintf("%v.bucket_id", tableName))
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(updateExpr), // column needed to be updated
	}).Table(string(tableName)).Create(&updates).Error
}