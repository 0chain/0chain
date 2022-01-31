package event

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type DelegatePool struct {
	gorm.Model

	PoolId string `json:"pool_id"`

	// foreign keys todo: when user(ID) created, enable it
	DelegateId string `json:"delegate_id"`
	BlobberId  string `json:"blobber_id"`

	Reward       int64 `json:"reward"`
	Penalty      int64 `json:"penalty"`
	Status       int   `json:"status"`
	RoundCreated int64 `json:"round_created"`
}

func (edb *EventDb) GetDelegatesByBlobber(blobberId string) (*[]DelegatePool, error) {
	var dps []DelegatePool
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{BlobberId: blobberId}).
		Find(&dps)
	return &dps, result.Error
}
func (edb *EventDb) GetDelegatesByUser(delegateId string) (*[]DelegatePool, error) {
	var dps []DelegatePool
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{DelegateId: delegateId}).
		Find(&dps)
	return &dps, result.Error
}

func (edb *EventDb) overwriteDelegatePool(dp DelegatePool) error {
	result := edb.Store.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{PoolId: dp.PoolId}).
		Updates(&dp)
	return result.Error
}

func (edb *EventDb) addOrOverwriteDelegatePool(dp DelegatePool) error {
	exists, err := dp.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteDelegatePool(dp)
	}

	result := edb.Store.Get().Create(&dp)
	return result.Error
}

func (wm *DelegatePool) exists(edb *EventDb) (bool, error) {
	var dp DelegatePool

	result := edb.Get().
		Model(&DelegatePool{}).
		Where(&DelegatePool{PoolId: dp.PoolId}).
		Take(&dp)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for write marker txn: %v, error %v",
			dp.PoolId, result.Error)
	}
	return true, nil
}
