package event

import (
	common2 "0chain.net/smartcontract/common"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type AllocationBlobberTerm struct {
	gorm.Model
	AllocationID    int64  `json:"alloc_id" gorm:"column:alloc_id; uniqueIndex:idx_alloc_blob,priority:1; not null"` // Foreign Key, priority: lowest first
	BlobberID       string `json:"blobber_id" gorm:"uniqueIndex:idx_alloc_blob,priority:2; not null"`                // Foreign Key
	ReadPrice       int64  `json:"read_price"`
	WritePrice      int64  `json:"write_price"`
	AllocBlobberIdx int64  `json:"alloc_blobber_idx"`

	AllocationIdHash string `json:"allocation_id" gorm:"-"` // Hash of AllocationID
}

// ByIndex implements sort.Interface based on the Age field.
type ByIndex []AllocationBlobberTerm

func (a ByIndex) Len() int           { return len(a) }
func (a ByIndex) Less(i, j int) bool { return a[i].AllocBlobberIdx < a[j].AllocBlobberIdx }
func (a ByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (edb *EventDb) GetAllocationBlobberTerm(allocationID string, blobberID string) (*AllocationBlobberTerm, error) {
	var term AllocationBlobberTerm

	err := edb.Store.Get().
		Joins("JOIN allocations ON allocation_blobber_terms.alloc_id = allocations.id").
		Model(&AllocationBlobberTerm{}).
		Select("allocations.allocation_id as allocation_id, allocation_blobber_terms.blobber_id as blobber_id, allocation_blobber_terms.read_price as read_price, allocation_blobber_terms.write_price as write_price").
		Where("allocations.allocation_id = ? AND allocation_blobber_terms.blobber_id = ?", allocationID, blobberID).
		Take(&term).Error

	if err == nil {
		term.AllocationIdHash = allocationID
	}

	return &term, err
}

func (edb *EventDb) GetAllocationBlobberTerms(allocationID string, limit common2.Pagination) ([]AllocationBlobberTerm, error) {
	var terms []AllocationBlobberTerm

	err := edb.Store.Get().
		Joins("JOIN allocations ON allocation_blobber_terms.alloc_id = allocations.id").
		Model(&AllocationBlobberTerm{}).
		Select("allocations.allocation_id as allocation_id, allocation_blobber_terms.blobber_id as blobber_id, allocation_blobber_terms.read_price as read_price, allocation_blobber_terms.write_price as write_price").
		Where("allocations.allocation_id = ?", allocationID).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order("alloc_blobber_idx").
		Find(&terms).Error

	if err == nil {
		for i := range terms {
			terms[i].AllocationIdHash = allocationID
		}
	}

	return terms, err
}

func deleteAllocationBlobberTerms(edb *EventDb, allocBlobbers map[string][]string) error {
	for allocationID, blobberIDs := range allocBlobbers {
		db := edb.Store.Get()

		var err error

		if len(blobberIDs) > 0 {
			err = db.Exec("DELETE FROM allocation_blobber_terms WHERE alloc_id = (SELECT id FROM allocations WHERE allocation_id=?) AND blobber_id IN (?)", allocationID, blobberIDs).Error
		} else {
			err = db.Exec("DELETE FROM allocation_blobber_terms WHERE alloc_id = (SELECT id FROM allocations WHERE allocation_id=?)", allocationID).Error
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (edb *EventDb) deleteAllocationBlobberTerms(terms []AllocationBlobberTerm) error {
	if len(terms) < 1 || terms[0].AllocationIdHash == "" {
		return nil
	}

	allocIDBlobberIDs := map[string][]string{}
	for _, term := range terms {
		if term.BlobberID == "" {
			continue
		}
		allocIDBlobberIDs[term.AllocationIdHash] = append(allocIDBlobberIDs[term.AllocationIdHash], term.BlobberID)
	}

	err := deleteAllocationBlobberTerms(edb, allocIDBlobberIDs)
	if err != nil {
		return err
	}
	return nil
}

func (edb *EventDb) updateAllocationBlobberTerms(terms []AllocationBlobberTerm) error {
	var (
		allocationIdList []int64
		blobberIdList    []string
		readPriceList    []int64
		writePriceList   []int64
	)

	for _, t := range terms {

		var allocationID int64
		err := edb.Store.Get().Model(&Allocation{}).Select("id").Where("allocation_id = ?", t.AllocationIdHash).Take(&allocationID).Error
		if err != nil {
			return err
		}

		allocationIdList = append(allocationIdList, allocationID)
		blobberIdList = append(blobberIdList, t.BlobberID)
		readPriceList = append(readPriceList, t.ReadPrice)
		writePriceList = append(writePriceList, t.WritePrice)
	}

	return CreateBuilder("allocation_blobber_terms", "alloc_id", allocationIdList).
		AddCompositeId("blobber_id", blobberIdList).
		AddUpdate("read_price", readPriceList).
		AddUpdate("write_price", writePriceList).
		Exec(edb).Error
}

func (edb *EventDb) addOrOverwriteAllocationBlobberTerms(terms []AllocationBlobberTerm) error {
	updateFields := []string{"read_price", "write_price", "max_offer_duration"}

	for i, t := range terms {
		var allocationID int64
		err := edb.Store.Get().Model(&Allocation{}).Select("id").Where("allocation_id = ?", t.AllocationIdHash).Take(&allocationID).Error
		if err != nil {
			return err
		}

		terms[i].AllocationID = allocationID
	}

	return edb.Store.Get().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "alloc_id"}, {Name: "blobber_id"}},
		DoUpdates: clause.AssignmentColumns(updateFields), // column needed to be updated
	}).Create(terms).Error
}

func (edb *EventDb) GetAllocationsByBlobberId(blobberId string, limit common2.Pagination) ([]Allocation, error) {
	var result []Allocation
	err := edb.Store.Get().Model(&Allocation{}).
		Joins("JOIN allocation_blobber_terms on allocation_blobber_terms.alloc_id = allocations.id").
		Where("blobber_id = ?", blobberId).
		Offset(limit.Offset).
		Limit(limit.Limit).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "allocations.created_at"},
			Desc:   limit.IsDescending,
		}).
		Debug().Find(&result).Error
	return result, err
}
