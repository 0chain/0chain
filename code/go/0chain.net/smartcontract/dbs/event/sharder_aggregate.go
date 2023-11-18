package event

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

type SharderAggregate struct {
	model.ImmutableModel

	SharderID string `json:"sharder_id" gorm:"index:idx_sharder_aggregate,unique"`
	Round     int64  `json:"round" gorm:"index:idx_sharder_aggregate,unique"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	Fees          currency.Coin `json:"fees"`
	TotalStake    currency.Coin `json:"total_stake"`
	TotalRewards  currency.Coin `json:"total_rewards"`
	ServiceCharge float64       `json:"service_charge"`
}

func (s *SharderAggregate) GetTotalStake() currency.Coin {
	return s.TotalStake
}

func (s *SharderAggregate) GetServiceCharge() float64 {
	return s.ServiceCharge
}

func (s *SharderAggregate) GetTotalRewards() currency.Coin {
	return s.TotalRewards
}

func (s *SharderAggregate) SetTotalStake(value currency.Coin) {
	s.TotalStake = value
}

func (s *SharderAggregate) SetServiceCharge(value float64) {
	s.ServiceCharge = value
}

func (s *SharderAggregate) SetTotalRewards(value currency.Coin) {
	s.TotalRewards = value
}

func (edb *EventDb) CreateSharderAggregates(sharders []*Sharder, round int64) error {
	var aggregates []SharderAggregate
	for _, s := range sharders {
		aggregate := SharderAggregate{
			Round:    round,
			SharderID:  s.ID,
		}
		recalculateProviderFields(s, &aggregate)
		aggregate.Fees = s.Fees
		aggregates = append(aggregates, aggregate)
	}
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			return result.Error
		}
	}
	return nil
}
