package event

import (
	"fmt"
	"time"
	"math/big"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"0chain.net/chaincore/config"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm"
)

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
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	DownTime        uint64 `json:"down_time"`
}

func (edb *EventDb) healthCheck(check dbs.HealthCheck) error {
	updates := dbs.NewDbUpdateProvider(check.ProviderId, check.ProviderType)
	updates.Updates["last_health_check"] = check.Now
	timeSinceLastHeathCheck := check.Now - check.LastHealthCheck
	if timeSinceLastHeathCheck.Duration() > check.HealthCheckPeriod {
		timeInactive := timeSinceLastHeathCheck.Duration() - check.HealthCheckPeriod
		downTime, err := edb.providerDownTime(check.ProviderType, check.ProviderId)
		if err != nil {
			return err
		}
		downTime += int64(timeInactive.Seconds())
		updates.Updates["down_time"] = downTime
	}

	return edb.updateProvider(*updates)
}

func (edb *EventDb) ProviderDownTime(
	id string,
	providerType spenum.Provider,
	now common.Timestamp,
	period time.Duration,
) (common.Timestamp, common.Timestamp, common.Timestamp, error) {
	var provider struct {
		DownTime        int64     `json:"down_time"`
		CreatedAt       time.Time `json:"created_at"`
		LastHealthCheck int64     `json:"last_health_check"`
	}
	model, err := providerModel(providerType)
	if err != nil {
		return 0, 0, 0, err
	}
	result := edb.Store.Get().
		Model(&model).
		Select("down_time, created_at, last_health_check").
		Where(providerType.String()+"_id = ?", id).
		Find(&provider)
	if result.Error != nil {
		return 0, 0, 0, result.Error
	}
	lifetime := (now - common.Timestamp(provider.CreatedAt.Unix()))
	downtime := common.Timestamp(provider.DownTime)
	uptime   := lifetime - downtime

	timeSinceLastHeathCheck := now - common.Timestamp(provider.LastHealthCheck)
	if timeSinceLastHeathCheck.Duration() > period {
		downtime += common.Timestamp(timeSinceLastHeathCheck.Duration().Seconds() - period.Seconds())
	}
	return downtime, lifetime, uptime, result.Error
}

func (edb *EventDb) providerDownTime(
	providerType spenum.Provider,
	providerId string,
) (int64, error) {
	var downTime int64
	model, err := providerModel(providerType)
	if err != nil {
		return 0, err
	}
	result := edb.Store.Get().
		Model(&model).
		Select("down_time").
		Where(providerType.String()+"_id = ?", providerId).
		Find(&downTime)
	return downTime, result.Error
}

func (edb *EventDb) updateProvider(
	updates dbs.DbUpdateProvider,
) error {
	model, err := providerModel(updates.Type)
	if err != nil {
		return err
	}
	return edb.Store.Get().
		Model(&model).
		Where(updates.Type.String()+"_id = ?", updates.Id).
		Updates(updates.Updates).Error
}

func providerModel(pType spenum.Provider) (interface{}, error) {
	switch pType {
	case spenum.Blobber:
		return Blobber{}, nil
	case spenum.Validator:
		return Validator{}, nil
	case spenum.Miner:
		return Miner{}, nil
	case spenum.Sharder:
		return Sharder{}, nil
	case spenum.Authorizer:
		return &Authorizer{}, nil
	default:
		return nil, fmt.Errorf("unrecognised provider type %v", pType)
	}
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
