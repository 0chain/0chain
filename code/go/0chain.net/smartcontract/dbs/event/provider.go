package event

import (
	"fmt"
	"math/big"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm"
)

type Provider struct {
	ID              string `gorm:"primaryKey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	BucketId        int64            `gorm:"not null,default:0"`
	DelegateWallet  string           `json:"delegate_wallet"`
	MinStake        currency.Coin    `json:"min_stake"`
	MaxStake        currency.Coin    `json:"max_stake"`
	NumDelegates    int              `json:"num_delegates"`
	ServiceCharge   float64          `json:"service_charge"`
	UnstakeTotal    currency.Coin    `json:"unstake_total"`
	TotalStake      currency.Coin    `json:"total_stake"`
	Rewards         ProviderRewards  `json:"rewards" gorm:"foreignKey:ProviderID"`
	Downtime        uint64           `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`
}

type ProviderAggregate interface {
	GetTotalStake() currency.Coin
	GetUnstakeTotal() currency.Coin
	GetServiceCharge() float64
	GetTotalRewards() currency.Coin
	SetTotalStake(value currency.Coin)
	SetUnstakeTotal(value currency.Coin)
	SetServiceCharge(value float64)
	SetTotalRewards(value currency.Coin)
}

func recalculateProviderFields(prev, curr, result ProviderAggregate) {
	result.SetTotalStake((curr.GetTotalStake() + prev.GetTotalStake()) / 2)
	result.SetUnstakeTotal((curr.GetUnstakeTotal() + prev.GetUnstakeTotal()) / 2)
	result.SetServiceCharge((curr.GetServiceCharge() + prev.GetServiceCharge()) / 2)
	result.SetTotalRewards((curr.GetTotalRewards() + prev.GetTotalRewards()) / 2)
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

func (edb *EventDb) updateProviderTotalStakes(providers []Provider, tablename string) error {
	var ids []string
	var stakes []int64
	for _, m := range providers {
		ids = append(ids, m.ID)
		i, err := m.TotalStake.Int64()
		if err != nil {
			return err
		}
		stakes = append(stakes, i)
	}

	return CreateBuilder(tablename, "id", ids).
		AddUpdate("total_stake", stakes).Exec(edb).Error
}

func (edb *EventDb) updateProvidersTotalUnStakes(providers []Provider, tablename string) error {
	var ids []string
	var unstakes []int64
	for _, m := range providers {
		ids = append(ids, m.ID)
		i, err := m.UnstakeTotal.Int64()
		if err != nil {
			return err
		}
		unstakes = append(unstakes, i)
	}

	return CreateBuilder(tablename, "id", ids).
		AddUpdate("unstake_total", unstakes).Exec(edb).Error
}

func (edb *EventDb) updateProvidersHealthCheck(updates []dbs.DbHealthCheck, tableName ProviderTable) error {
	table := string(tableName)

	var ids []string
	var lastHealthCheck []int64
	var downtime []int64
	for _, u := range updates {
		ids = append(ids, u.ID)
		lastHealthCheck = append(lastHealthCheck, int64(u.LastHealthCheck))
		downtime = append(downtime, int64(u.Downtime))
	}

	return CreateBuilder(table, "id", ids).
		AddUpdate("downtime", downtime, table+".downtime + t.downtime").
		AddUpdate("last_health_check", lastHealthCheck).Exec(edb).Error
}

func (edb *EventDb) ReplicateProviderAggregates(round int64, limit int, offset int, provider string, scanInto interface{}) error {
	query := fmt.Sprintf("SELECT * FROM %v_aggregates WHERE round >= %v ORDER BY round, %v_id ASC LIMIT %v OFFSET %v", provider, round, provider, limit, offset)
	result := edb.Store.Get().
		Raw(query).Scan(scanInto)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func providerToTableName(pType spenum.Provider) string {
	return pType.String() + "s"
}

func splitProviders(
	providers []dbs.Provider,
) map[spenum.Provider][]string {
	idSlices := make(map[spenum.Provider][]string, 5)
	for _, provider := range providers {
		var ids []string
		ids, _ = idSlices[provider.ProviderType]
		ids = append(ids, provider.ProviderId)
		idSlices[provider.ProviderType] = ids
	}
	return idSlices
}

func (edb *EventDb) providersSetBoolean(providers []dbs.Provider, field string, value bool) error {
	splitProviders := splitProviders(providers)
	for pType, ids := range splitProviders {
		table := providerToTableName(pType)
		var values []bool
		for i := 0; i < len(ids); i++ {
			values = append(values, value)
		}
		if err := edb.setBoolean(table, ids, field, values); err != nil {
			logging.Logger.Error("updating boolean field "+table+"."+field,
				zap.Error(err))
		}
	}
	return nil
}

func (edb *EventDb) setBoolean(
	table string,
	ids []string,
	column string,
	values []bool,
) error {
	return CreateBuilder(table, "id", ids).
		AddUpdate(column, values).
		Exec(edb).Error
}
