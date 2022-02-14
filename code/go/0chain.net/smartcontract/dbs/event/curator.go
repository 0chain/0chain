package event

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type Curator struct {
	gorm.Model
	CuratorID    string     `json:"curator_id" gorm:"uniqueIndex"`
	AllocationID string     `json:"allocation_id"` // Foreign Key
	Allocation   Allocation `json:"-" gorm:"references:AllocationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (edb *EventDb) overwriteCurator(c Curator) error {
	result := edb.Store.Get().
		Model(&Curator{}).
		Where(&Curator{CuratorID: c.CuratorID}).
		Updates(map[string]interface{}{
			"allocation_id": c.AllocationID,
			"curator_id":    c.CuratorID,
		})
	return result.Error
}

//GetCuratorsByAllocationID returns an array of curator ID
//from curators table matching the given allocation ID
func (edb *EventDb) GetCuratorsByAllocationID(allocationID string) ([]string, error) {
	var curators []Curator
	curatorIDs := make([]string, 0)
	result := edb.Store.Get().Model(&Curator{}).
		Where(&Curator{AllocationID: allocationID}).
		Find(&curators)

	if result.Error != nil {
		return curatorIDs, result.Error
	}

	for _, curator := range curators {
		curatorIDs = append(curatorIDs, curator.CuratorID)
	}

	return curatorIDs, nil
}

func (edb *EventDb) addOrOverwriteCurator(c Curator) error {
	exists, err := c.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteCurator(c)
	}

	result := edb.Store.Get().Create(&c)
	return result.Error
}

func (c *Curator) exists(edb *EventDb) (bool, error) {
	var curator Curator
	result := edb.Store.Get().Model(&Curator{}).Where(&Curator{CuratorID: c.CuratorID}).Take(&curator)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if result.Error != nil {
		return false, fmt.Errorf("failed to check Curator existence %v, error %v",
			curator, result.Error)
	}
	return true, nil
}

func (edb *EventDb) removeCurator(c Curator) error {
	res := edb.Store.Get().Where("curator_id = ?", c.CuratorID).Delete(Curator{})
	return res.Error
}
