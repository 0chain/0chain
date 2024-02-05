package storagesc

import (
	"fmt"

	"0chain.net/smartcontract/partitions"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	state "0chain.net/chaincore/chain/state"
	partitions_v_1 "0chain.net/smartcontract/partitions_v_1"
	partitions_v_2 "0chain.net/smartcontract/partitions_v_2"
)

//go:generate msgp -io=false -tests=false -v

//------------------------------------------------------------------------------

// BlobberAllocationNode represents the allocation that belongs to a blobber,
// will be saved in blobber allocations partitions.
type BlobberAllocationNode struct {
	ID string `json:"id"` // allocation id
}

func (z *BlobberAllocationNode) GetID() string {
	return z.ID
}

func partitionsBlobberAllocations(blobberID string, balances state.StateContextI) (res partitions.Partitions, err error) {
	actErr := state.WithActivation(balances, "apollo", func() error {
		res, err = partitions_v_1.CreateIfNotExists(balances, getBlobberAllocationsKey(blobberID), blobberAllocationPartitionSize)
		return err
	}, func() error {
		res, err = partitions_v_2.CreateIfNotExists(balances, getBlobberAllocationsKey(blobberID), blobberAllocationPartitionSize)
		return err
	})
	if actErr != nil {
		return nil, actErr
	}
	return
}
func partitionsBlobberAllocations_v_1(blobberID string, balances state.StateContextI) (partitions.Partitions, error) {
	return partitions_v_1.CreateIfNotExists(balances, getBlobberAllocationsKey(blobberID), blobberAllocationPartitionSize)
}
func partitionsBlobberAllocations_v_2(blobberID string, balances state.StateContextI) (partitions.Partitions, error) {
	return partitions_v_2.CreateIfNotExists(balances, getBlobberAllocationsKey(blobberID), blobberAllocationPartitionSize)
}

func partitionsBlobberAllocationsAdd(balances state.StateContextI, blobberID, allocID string) error {
	return state.WithActivation(balances, "apollo", func() error {
		return partitionsBlobberAllocationsAdd_v_1(balances, blobberID, allocID)
	}, func() error {
		return partitionsBlobberAllocationsAdd_v_2(balances, blobberID, allocID)

	})
}
func partitionsBlobberAllocationsAdd_v_1(state state.StateContextI, blobberID, allocID string) error {
	blobAllocsParts, err := partitionsBlobberAllocations_v_1(blobberID, state)
	if err != nil {
		return fmt.Errorf("error fetching blobber challenge allocation partition, %v", err)
	}

	err = blobAllocsParts.Add(state, &BlobberAllocationNode{ID: allocID})
	if err != nil && !partitions.ErrItemExist(err) {
		return err
	} else if partitions.ErrItemExist(err) {
		return nil
	}

	if err := blobAllocsParts.Save(state); err != nil {
		return fmt.Errorf("could not update blobber allocations partitions: %v", err)
	}

	return nil
}
func partitionsBlobberAllocationsAdd_v_2(state state.StateContextI, blobberID, allocID string) error {
	blobAllocsParts, err := partitionsBlobberAllocations_v_2(blobberID, state)
	if err != nil {
		return fmt.Errorf("error fetching blobber challenge allocation partition, %v", err)
	}

	err = blobAllocsParts.Add(state, &BlobberAllocationNode{ID: allocID})
	if err != nil && !partitions_v_2.ErrItemExist(err) {
		return err
	} else if partitions_v_2.ErrItemExist(err) {
		return nil
	}

	if err := blobAllocsParts.Save(state); err != nil {
		return fmt.Errorf("could not update blobber allocations partitions: %v", err)
	}

	return nil
}

func removeAllocationFromBlobberPartitions(st state.StateContextI, blobberID, allocID string) error {
	return state.WithActivation(st, "apollo", func() error {
		return removeAllocationFromBlobberPartitions_v_1(st, blobberID, allocID)
	}, func() error {
		return removeAllocationFromBlobberPartitions_v_2(st, blobberID, allocID)
	})
}

