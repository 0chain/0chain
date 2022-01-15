package event

import (
	"0chain.net/core/common"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type ReadMarker struct {
	gorm.Model
	ClientID      string  `json:"client_id"`
	BlobberID     string  `json:"blobber_id"`
	AllocationID  string  `json:"allocation_id"`
	TransactionID string  `json:"transaction_id"`
	OwnerID       string  `json:"owner_id"`
	Timestamp     int64   `json:"timestamp"`
	ReadCounter   int64   `json:"read_counter"`
	ReadSize      float64 `json:"read_size"`
	Signature     string  `json:"signature"`
	PayerID       string  `json:"payer_id"`
	AuthTicket    string  `json:"auth_ticket"`
	BlockNumber   int64   `json:"block_number"`
}

func (edb *EventDb) GetReadMarkersFromQuery(query *ReadMarker) (*[]ReadMarker, error) {

	if query == nil {
		return nil, common.NewError("get_read_markers", "empty query")
	}

	var rms []ReadMarker
	result := edb.Store.Get().
		Model(&ReadMarker{}).
		Where(query).
		Find(&rms)
	return &rms, result.Error
}

func (edb EventDb) CountReadMarkersFromQuery(query *ReadMarker) (count int64, err error) {

	if query == nil {
		err = common.NewError("count_read_markers", "empty query")
		return
	}

	result := edb.Get().
		Model(&ReadMarker{}).
		Where(query).
		Count(&count)

	if result.Error != nil {
		err = errors.New("error searching for read marker")
		return
	}

	err = nil
	return
}

func (edb *EventDb) overwriteReadMarker(rm ReadMarker) error {
	result := edb.Store.Get().
		Model(&ReadMarker{}).
		Where(&ReadMarker{TransactionID: rm.TransactionID}).
		Updates(&rm)
	return result.Error
}

func (edb *EventDb) addOrOverwriteReadMarker(rm ReadMarker) error {
	exists, err := rm.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteReadMarker(rm)
	}

	result := edb.Store.Get().Create(&rm)
	return result.Error
}

func (rm *ReadMarker) exists(edb *EventDb) (bool, error) {
	var readMarker ReadMarker
	result := edb.Get().
		Model(&ReadMarker{}).
		Where(&ReadMarker{TransactionID: rm.TransactionID}).
		Take(&readMarker)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil
	} else if result.Error != nil {
		return false, fmt.Errorf("error searching for read marker txn: %v, error %v",
			rm.TransactionID, result.Error)
	}
	return true, nil
}
