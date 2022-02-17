package event

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"github.com/guregu/null"
	"gorm.io/gorm"
)

type Miner struct {
	gorm.Model
	MinerID           string           `json:"id",gorm:"uniqueIndex"`
	N2NHost           string           `json:"n2n_host,gorm:"column:n2n_host"`
	Host              string           `json:"host"`
	Port              int              `json:"port"`
	Path              string           `json:"path"`
	PublicKey         string           `json:"public_key"`
	ShortName         string           `json:"short_name"`
	BuildTag          string           `json:"build_tag"`
	TotalStaked       state.Balance    `json:"total_stake"`
	Delete            bool             `json:"delete"`
	DelegateWallet    string           `json:"delegate_wallet"`
	ServiceCharge     float64          `json:"service_charge"`
	NumberOfDelegates int              `json:"number_of_delegates"`
	MinStake          state.Balance    `json:"min_stake"`
	MaxStake          state.Balance    `json:"max_stake"`
	LastHealthCheck   common.Timestamp `json:"last_health_check"`
	Rewards           state.Balance    `json:"rewards"`
	Fees              state.Balance    `json:"fees"`
	Active            bool             `json:"active"`
	Longitude         int64            `json:"longitude"`
	Latitude          int64            `json:"latitude"`
}

func (edb *EventDb) GetMiner(id string) (*Miner, error) {

	var miner Miner

	result := edb.Store.Get().
		Model(&Miner{}).
		Where(&Miner{MinerID: id}).
		First(&miner)

	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving miner %v, error %v",
			id, result.Error)
	}

	return &miner, nil

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
	TotalStaked       state.Balance
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

func (edb *EventDb) GetMinersWithFiltersAndPagination(filter MinerQuery, offset, limit int) ([]Miner, error) {
	var miners []Miner
	query := edb.Get().Debug().Model(&Miner{}).Where(&filter)
	if offset != -1 {
		query = query.Offset(offset)
	}
	if limit != -1 {
		query = query.Limit(limit)
	}
	return miners, query.Scan(&miners).Error
}

func (edb *EventDb) GetMinersFromQuery(query interface{}) ([]Miner, error) {

	var miners []Miner

	result := edb.Store.Get().
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

	err := edb.Store.Get().Table("miners").Select("sum(total_staked)").Row().Scan(&count)
	return count, err
}

func (edb *EventDb) GetMiners() ([]Miner, error) {

	var miners []Miner

	result := edb.Store.Get().
		Model(&Miner{}).
		Find(&miners)

	return miners, result.Error
}

func (edb *EventDb) addMiner(miner Miner) error {

	result := edb.Store.Get().Create(&miner)

	return result.Error
}

func (edb *EventDb) overwriteMiner(miner Miner) error {

	result := edb.Store.Get().
		Model(&Miner{}).
		Where(&Miner{MinerID: miner.MinerID}).
		Updates(map[string]interface{}{
			"n2n_host":            miner.N2NHost,
			"host":                miner.Host,
			"port":                miner.Port,
			"path":                miner.Path,
			"public_key":          miner.PublicKey,
			"short_name":          miner.ShortName,
			"build_tag":           miner.BuildTag,
			"total_staked":        miner.TotalStaked,
			"delete":              miner.Delete,
			"delegate_wallet":     miner.DelegateWallet,
			"service_charge":      miner.ServiceCharge,
			"number_of_delegates": miner.NumberOfDelegates,
			"min_stake":           miner.MinStake,
			"max_stake":           miner.MaxStake,
			"last_health_check":   miner.LastHealthCheck,
			"rewards":             miner.Rewards,
			"fees":                miner.Fees,
			"active":              miner.Active,
			"longitude":           miner.Longitude,
			"latitude":            miner.Latitude,
		})

	return result.Error
}

func (edb *EventDb) addOrOverwriteMiner(miner Miner) error {

	exists, err := miner.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteMiner(miner)
	}

	err = edb.addMiner(miner)

	return err
}

func (mn *Miner) exists(edb *EventDb) (bool, error) {

	var miner Miner

	result := edb.Get().
		Model(&Miner{}).
		Where(&Miner{MinerID: mn.MinerID}).
		Take(&miner)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for miner %v, error %v",
			mn.MinerID, result.Error)
	}

	return true, nil
}

func (edb *EventDb) updateMiner(updates dbs.DbUpdates) error {

	var miner = Miner{MinerID: updates.Id}
	exists, err := miner.exists(edb)

	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("miner %v not in database cannot update",
			miner.MinerID)
	}

	result := edb.Store.Get().
		Model(&Miner{}).
		Where(&Miner{MinerID: miner.MinerID}).
		Updates(updates.Updates)

	return result.Error
}

func (edb *EventDb) deleteMiner(id string) error {

	result := edb.Store.Get().
		Where(&Miner{MinerID: id}).
		Delete(&Miner{})

	return result.Error
}
