package event

import (
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

type MinerAggregate struct {
	model.ImmutableModel
	MinerID         string           `json:"miner_id" gorm:"index:idx_miner_aggregate,unique"`
	Round           int64            `json:"round" gorm:"index:idx_miner_aggregate,unique"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	Fees            currency.Coin    `json:"fees"`
	TotalStake      currency.Coin    `json:"total_stake"`
	TotalRewards    currency.Coin    `json:"total_rewards"`
	BlocksFinalised int64            `json:"blocks_finalised"`
	ServiceCharge   float64          `json:"service_charge"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`
}

func (m *MinerAggregate) GetTotalStake() currency.Coin {
	return m.TotalStake
}

func (m *MinerAggregate) GetServiceCharge() float64 {
	return m.ServiceCharge
}

func (m *MinerAggregate) GetTotalRewards() currency.Coin {
	return m.TotalRewards
}

func (m *MinerAggregate) SetTotalStake(value currency.Coin) {
	m.TotalStake = value
}

func (m *MinerAggregate) SetServiceCharge(value float64) {
	m.ServiceCharge = value
}

func (m *MinerAggregate) SetTotalRewards(value currency.Coin) {
	m.TotalRewards = value
}

func (edb *EventDb) CreateMinerAggregates(miners []*Miner, round int64) error {
	var aggregates []MinerAggregate
	for _, m := range miners {
		aggregate := MinerAggregate{
			Round:           round,
			MinerID:         m.ID,
			LastHealthCheck: m.LastHealthCheck,
			BlocksFinalised: m.BlocksFinalised,
		}
		recalculateProviderFields(m, &aggregate)
		aggregate.Fees = m.Fees
		aggregates = append(aggregates, aggregate)
	}
	if len(aggregates) > 0 {
		if result := edb.Store.Get().Create(&aggregates); result.Error != nil {
			return result.Error
		}
	}
	return nil
}
