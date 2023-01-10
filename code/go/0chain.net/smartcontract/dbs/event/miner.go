package event

import (
	common2 "0chain.net/smartcontract/common"
	"fmt"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"github.com/guregu/null"
	"gorm.io/gorm"
)

type Miner struct {
	Provider
	N2NHost   string `gorm:"column:n2n_host"`
	Host      string
	Port      int
	Path      string
	PublicKey string
	ShortName string
	BuildTag  string

	Delete          bool
	LastHealthCheck common.Timestamp
	Fees            currency.Coin
	Active          bool
	Longitude       float64
	Latitude        float64

	CreationRound int64 `json:"creation_round" gorm:"index:idx_miner_creation_round"`
}

// swagger:model MinerGeolocation
type MinerGeolocation struct {
	MinerID   string  `json:"miner_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (edb *EventDb) GetMiner(id string) (Miner, error) {
	var miner Miner
	return miner, edb.Store.Get().
		Preload("Rewards").
		Model(&Miner{}).
		Where(&Miner{Provider: Provider{ID: id}}).
		First(&miner).Error
}

func (edb *EventDb) GetMinerWithDelegatePools(id string) (Miner, []DelegatePool, error) {
	var minerDps []struct {
		Miner
		DelegatePool
		ProviderRewards
	}
	var m Miner
	var dps []DelegatePool

	result := edb.Get().Preload("Rewards").
		Table("miners").
		Joins("left join provider_rewards on miners.id = provider_rewards.provider_id").
		Joins("left join delegate_pools on miners.id = delegate_pools.provider_id").
		Where("miners.id = ?", id).
		Scan(&minerDps)
	if result.Error != nil {
		return m, nil, result.Error
	}
	logging.Logger.Info("piers GetMinerWithDelegatePools",
		zap.Any("miner and dps", minerDps))
	if len(minerDps) == 0 {
		return m, nil, fmt.Errorf("get miner %s found no records", id)
	}
	m = minerDps[0].Miner
	m.Rewards = minerDps[0].ProviderRewards
	for i := range minerDps {
		dps = append(dps, minerDps[i].DelegatePool)
		logging.Logger.Info("piers GetMinerWithDelegatePools",
			zap.Any("minerDps", minerDps[i]),
			zap.Any("Miner", minerDps[i].Miner),
			zap.Any("DelegatePool", minerDps[i].DelegatePool),
			zap.Any("ProviderRewards", minerDps[i].ProviderRewards),
		)
	}

	return m, dps, nil
}

type MinerQuery struct {
	gorm.Model
	MinerID           null.String
	N2NHost           null.String
	Host              null.String
	Port              null.Int
	Path              null.String
	PublicKey         null.String
	ShortName         null.String
	BuildTag          null.String
	TotalStaked       currency.Coin
	Delete            null.Bool
	DelegateWallet    null.String
	ServiceCharge     null.Float
	NumberOfDelegates null.Int
	MinStake          null.Int
	MaxStake          null.Int
	LastHealthCheck   null.Int
	Rewards           null.Int
	Fees              null.Int
	Active            null.Bool
	Longitude         null.Float
	Latitude          null.Float
}

func (m *Miner) GetTotalStake() currency.Coin {
	return m.TotalStake
}

func (m *Miner) GetUnstakeTotal() currency.Coin {
	return m.UnstakeTotal
}

func (m *Miner) GetServiceCharge() float64 {
	return m.ServiceCharge
}

func (m *Miner) SetTotalStake(value currency.Coin) {
	m.TotalStake = value
}

func (m *Miner) SetUnstakeTotal(value currency.Coin) {
	m.UnstakeTotal = value
}

func (m *Miner) SetServiceCharge(value float64) {
	m.ServiceCharge = value
}

func (edb *EventDb) GetMinersWithFiltersAndPagination(filter MinerQuery, p common2.Pagination) ([]Miner, error) {
	var miners []Miner
	query := edb.Get().
		Preload("Rewards").
		Model(&Miner{}).
		Where(&filter).Offset(p.Offset).Limit(p.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_at"},
			Desc:   p.IsDescending,
		})
	return miners, query.Scan(&miners).Error
}

func (edb *EventDb) GetMinerGeolocations(filter MinerQuery, p common2.Pagination) ([]MinerGeolocation, error) {
	var minerLocations []MinerGeolocation
	query := edb.Get().Model(&Miner{}).Where(&filter).Offset(p.Offset).Limit(p.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_at"},
		Desc:   p.IsDescending,
	})
	result := query.Scan(&minerLocations)

	return minerLocations, result.Error
}

func (edb *EventDb) GetMinersFromQuery(query interface{}) ([]Miner, error) {
	var miners []Miner

	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Miner{}).
		Where(query).
		Find(&miners)

	return miners, result.Error
}

func (edb *EventDb) CountActiveMiners() (int64, error) {
	var count int64

	result := edb.Store.Get().
		Model(&Miner{}).
		Where("active = ?", true).
		Count(&count)

	return count, result.Error
}

func (edb *EventDb) CountInactiveMiners() (int64, error) {
	var count int64

	result := edb.Store.Get().
		Model(&Miner{}).
		Where("active = ?", false).
		Count(&count)

	return count, result.Error
}

func (edb *EventDb) GetMinersTotalStake() (int64, error) {
	var count int64

	err := edb.Store.Get().Table("miners").Select("sum(total_stake)").Row().Scan(&count)
	return count, err
}

func (edb *EventDb) GetMiners() ([]Miner, error) {
	var miners []Miner

	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Miner{}).
		Find(&miners)

	return miners, result.Error
}

func (edb *EventDb) GetMinerCount() (int64, error) {
	var count int64
	res := edb.Store.Get().Model(Miner{}).Count(&count)

	return count, res.Error
}

func (edb *EventDb) addMiner(miners []Miner) error {
	err := edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&miners).Error
	return err
}

func (edb *EventDb) updateMiner(updates []dbs.DbUpdates) error {
	for i := range updates {
		if err := edb.Store.Get().
			Model(&Miner{}).
			Where(&Miner{Provider: Provider{ID: updates[i].Id}}).
			Updates(updates[i].Updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (edb *EventDb) deleteMiner(id string) error {
	result := edb.Store.Get().
		Where(&Miner{Provider: Provider{ID: id}}).
		Delete(&Miner{})

	return result.Error
}

func NewUpdateMinerTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateMinerTotalStake, Miner{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake,
		},
	}
}
func NewUpdateMinerTotalUnStakeEvent(ID string, unstakeTotal currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateMinerTotalUnStake, Miner{
		Provider: Provider{
			ID:         ID,
			TotalStake: unstakeTotal,
		},
	}
}

func (edb *EventDb) updateMinersTotalStakes(miners []Miner) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake"}),
	}).Create(&miners).Error
}

func (edb *EventDb) updateMinersTotalUnStakes(miners []Miner) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"unstake_total"}),
	}).Create(&miners).Error
}

func mergeUpdateMinerTotalStakesEvents() *eventsMergerImpl[Miner] {
	return newEventsMerger[Miner](TagUpdateMinerTotalStake, withUniqueEventOverwrite())
}
func mergeUpdateMinerTotalUnStakesEvents() *eventsMergerImpl[Miner] {
	return newEventsMerger[Miner](TagUpdateMinerTotalUnStake, withUniqueEventOverwrite())
}

func addMinerLastUpdateRound() eventMergeMiddleware {
	return func(events []Event) ([]Event, error) {
		for i := range events {
			m, ok := events[i].Data.(Miner)
			if !ok {
				return nil, fmt.Errorf(
					"merging, %v shold be a miner", events[i].Data)
			}
			m.RoundLastUpdated = events[i].BlockNumber
			events[i].Data = m
		}
		return events, nil
	}
}

func updateLastRoundUpdatedMiddleware(tag EventTag) *eventsMergerImpl[dbs.DbUpdates] {
	return &eventsMergerImpl[dbs.DbUpdates]{
		tag:         tag,
		middlewares: []eventMergeMiddleware{updateLastUpdateRound()},
	}
}

func updateLastUpdateRound() eventMergeMiddleware {
	return func(events []Event) ([]Event, error) {
		for i := range events {
			updates, ok := events[i].Data.(dbs.DbUpdates)
			if !ok {
				return nil, fmt.Errorf(
					"merging, %v shold be a dbs update", events[i].Data)
			}
			updates.Updates["round_last_updated"] = events[i].BlockNumber
			events[i].Data = updates
			//logging.Logger.Info("piers updateLastUpdateRound", zap.Any("updates", updates))
		}
		return events, nil
	}
}
