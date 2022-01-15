package event

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
)

type ReadMarker struct {
	gorm.Model
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	AllocationID  string `json:"allocation_id"`
	TransactionID string `json:"transaction_id"`
	OwnerID       string `json:"owner_id"`
	Timestamp     int64  `json:"timestamp"`
	ReadCounter   int64  `json:"read_counter"`
	ReadSize      int64  `json:"read_size"`
	Signature     string `json:"signature"`
	PayerID       string `json:"payer_id"`
	AuthTicket    string `json:"auth_ticket"`
	BlockNumber   int64  `json:"block_number"`
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
