package event

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type Sharder struct {
	gorm.Model
	SharderID         string `gorm:"uniqueIndex"`
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

func (edb *EventDb) GetSharder(id string) (*Sharder, error) {

	var sharder Sharder

	result := edb.Store.Get().
		Model(&Sharder{}).
		Where(&Sharder{SharderID: id}).
		First(&sharder)

	if result.Error != nil {
		return nil, fmt.Errorf("error retrieving sharder %v, error %v",
			id, result.Error)
	}

	return &sharder, nil

}

func (edb *EventDb) GetShardersFromQuery(query *Sharder) ([]Sharder, error) {

	var sharders []Sharder

	result := edb.Store.Get().
		Model(&Sharder{}).
		Where(query).
		Find(&sharders)

	return sharders, result.Error
}

func (edb *EventDb) GetSharders() ([]Sharder, error) {

	var sharders []Sharder

	result := edb.Store.Get().
		Model(&Sharder{}).
		Find(&sharders)

	return sharders, result.Error
}

func (edb *EventDb) CountShardersFromQuery(query interface{}) (int64, error) {

	var count int64

	result := edb.Store.Get().
		Model(&Sharder{}).
		Where(query).
		Count(&count)

	return count, result.Error
}

func (edb *EventDb) GetShardersTotalStake() (int64, error) {
	var count int64

	err := edb.Store.Get().Table("sharders").Select("sum(total_staked)").Row().Scan(&count)
	return count, err
}

func (edb *EventDb) addSharder(sharder Sharder) error {

	result := edb.Store.Get().Create(&sharder)

	return result.Error
}

func (edb *EventDb) overwriteSharder(sharder Sharder) error {

	result := edb.Store.Get().
		Model(&Sharder{}).
		Where(&Sharder{SharderID: sharder.SharderID}).
		Updates(map[string]interface{}{
			"n2n_host":            sharder.N2NHost,
			"host":                sharder.Host,
			"port":                sharder.Port,
			"path":                sharder.Path,
			"public_key":          sharder.PublicKey,
			"short_name":          sharder.ShortName,
			"build_tag":           sharder.BuildTag,
			"total_staked":        sharder.TotalStaked,
			"delete":              sharder.Delete,
			"delegate_wallet":     sharder.DelegateWallet,
			"service_charge":      sharder.ServiceCharge,
			"number_of_delegates": sharder.NumberOfDelegates,
			"min_stake":           sharder.MinStake,
			"max_stake":           sharder.MaxStake,
			"last_health_check":   sharder.LastHealthCheck,
			"rewards":             sharder.Rewards,
			"fees":                sharder.Fees,
			"active":              sharder.Active,
			"longitude":           sharder.Longitude,
			"latitude":            sharder.Latitude,
		})

	return result.Error
}

func (edb *EventDb) addOrOverwriteSharder(sharder Sharder) error {

	exists, err := sharder.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteSharder(sharder)
	}

	err = edb.addSharder(sharder)

	return err
}

func (sh *Sharder) exists(edb *EventDb) (bool, error) {

	var sharder Sharder

	result := edb.Get().
		Model(&Sharder{}).
		Where(&Sharder{SharderID: sh.SharderID}).
		Take(&sharder)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for sharder %v, error %v",
			sh.SharderID, result.Error)
	}

	return true, nil
}

func (edb *EventDb) updateSharder(updates dbs.DbUpdates) error {

	var sharder = Sharder{SharderID: updates.Id}
	exists, err := sharder.exists(edb)

	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("sharder %v not in database cannot update",
			sharder.SharderID)
	}

	result := edb.Store.Get().
		Model(&Sharder{}).
		Where(&Sharder{SharderID: sharder.SharderID}).
		Updates(updates.Updates)

	return result.Error
}

func (edb *EventDb) deleteSharder(id string) error {

	result := edb.Store.Get().
		Where(&Sharder{SharderID: id}).
		Delete(&Sharder{})

	return result.Error
}
