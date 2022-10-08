package event

import (
	"time"

	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/zbig"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type AllocationBlobberTerm struct {
	gorm.Model
	AllocationID            string        `json:"allocation_id" gorm:"uniqueIndex:idx_alloc_blob,priority:1; not null"` // Foreign Key, priority: lowest first
	BlobberID               string        `json:"blobber_id" gorm:"uniqueIndex:idx_alloc_blob,priority:2; not null"`    // Foreign Key
	ReadPrice               int64         `json:"read_price"`
	WritePrice              int64         `json:"write_price"`
	MinLockDemand           zbig.BigRat   `json:"min_lock_demand"`
	MaxOfferDuration        time.Duration `json:"max_offer_duration"`
	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
	Allocation              Allocation    `json:"-" gorm:"references:AllocationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Blobber                 Blobber       `json:"-" gorm:"references:BlobberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (edb *EventDb) GetAllocationBlobberTerm(allocationID string, blobberID string) (*AllocationBlobberTerm, error) {
	var term AllocationBlobberTerm
	return &term, edb.Store.Get().Model(&AllocationBlobberTerm{}).
		Where("allocation_id = ? and blobber_id = ?", allocationID, blobberID).
		Take(&term).Error
}

func (edb *EventDb) GetAllocationBlobberTerms(allocationID, blobberID string, limit common2.Pagination) ([]AllocationBlobberTerm, error) {
	var terms []AllocationBlobberTerm
	return terms, edb.Store.Get().Model(&AllocationBlobberTerm{}).
		Where(AllocationBlobberTerm{AllocationID: allocationID, BlobberID: blobberID}).
		Offset(limit.Offset).Limit(limit.Limit).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "id"},
		Desc:   limit.IsDescending,
	}).Find(&terms).Error
}

func deleteAllocationBlobberTerms(edb *EventDb, allocBlobbers map[string][]string) error {
	for allocationID, blobberIDs := range allocBlobbers {
		db := edb.Store.Get()
		if len(blobberIDs) > 0 {
			db = db.Where("allocation_id = ? and blobber_id in ?", allocationID, blobberIDs)
		} else {
			db = db.Where("allocation_id = ?", allocationID)
		}

		err := db.Delete(&AllocationBlobberTerm{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (edb *EventDb) deleteAllocationBlobberTerms(terms []AllocationBlobberTerm) error {
	if len(terms) < 1 || terms[0].AllocationID == "" {
		return nil
	}

	allocIDBlobberIDs := map[string][]string{}
	for _, term := range terms {
		if term.BlobberID == "" {
			continue
		}
		allocIDBlobberIDs[term.AllocationID] = append(allocIDBlobberIDs[term.AllocationID], term.BlobberID)
	}

	err := deleteAllocationBlobberTerms(edb, allocIDBlobberIDs)
	if err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) updateAllocationBlobberTerms(terms []AllocationBlobberTerm) error {
	return edb.addOrOverwriteAllocationBlobberTerms(terms)
}

func (edb *EventDb) addOrOverwriteAllocationBlobberTerms(terms []AllocationBlobberTerm) error {
	updateFields := []string{"read_price", "write_price", "min_lock_demand",
		"max_offer_duration", "challenge_completion_time"}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "allocation_id"}, {Name: "blobber_id"}},
		DoUpdates: clause.AssignmentColumns(updateFields), // column needed to be updated
	}).Create(terms).Error
}
