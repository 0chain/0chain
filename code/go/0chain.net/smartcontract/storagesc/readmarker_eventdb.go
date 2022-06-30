package storagesc

import (
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
		ReadCounter:  rm.ReadCounter,
		ReadSize:     rm.ReadSize,
		Signature:    rm.Signature,
	}

	if rm.AuthTicket != nil {
		readMarker.AuthTicket = encryption.Hash(rm.AuthTicket.getHashData())
	}

	return readMarker
}

func emitAddOrOverwriteReadMarker(rm *ReadMarker, balances cstate.StateContextI, t *transaction.Transaction) error {

	balances.EmitEvent(event.TypeStats, event.TagAddReadMarker, t.Hash, readMarkerToReadMarkerTable(rm))

	return nil
}
