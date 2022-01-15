package event

import (
	"gorm.io/gorm"
)

type Curator struct {
	gorm.Model

	// Foreign Key
	AllocationID string `json:"allocation_id"`

	CuratorID string `json:"curator_id"`
}

func (edb *EventDb) addCurator(c Curator) error {
	result := edb.Store.Get().Create(&c)
	return result.Error
}

func (edb *EventDb) removeCurator(c Curator) error {
	res := edb.Store.Get().Where("curator_id = ?", c.CuratorID).Delete(Curator{})
	return res.Error
}
