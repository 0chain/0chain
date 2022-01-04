package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"
)

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

func emitAddOrOverwriteWriteMarker(wm *WriteMarker, balances cstate.StateContextI) error {

	_, err := json.Marshal(writeMarkerToWriteMarkerTable(wm))
	if err != nil {
		return fmt.Errorf("marshalling writemarker: %v", err)
	}

	return nil
}
