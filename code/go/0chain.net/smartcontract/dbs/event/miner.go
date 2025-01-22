package event

import (
	"errors"
	"fmt"

	common2 "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"0chain.net/smartcontract/dbs"
	"github.com/guregu/null"
	"gorm.io/gorm"
)

type Miner struct {
	Provider
	N2NHost         string `gorm:"column:n2n_host"`
	Host            string
	Port            int
	Path            string
	PublicKey       string
	ShortName       string
	BuildTag        string
	Delete          bool
	Fees            currency.Coin
	Active          bool
	BlocksFinalised int64
	CreationRound   int64 `json:"creation_round" gorm:"index:idx_miner_creation_round"`
}

func (m Miner) GetID() string {
	return m.ID
}

func (edb *EventDb) GetMiner(id string) (Miner, error) {
	var miner Miner
	return miner, edb.Store.Get().
		Preload("Rewards").
		Model(&Miner{}).
		Where(&Miner{Provider: Provider{ID: id}}).
		First(&miner).Error
}

// nolint
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
		Joins("left join delegate_pools on miners.id = delegate_pools.provider_id AND delegate_pools.status = 0").
		Where("miners.id = ?", id).
		Scan(&minerDps)
	if result.Error != nil {
		return m, nil, result.Error
	}

	if len(minerDps) == 0 {
		return m, nil, fmt.Errorf("get miner %s found no records", id)
	}
	if id != minerDps[0].Miner.ID {
		return m, nil, fmt.Errorf("mismatched miner; want id %s but have id %s", id, minerDps[0].Miner.ID)
	}
	m = minerDps[0].Miner

	m.Rewards = minerDps[0].ProviderRewards
	m.Rewards.ProviderID = id
	if len(minerDps) == 1 && minerDps[0].DelegatePool.PoolID == "" {
		// The miner has no delegate pools
		return m, nil, nil
	}
	for i := range minerDps {
		dps = append(dps, minerDps[i].DelegatePool)
		if id != minerDps[i].DelegatePool.ProviderID {
			return m, nil, fmt.Errorf("mismatched miner id in delegate pool;"+
				"want id %s but have id %s", id, minerDps[i].DelegatePool.ProviderID)
		}
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
	LastHealthCheck   null.Int
	Rewards           null.Int
	Fees              null.Int
	Active            null.Bool
	IsKilled          null.Bool
}

func (m *Miner) TableName() string {
	return "miners"
}

func (m *Miner) GetTotalStake() currency.Coin {
	return m.TotalStake
}

func (m *Miner) GetServiceCharge() float64 {
	return m.ServiceCharge
}

func (m *Miner) GetTotalRewards() currency.Coin {
	return m.Rewards.TotalRewards
}

func (m *Miner) SetTotalStake(value currency.Coin) {
	m.TotalStake = value
}

func (m *Miner) SetServiceCharge(value float64) {
	m.ServiceCharge = value
}

func (m *Miner) SetTotalRewards(value currency.Coin) {
	m.Rewards.TotalRewards = value
}

func (edb *EventDb) GetMinersWithFiltersAndPagination(filter MinerQuery, p common2.Pagination) ([]Miner, error) {
	var miners []Miner
	query := edb.Get().
		Preload("Rewards").
		Model(&Miner{}).
		Where(&filter).Offset(p.Offset).Limit(p.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "creation_round"},
			Desc:   p.IsDescending,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   p.IsDescending,
		})

	return miners, query.Scan(&miners).Error
}

func (edb *EventDb) GetStakableMinersWithFiltersAndPagination(filter MinerQuery, pagination common2.Pagination) ([]Miner, error) {
	var miners []Miner
	result := edb.Store.Get().
		Select("miners.*").
		Table("miners").
		Joins("left join delegate_pools ON delegate_pools.provider_type = 1 AND delegate_pools.provider_id = miners.id AND delegate_pools.status = 0").
		Where(&filter).
		Group("miners.id").
		Having("count(delegate_pools.id) < miners.num_delegates").
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
		Find(&miners)

	return miners, result.Error
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
	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&miners).Error
}

func (mn *Miner) exists(edb *EventDb) (bool, error) {

	var miner Miner

	result := edb.Get().
		Model(&Miner{}).
		Where(&Miner{Provider: Provider{ID: mn.ID}}).
		Take(&miner)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for miner %v, error %v",
			mn.ID, result.Error)
	}

	return true, nil
}

func (edb *EventDb) updateMiner(updates []dbs.DbUpdates) error {
	errs := make([]error, 0)
	for _, update := range updates {
		var miner = Miner{Provider: Provider{ID: update.Id}}
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
			Where(&Miner{Provider: Provider{ID: miner.ID}}).
			Updates(update.Updates)

		if result.Error != nil {
			errs = append(errs, result.Error)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors updating miners %v", errs)
	}
	return nil
}

func (edb *EventDb) updateMinerBlocksFinalised(minerID string) error {
	result := edb.Store.Get().
		Model(&Miner{}).
		Where(&Miner{Provider: Provider{ID: minerID}}).
		Update("blocks_finalised", gorm.Expr("blocks_finalised + ?", 1))

	return result.Error
}

func (edb *EventDb) deleteMiner(id string) error {
	logging.Logger.Debug("[mvc] event db: deleting miner", zap.String("id", id))
	result := edb.Store.Get().
		Where(&Miner{Provider: Provider{ID: id}}).
		Delete(&Miner{})

	if result.Error != nil {
		return result.Error
	}

	// delete from provider rewards table
	return edb.Store.Get().Where(&ProviderRewards{ProviderID: id}).Delete(&ProviderRewards{}).Error

	// return result.Error
}

func NewUpdateMinerTotalStakeEvent(ID string, totalStake currency.Coin) (tag EventTag, data interface{}) {
	return TagUpdateMinerTotalStake, Miner{
		Provider: Provider{
			ID:         ID,
			TotalStake: totalStake,
		},
	}
}

func (edb *EventDb) updateMinersTotalStakes(miners []Miner) error {
	var provs []Provider
	for _, s := range miners {
		provs = append(provs, s.Provider)
	}
	return edb.updateProviderTotalStakes(provs, "miners")
}

func mergeUpdateMinerTotalStakesEvents() *eventsMergerImpl[Miner] {
	return newEventsMerger[Miner](TagUpdateMinerTotalStake, withUniqueEventOverwrite())
}

func mergeMinerHealthCheckEvents() *eventsMergerImpl[dbs.DbHealthCheck] {
	return newEventsMerger[dbs.DbHealthCheck](TagMinerHealthCheck, withUniqueEventOverwrite())
}
