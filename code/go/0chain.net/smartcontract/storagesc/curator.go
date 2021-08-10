package storagesc

import (
	chainstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"encoding/json"
)

type curatorInput struct {
	CuratorId    string `json:"curator_id"`
	AllocationId string `json:"allocation_id"`
}

func (aci *curatorInput) decode(input []byte) error {
	return json.Unmarshal(input, aci)
}

func (sc *StorageSmartContract) removeCurator(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (err error) {
	var rci curatorInput
	if err = rci.decode(input); err != nil {
		return common.NewError("remove_curator_failed",
			"error unmarshalling input: "+err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(rci.AllocationId, balances)
	if err != nil {
		return common.NewError("remove_curator_failed alloc_cancel_failed", err.Error())
	}

	if alloc.Owner != txn.ClientID {
		return common.NewError("remove_curator_failed",
			"only owner can remove a curator")
	}

	var found = false
	for i, curator := range alloc.Curators {
		if curator == rci.CuratorId {
			// we don't care about order
			alloc.Curators[i] = alloc.Curators[len(alloc.Curators)-1]
			alloc.Curators = alloc.Curators[:len(alloc.Curators)-1]
			found = true
		}
	}
	if !found {
		return common.NewError("remove_curator_failed",
			"cannot find curator: "+rci.CuratorId)
	}

	alloc.Curators = append(alloc.Curators, rci.CuratorId)

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return common.NewError("remove_curator_failed",
			"cannot save allocation"+err.Error())
	}

	return nil
}

func (sc *StorageSmartContract) addCurator(
	txn *transaction.Transaction,
	input []byte,
	balances chainstate.StateContextI,
) (err error) {
	var aci curatorInput
	if err = aci.decode(input); err != nil {
		return common.NewError("add_curator_failed",
			"error unmarshalling input: "+err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(aci.AllocationId, balances)
	if err != nil {
		return common.NewError("alloc_cancel_failed", err.Error())
	}

	if alloc.Owner != txn.ClientID {
		return common.NewError("add_curator_failed",
			"only owner can add a curator")
	}

	if alloc.isCurator(aci.CuratorId) {
		return common.NewError("add_curator_failed",
			"already a curator: "+aci.CuratorId)
	}

	alloc.Curators = append(alloc.Curators, aci.CuratorId)

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return common.NewError("add_curator_failed",
			"cannot save allocation"+err.Error())
	}

	return nil
}

func (sa StorageAllocation) isCurator(id string) bool {
	for _, curator := range sa.Curators {
		if curator == id {
			return true
		}
	}
	return false
}
