package event

import (
	"encoding/json"

	"0chain.net/chaincore/currency"
	"gorm.io/gorm"
)

type AllocationPool struct {
	gorm.Model
	AllocationID string `json:"allocation_id"`
	ClientID     string `json:"client_id"`

	Balance currency.Coin `json:"balance"`
	Expires int64         `json:"expires"`
}

func (ap *AllocationPool) Encode() []byte {
	var b, err = json.Marshal(ap)
	if err != nil {
		panic(err)
	}
	return b
}

func (edb *EventDb) GetAllocationPools(allocationId, clientId string) ([]AllocationPool, error) {
	var pools []AllocationPool
	result := edb.Store.Get().
		Model(&AllocationPool{}).
		Where(&AllocationPool{
			AllocationID: allocationId,
			ClientID:     clientId,
		}).
		Find(&pools)

	return pools, result.Error
}

func (edb *EventDb) addOrUpdatePools(aps []AllocationPool) error {
	for _, ap := range aps {
		if err := edb.addOrUpdatePool(ap); err != nil {
			return err
		}
	}
	return nil
}

func (edb *EventDb) deleteAllocationPool(ap AllocationPool) error {
	return edb.Store.Get().
		Model(&AllocationPool{}).
		Where("allocation_id = ? and client_id = ?", ap.AllocationID, ap.ClientID).
		Delete(&AllocationPool{}).
		Error
}

func (edb *EventDb) updateAllocationPool(ap AllocationPool) error {
	return edb.Store.Get().
		Model(&AllocationPool{}).
		Where("allocation_id = ? and client_id = ?", ap.AllocationID, ap.ClientID).
		Updates(map[string]interface{}{
			"balance": ap.Balance,
			"expires": ap.Expires,
		}).Error
}

func (edb *EventDb) addOrUpdatePool(ap AllocationPool) error {
	exists, err := ap.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.updateAllocationPool(ap)
	}
	return edb.Store.Get().Create(&ap).Error
}

func (ap *AllocationPool) exists(edb *EventDb) (bool, error) {
	var exists bool
	err := edb.Store.Get().Model(&AllocationPool{}).
		Select("count(*) > 0").
		Where("allocation_id = ? and client_id = ?", ap.AllocationID, ap.ClientID).
		Find(&exists).
		Error

	return exists, err
}
