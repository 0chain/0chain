package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/currency"
)

// TransactionID and BlockNumber is added at the time of emitting event
func writeMarkerToWriteMarkerTable(wm *WriteMarker, movedTokens currency.Coin, txnHash string) *event.WriteMarker {
	wmb := wm.mustBase()
	evm := &event.WriteMarker{
		ClientID:               wmb.ClientID,
		BlobberID:              wmb.BlobberID,
		AllocationID:           wmb.AllocationID,
		AllocationRoot:         wmb.AllocationRoot,
		PreviousAllocationRoot: wmb.PreviousAllocationRoot,
		FileMetaRoot:           wmb.FileMetaRoot,
		Size:                   wmb.Size,
		Timestamp:              int64(wmb.Timestamp),
		Signature:              wmb.Signature,
		MovedTokens:            movedTokens,
		TransactionID:          txnHash,
	}
	if wm.GetVersion() == "v2" {
		wm2 := wm.Entity().(*writeMarkerV2)
		evm.ChainHash = wm2.ChainHash
		evm.ChainSize = wm2.ChainSize
	}
	return evm
}

func emitAddWriteMarker(t *transaction.Transaction, wm *WriteMarker, alloc *StorageAllocation, movedTokens currency.Coin, changeSize int64,
	balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagAddWriteMarker,
		t.Hash, writeMarkerToWriteMarkerTable(wm, movedTokens, t.Hash))

	emitUpdateAllocationStatEvent(alloc, balances)
	emitUpdateBlobberWriteStatEvent(wm, changeSize, balances)
}
