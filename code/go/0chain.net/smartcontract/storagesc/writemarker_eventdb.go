package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/dbs/event"
)

// TransactionID and BlockNumber is added at the time of emitting event
func writeMarkerToWriteMarkerTable(wm *WriteMarker) *event.WriteMarker {
	return &event.WriteMarker{
		ClientID:               wm.ClientID,
		BlobberID:              wm.BlobberID,
		AllocationID:           wm.AllocationID,
		AllocationRoot:         wm.AllocationRoot,
		PreviousAllocationRoot: wm.PreviousAllocationRoot,
		Size:                   wm.Size,
		Timestamp:              int64(wm.Timestamp),
		Signature:              wm.Signature,
		LookupHash:             wm.LookupHash,
		Name:                   wm.Name,
		ContentHash:            wm.ContentHash,
	}
}

func emitAddWriteMarker(wm *WriteMarker, balances cstate.StateContextI, t *transaction.Transaction) {
	balances.EmitEvent(event.TypeSmartContract, event.TagAddWriteMarker, t.Hash, writeMarkerToWriteMarkerTable(wm))
}
