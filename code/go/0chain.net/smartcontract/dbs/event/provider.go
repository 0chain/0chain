package event

import (
	"fmt"
	"math/big"
	"reflect"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm"
)

// TODO: Move to a config file
const healthCheckPeriod = common.Timestamp(1 * time.Minute)
const healthCheckDelayLimit = common.Timestamp(10 * time.Second)

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

// ProviderHealthCheck Update given provider (providerType, providerId) with the given HealthCheckData and calculate aggregates e.g. downtime
func (edb *EventDb) ProviderHealthCheck(providerType spenum.Provider, providerId string, data interface{}) (err error) {
	providerModel, err	  := edb.getProviderModel(providerType)
	if err != nil {
		return fmt.Errorf("provider_health_check: %v", err.Error()) 
	}
	providerInstance, err := edb.getProviderInstance(providerModel, providerId)
	if err != nil {
		return fmt.Errorf("provider_health_check: %v", err.Error())
	}

	healthCheckData, ok := data.(*dbs.DbHealthCheck)
	if !ok  {
		return fmt.Errorf("provider_health_check: invalid data")
	}
	prevHealthCheck := providerInstance.LastHealthCheck
	curHealthCheck 	:= healthCheckData.LastHealthCheck
	diff 			:= curHealthCheck - prevHealthCheck
	updates 		:= &Provider{}
	
	if  diff > (healthCheckPeriod + healthCheckDelayLimit) {
		updates.Downtime = providerInstance.Downtime + uint64(diff)
	}
	updates.LastHealthCheck = curHealthCheck

	if err = edb.updateProvider(providerModel, providerId, updates); err != nil {
		return fmt.Errorf("provider_health_check: %v", err.Error())
	}
	return nil
}

// getProviderModel - Given provider type enum, return a model representing the type of this provider e.g. MinerType => *event.Miner{}
func (edb *EventDb) getProviderModel(providerType spenum.Provider) (interface{}, error) {
	providerModel, ok := ProviderModel[providerType]
	if !ok {
		return nil, fmt.Errorf("invalid provider type %v", providerType)
	}

	return providerModel, nil
}

// getProviderInstance - Given provider model and id, find the corresponding entry to this provider in events_db
func (edb *EventDb) getProviderInstance(providerModel interface{}, providerId string) (*Provider, error) {
	providerInstance := Provider{}
	result := edb.Get().
		Model(providerModel).
		Where("id = ?", providerId).
		Find(&providerInstance)
	if result.Error != nil {
		return nil, fmt.Errorf("cannot fetch provider from events_db, providerType = %v, providerId = %v", reflect.TypeOf(providerModel), providerId)
	}
	return &providerInstance, nil
}

// updateProviderInstance - Given provider model and id, update provider-related date in events_db given a payload
func (edb *EventDb) updateProvider(providerModel interface{}, providerId string, providerUpdates *Provider) error {
	result := edb.Get().
		Model(providerModel).
		Where("id = ?", providerId).
		Updates(providerUpdates)
	if result.Error != nil {
		return fmt.Errorf("cannot update provider in events_db, providerType = %v, providerId = %v", reflect.TypeOf(providerModel), providerId)
	}
	return nil
}
