package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/dbs/event"
	"gorm.io/gorm"
)

func readMarkerToReadMarkerTable(rm *ReadMarker, txnHash string) *event.ReadMarker {

	readMarker := &event.ReadMarker{
		Model:         gorm.Model{},
		ClientID:      rm.ClientID,
		BlobberID:     rm.BlobberID,
		AllocationID:  rm.AllocationID,
		OwnerID:       rm.OwnerID,
		Timestamp:     int64(rm.Timestamp),
		ReadCounter:   rm.ReadCounter,
		ReadSize:      rm.ReadSize,
		Signature:     rm.Signature,
		TransactionID: txnHash,
	}

	return readMarker
}

func emitAddOrOverwriteReadMarker(rm *ReadMarker, balances cstate.StateContextI, t *transaction.Transaction) error {

	balances.EmitEvent(event.TypeStats, event.TagAddReadMarker, t.Hash, readMarkerToReadMarkerTable(rm, t.Hash))
	emitUpdateBlobberReadStatEvent(rm, balances)
	return nil
}
