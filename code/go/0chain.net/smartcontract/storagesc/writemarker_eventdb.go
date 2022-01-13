package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"
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
	}
}

func emitAddOrOverwriteWriteMarker(wm *WriteMarker, balances cstate.StateContextI, t *transaction.Transaction) error {

	data, err := json.Marshal(writeMarkerToWriteMarkerTable(wm))
	if err != nil {
		return fmt.Errorf("failed to marshal writemarker: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteWriteMarker, t.Hash, string(data))

	return nil
}
