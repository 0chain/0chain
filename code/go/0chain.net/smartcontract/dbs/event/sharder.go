package event

import (
	common2 "0chain.net/smartcontract/common"
	"fmt"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"

	"github.com/guregu/null"
	"gorm.io/gorm"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
)

type Sharder struct {
	Provider
	N2NHost         string `gorm:"column:n2n_host"`
	Host            string
	Port            int
	Path            string
	PublicKey       string
	ShortName       string
	BuildTag        string
	Delete          bool
	LastHealthCheck common.Timestamp
	Fees            currency.Coin
	Active          bool
	Longitude       float64
	Latitude        float64

	CreationRound int64 `json:"creation_round" gorm:"index:idx_sharder_creation_round"`
}

func (s *Sharder) GetTotalStake() currency.Coin {
	return s.TotalStake
}

func (s *Sharder) GetUnstakeTotal() currency.Coin {
	return s.UnstakeTotal
}

func (s *Sharder) GetServiceCharge() float64 {
	return s.ServiceCharge
}

func (s *Sharder) SetTotalStake(value currency.Coin) {
	s.TotalStake = value
}

func (s *Sharder) SetUnstakeTotal(value currency.Coin) {
	s.UnstakeTotal = value
}

func (s *Sharder) SetServiceCharge(value float64) {
	s.ServiceCharge = value
}

// swagger:model SharderGeolocation
type SharderGeolocation struct {
	SharderID string  `json:"sharder_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (edb *EventDb) GetSharderCount() (int64, error) {
	var count int64
	res := edb.Store.Get().Model(Sharder{}).Count(&count)

	return count, res.Error
}

func (edb *EventDb) GetSharder(id string) (Sharder, error) {
	var sharder Sharder
	return sharder, edb.Store.Get().
		Preload("Rewards").
		Model(&Sharder{}).
		Where(&Sharder{Provider: Provider{ID: id}}).
		First(&sharder).Error
}

func (edb *EventDb) GetSharderWithDelegatePools(id string) (Sharder, []DelegatePool, error) {
	var sharderDps []struct {
		Sharder
		DelegatePool
	}
	var s Sharder
	var dps []DelegatePool

	result := edb.Get().Preload("Rewards").
		Table("sharder").
		Joins("left join delegate_pools on sharder.id = delegate_pools.provider_id").
		Where("sharder.id = ?", id).
		Scan(&sharderDps)
	if result.Error != nil {
		return s, nil, result.Error
	}
	if len(sharderDps) == 0 {
		return s, nil, fmt.Errorf("get miner %s found no records", id)
	}
	s = sharderDps[0].Sharder
	for i := range sharderDps {
		dps = append(dps, sharderDps[i].DelegatePool)
	}

	return s, dps, nil
}

func (edb *EventDb) GetShardersFromQuery(query *Sharder) ([]Sharder, error) {
	var sharders []Sharder

	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Sharder{}).
		Where(query).
		Find(&sharders)

	return sharders, result.Error
}

func (edb *EventDb) GetSharders() ([]Sharder, error) {
	var sharders []Sharder

	result := edb.Store.Get().
		Preload("Rewards").
		Model(&Sharder{}).
		Find(&sharders)

	return sharders, result.Error
}

func (edb *EventDb) CountActiveSharders() (int64, error) {
	var count int64

	result := edb.Store.Get().
		Model(&Sharder{}).
		Where("active = ?", true).
		Count(&count)

	return count, result.Error
}

func (edb *EventDb) CountInactiveSharders() (int64, error) {
	var count int64

	result := edb.Store.Get().
		Model(&Sharder{}).
		Where("active = ?", false).
		Count(&count)

	return count, result.Error
}

func (edb *EventDb) GetShardersTotalStake() (int64, error) {
	var count int64

	err := edb.Store.Get().Table("sharders").Select("sum(total_stake)").Row().Scan(&count)
	return count, err
}

func (edb *EventDb) addSharders(sharders []Sharder) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&sharders).Error
}

type SharderQuery struct {
	gorm.Model
	SharderID         null.String
	N2NHost           null.String
	Host              null.String
	Port              null.Int
	Path              null.String
	PublicKey         null.String
	ShortName         null.String
	BuildTag          null.String
	TotalStaked       null.Int
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
	Longitude         null.Int
	Latitude          null.Int
}

func (edb *EventDb) GetShardersWithFilterAndPagination(filter SharderQuery, p common2.Pagination) ([]Sharder, error) {
	var sharders []Sharder
	query := edb.Get().
		Preload("Rewards").
		Model(&Sharder{}).
		Where(&filter).Offset(p.Offset).Limit(p.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_at"},
			Desc:   p.IsDescending,
		})
	return sharders, query.Scan(&sharders).Error
}

func (edb *EventDb) GetSharderGeolocations(filter SharderQuery, p common2.Pagination) ([]SharderGeolocation, error) {
	var sharderLocations []SharderGeolocation
	query := edb.Get().
		Preload("Rewards").
		Model(&Sharder{}).
		Where(&filter).Offset(p.Offset).Limit(p.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_at"},
			Desc:   p.IsDescending,
		})

	result := query.Scan(&sharderLocations)

	return sharderLocations, result.Error
}

func (edb *EventDb) updateSharder(updates []dbs.DbUpdates) error {
	for i := range updates {
		if err := edb.Store.Get().
			Model(&Sharder{}).
			Where(&Sharder{Provider: Provider{ID: updates[i].Id}}).
			Updates(updates[i].Updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (edb *EventDb) deleteSharder(id string) error {
	result := edb.Store.Get().
		Where(&Sharder{Provider: Provider{ID: id}}).
		Delete(&Sharder{})

	return result.Error
}

func NewUpdateSharderTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateSharderTotalStake, Sharder{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake,
		},
	}
}
func NewUpdateSharderTotalUnStakeEvent(ID string, unstakeTotal currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateSharderTotalUnStake, Sharder{
		Provider: Provider{
			ID:         ID,
			TotalStake: unstakeTotal,
		},
	}
}

func (edb *EventDb) updateShardersTotalStakes(sharders []Sharder) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake"}),
	}).Create(&sharders).Error
}

func (edb *EventDb) updateShardersTotalUnStakes(sharders []Sharder) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"unstake_total"}),
	}).Create(&sharders).Error
}

func mergeUpdateSharderTotalStakesEvents() *eventsMergerImpl[Sharder] {
	return newEventsMerger[Sharder](TagUpdateSharderTotalStake, withUniqueEventOverwrite())
}
func mergeUpdateSharderTotalUnStakesEvents() *eventsMergerImpl[Sharder] {
	return newEventsMerger[Sharder](TagUpdateSharderTotalUnStake, withUniqueEventOverwrite())
}
