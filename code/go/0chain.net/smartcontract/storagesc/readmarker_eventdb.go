package storagesc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs/event"
	"gorm.io/gorm"
)

func readMarkerToReadMarkerTable(rm *ReadMarker) *event.ReadMarker {

	readMarker := &event.ReadMarker{
		Model:        gorm.Model{},
		ClientID:     rm.ClientID,
		BlobberID:    rm.BlobberID,
		AllocationID: rm.AllocationID,
		OwnerID:      rm.OwnerID,
		Timestamp:    int64(rm.Timestamp),
		ReadSize:     rm.ReadSize,
		ReadSizeInGB: rm.ReadSizeInGB,
		Signature:    rm.Signature,
		PayerID:      rm.PayerID,
	}

	if rm.AuthTicket != nil {
		readMarker.AuthTicket = encryption.Hash(rm.AuthTicket.getHashData())
	}

	return readMarker
}

func emitAddOrOverwriteReadMarker(rm *ReadMarker, balances cstate.StateContextI, t *transaction.Transaction) error {

	data, err := json.Marshal(readMarkerToReadMarkerTable(rm))
	if err != nil {
		return fmt.Errorf("failed to marshal readmarker: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteReadMarker, t.Hash, string(data))

	return nil
}