// removeAllocationFromBlobberPartitions removes the allocation from blobber
func removeAllocationFromBlobberPartitions_v_1(state state.StateContextI, blobberID, allocID string) error {
	blobAllocsParts, err := partitionsBlobberAllocations_v_1(blobberID, state)
	if err != nil {
		return fmt.Errorf("could not get blobber allocations partition: %v", err)
	}

	err = blobAllocsParts.Remove(state, allocID)

	logging.Logger.Info("removeAllocationFromBlobberPartitions", zap.Any("blobberID", blobberID), zap.Any("allocID", allocID), zap.Any("err", err))

	if err == nil {
		if err := blobAllocsParts.Save(state); err != nil {
			logging.Logger.Info("could not update blobber allocation partitions",
				zap.Error(err),
				zap.String("blobber", blobberID),
				zap.String("allocation", allocID))
			return fmt.Errorf("could not update blobber allocation partitions: %v", err)
		}

		allocNum, err := blobAllocsParts.Size(state)
		if err != nil {
			return fmt.Errorf("could not get challenge partition size: %v", err)
		}

		if allocNum > 0 {
			return nil
		}

		// remove blobber from challenge ready partition when there's no allocation bind to it
		err = partitionsChallengeReadyBlobbersRemove_v_1(state, blobberID)
		if err != nil && !partitions.ErrItemNotFound(err) {
			// it could be empty if we finalize the allocation before committing any read or write
			return fmt.Errorf("failed to remove blobber from challenge ready partitions: %v", err)
		}

		return nil
	} else {
		if partitions.ErrItemNotFound(err) {
			logging.Logger.Error("allocation is not in partition",
				zap.Error(err),
				zap.String("blobber", blobberID),
				zap.String("allocation", allocID))
		} else {
			logging.Logger.Error("error removing allocation from blobber",
				zap.Error(err),
				zap.String("blobber", blobberID),
				zap.String("allocation", allocID),
				zap.String("error", err.Error()))
			return fmt.Errorf("could not remove allocation from blobber: %v", err)
		}
	}

	return nil
}

// removeAllocationFromBlobberPartitions removes the allocation from blobber
func removeAllocationFromBlobberPartitions_v_2(state state.StateContextI, blobberID, allocID string) error {
	blobAllocsParts, err := partitionsBlobberAllocations_v_2(blobberID, state)
	if err != nil {
		return fmt.Errorf("could not get blobber allocations partition: %v", err)
	}

	err = blobAllocsParts.Remove(state, allocID)

	logging.Logger.Info("removeAllocationFromBlobberPartitions", zap.Any("blobberID", blobberID), zap.Any("allocID", allocID), zap.Any("err", err))

	if err == nil {
		if err := blobAllocsParts.Save(state); err != nil {
			logging.Logger.Info("could not update blobber allocation partitions",
				zap.Error(err),
				zap.String("blobber", blobberID),
				zap.String("allocation", allocID))
			return fmt.Errorf("could not update blobber allocation partitions: %v", err)
		}

		allocNum, err := blobAllocsParts.Size(state)
		if err != nil {
			return fmt.Errorf("could not get challenge partition size: %v", err)
		}

		if allocNum > 0 {
			return nil
		}

		// remove blobber from challenge ready partition when there's no allocation bind to it
		err = partitionsChallengeReadyBlobbersRemove_v_2(state, blobberID)
		if err != nil && !partitions.ErrItemNotFound(err) {
			// it could be empty if we finalize the allocation before committing any read or write
			return fmt.Errorf("failed to remove blobber from challenge ready partitions: %v", err)
		}

		return nil
	} else {
		if partitions.ErrItemNotFound(err) {
			logging.Logger.Error("allocation is not in partition",
				zap.Error(err),
				zap.String("blobber", blobberID),
				zap.String("allocation", allocID))
		} else {
			logging.Logger.Error("error removing allocation from blobber",
				zap.Error(err),
				zap.String("blobber", blobberID),
				zap.String("allocation", allocID),
				zap.String("error", err.Error()))
			return fmt.Errorf("could not remove allocation from blobber: %v", err)
		}
	}

	return nil
}
