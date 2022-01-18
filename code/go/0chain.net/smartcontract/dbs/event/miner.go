package event

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type Miner struct {
	gorm.Model
	MinerID           string `gorm:"uniqueIndex"`
	N2NHost           string `gorm:"column:n2n_host"`
	Host              string
	Port              int
	Path              string
	PublicKey         string
	ShortName         string
	BuildTag          string
	TotalStaked       state.Balance
	Delete            bool
	DelegateWallet    string
	ServiceCharge     float64
	NumberOfDelegates int
	MinStake          state.Balance
	MaxStake          state.Balance
	LastHealthCheck   common.Timestamp
	Rewards           state.Balance
	Fees              state.Balance
	Active            bool
	Longitude         int64
	Latitude          int64
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

func (edb *EventDb) GetMinersFromQuery(query interface{}) ([]Miner, error) {

	var miners []Miner

	result := edb.Store.Get().
		Model(&Miner{}).
		Where(query).
		Find(&miners)

	return miners, result.Error
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
