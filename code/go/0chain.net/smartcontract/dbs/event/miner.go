package event

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"fmt"
	"gorm.io/gorm"
)

type Miner struct {
	gorm.Model
	MinerID           string `gorm:"uniqueIndex"`
	N2NHost           string
	Host              string
	Port              int
	Path              string
	PublicKey         string
	ShortName         string
	BuildTag          string
	TotalStaked       int64
	Delete            bool
	DelegateWallet    string
	ServiceCharge     float64
	NumberOfDelegates int
	MinStake          state.Balance
	MaxStake          state.Balance
	LastHealthCheck   common.Timestamp
	Rewards           state.Balance
	Fees              state.Balance
	TotalStake        state.Balance
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
