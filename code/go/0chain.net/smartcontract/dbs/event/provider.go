package event

import (
	"fmt"
	"math/big"
	"time"

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

func (edb *EventDb) ReplicateProviderAggregate(round int64, limit int, offset int, provider string, scanInto interface{}) error {
	query := fmt.Sprintf("SELECT * FROM %v_aggregates WHERE round >= %v ORDER BY round, %v_id ASC LIMIT %v OFFSET %v", provider, round, provider, limit, offset)
	result := edb.Store.Get().
		Raw(query).Scan(scanInto)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
