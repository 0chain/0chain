package event

import (
	"errors"
	"fmt"

	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"github.com/guregu/null"
	"gorm.io/gorm"
)

type Miner struct {
	*Provider
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
		Where(&Miner{Provider: &Provider{ID: id}}).
		First(&miner).Error
}

//func (edb *EventDb) minerAggregateStats(id string) (*providerAggregateStats, error) {
//	var miner providerAggregateStats
//	result := edb.Store.Get().
//		Model(&Miner{}).
//		Where(&Miner{ID: id}).
//		First(&miner)
//	if result.Error != nil {
//		return nil, fmt.Errorf("error retrieving miner %v, error %v",
//			id, result.Error)
//	}
//
//	return &miner, nil
//}

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

func (edb *EventDb) addOrOverwriteMiner(miners []Miner) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&miners).Error
}

func (mn *Miner) exists(edb *EventDb) (bool, error) {

	var miner Miner

	result := edb.Get().
		Model(&Miner{}).
		Where(&Miner{Provider: &Provider{ID: mn.ID}}).
		Take(&miner)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for miner %v, error %v",
			mn.ID, result.Error)
	}

	return true, nil
}

func (edb *EventDb) updateMiner(updates dbs.DbUpdates) error {

	var miner = Miner{Provider: &Provider{ID: updates.Id}}
	exists, err := miner.exists(edb)

	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("miner %v not in database cannot update",
			miner.ID)
	}

	result := edb.Store.Get().
		Model(&Miner{}).
		Where(&Miner{Provider: &Provider{ID: miner.ID}}).
		Updates(updates.Updates)

	return result.Error
}

func (edb *EventDb) deleteMiner(id string) error {

	result := edb.Store.Get().
		Where(&Miner{Provider: &Provider{ID: id}}).
		Delete(&Miner{})

	return result.Error
}

func NewUpdateMinerTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateMinerTotalStake, Miner{
		Provider: &Provider{
			ID:         ID,
			TotalStake: totalStake,
		},
	}
}

func (edb *EventDb) updateMinersTotalStakes(miners []Miner) error {
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake"}),
	}).Create(&miners).Error
}

func mergeUpdateMinerTotalStakesEvents() *eventsMergerImpl[Miner] {
	return newEventsMerger[Miner](TagUpdateMinerTotalStake, withMinerTotalStakesAdded())
}

func withMinerTotalStakesAdded() eventMergeMiddleware {
	return withEventMerge(func(a, b *Miner) (*Miner, error) {
		//use the newer one
		return b, nil
	})
}
