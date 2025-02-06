package event

import (
	"fmt"

	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm/clause"

	"github.com/guregu/null"
	"gorm.io/gorm"

	"0chain.net/smartcontract/dbs"
)

type Sharder struct {
	Provider
	N2NHost   string `gorm:"column:n2n_host"`
	Host      string
	Port      int
	Path      string
	PublicKey string
	ShortName string
	BuildTag  string
	Delete    bool
	Fees      currency.Coin
	Active    bool

	CreationRound int64 `json:"creation_round" gorm:"index:idx_sharder_creation_round"`
}

func (m *Sharder) TableName() string {
	return "sharders"
}

func (s Sharder) GetID() string {
	return s.ID
}

func (s *Sharder) GetTotalStake() currency.Coin {
	return s.TotalStake
}

func (s *Sharder) GetServiceCharge() float64 {
	return s.ServiceCharge
}

func (s *Sharder) GetTotalRewards() currency.Coin {
	return s.Rewards.TotalRewards
}

func (s *Sharder) SetTotalStake(value currency.Coin) {
	s.TotalStake = value
}

func (s *Sharder) SetServiceCharge(value float64) {
	s.ServiceCharge = value
}

func (s *Sharder) SetTotalRewards(value currency.Coin) {
	s.Rewards.TotalRewards = value
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
		ProviderRewards //nolint
	}
	var s Sharder
	var dps []DelegatePool

	result := edb.Get().
		Table("sharders").
		Joins("left join provider_rewards on sharders.id = provider_rewards.provider_id").
		Joins("left join delegate_pools on sharders.id = delegate_pools.provider_id AND delegate_pools.status = 0").
		Where("sharders.id = ?", id).
		Scan(&sharderDps)
	if result.Error != nil {
		return s, nil, result.Error
	}
	if len(sharderDps) == 0 {
		return s, nil, fmt.Errorf("get sharder %s found no records", id)
	}
	if id != sharderDps[0].Sharder.ID {
		return s, nil, fmt.Errorf("mismatched sharder; want id %s but have id %s", id, sharderDps[0].Sharder.ID)
	}
	s = sharderDps[0].Sharder

	s.Rewards = sharderDps[0].ProviderRewards
	s.Rewards.ProviderID = id
	if len(sharderDps) == 1 && sharderDps[0].DelegatePool.PoolID == "" {
		// The sharder has no delegate pools
		return s, nil, nil
	}
	for i := range sharderDps {
		dps = append(dps, sharderDps[i].DelegatePool)
		if id != sharderDps[i].DelegatePool.ProviderID {
			return s, nil, fmt.Errorf("mismatched sharder id in delegate pool;"+
				"want id %s but have id %s", id, sharderDps[i].DelegatePool.ProviderID)
		}
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
	LastHealthCheck   null.Int
	Rewards           null.Int
	Fees              null.Int
	Active            null.Bool
	IsKilled          null.Bool
}

func (edb *EventDb) GetShardersWithFilterAndPagination(filter SharderQuery, p common2.Pagination) ([]Sharder, error) {
	var sharders []Sharder
	query := edb.Get().
		Preload("Rewards").
		Model(&Sharder{}).
		Where(&filter).Offset(p.Offset).Limit(p.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "creation_round"},
			Desc:   p.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   p.IsDescending,
		})
	return sharders, query.Scan(&sharders).Error
}

func (edb *EventDb) GetStakableShardersWithFilterAndPagination(filter SharderQuery, pagination common2.Pagination) ([]Sharder, error) {
	var sharders []Sharder
	result := edb.Store.Get().
		Select("sharders.*").
		Table("sharders").
		Joins("left join delegate_pools ON delegate_pools.provider_type = 2 AND delegate_pools.provider_id = sharders.id AND delegate_pools.status = 0").
		Where(&filter).
		Group("sharders.id").
		Having("count(delegate_pools.id) < sharders.num_delegates").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "creation_round"},
			Desc:   pagination.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   pagination.IsDescending,
		}).
		Find(&sharders)

	return sharders, result.Error
}

func (edb *EventDb) updateSharder(updates []dbs.DbUpdates) error {
	var errs []error
	for _, update := range updates {
		var sharder = Sharder{Provider: Provider{ID: update.Id}}
		result := edb.Store.Get().
			Model(&Sharder{}).
			Where(&Sharder{Provider: Provider{ID: sharder.ID}}).
			Updates(update.Updates)

		if result.Error != nil {
			errs = append(errs, result.Error)
		}

	}

	if len(errs) > 0 {
		return fmt.Errorf("update sharder: %v", errs)
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

func (edb *EventDb) updateShardersTotalStakes(sharders []Sharder) error {
	var provs []Provider
	for _, s := range sharders {
		provs = append(provs, s.Provider)
	}
	return edb.updateProviderTotalStakes(provs, "sharders")
}

func mergeUpdateSharderTotalStakesEvents() *eventsMergerImpl[Sharder] {
	return newEventsMerger[Sharder](TagUpdateSharderTotalStake, withUniqueEventOverwrite())
}

func mergeSharderHealthCheckEvents() *eventsMergerImpl[dbs.DbHealthCheck] {
	return newEventsMerger[dbs.DbHealthCheck](TagSharderHealthCheck, withUniqueEventOverwrite())
}
