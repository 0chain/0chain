package storagesc

import (
	"context"
	"net/url"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) GetAllocationMinLockHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	var err error
	var creationDate = common.Timestamp(time.Now().Unix())

	allocData := params.Get("allocation_data")
	var request NewAllocationRequest
	if err = request.Decode([]byte(allocData)); err != nil {
		return "", common.NewErrInternal("can't decode allocation request", err.Error())
	}

	var allBlobbersList *StorageNodes
	allBlobbersList, err = ssc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewErrInternal("can't get blobbers list", err.Error())
	}
	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewErrInternal("can't get blobbers list",
			"no blobbers found")
	}

	var sa = request.StorageAllocation()

	blobberNodes, bSize, err := ssc.SelectBlobbers(
		creationDate, *allBlobbersList, sa, int64(creationDate), balances)
	if err != nil {
		return "", common.NewErrInternal("selecting blobbers", err.Error())
	}

	var gbSize = sizeInGB(bSize)
	var minLockDemand state.Balance
	for _, b := range blobberNodes {
		minLockDemand += b.Terms.minLockDemand(gbSize,
			sa.restDurationInTimeUnits(creationDate))
	}

	var response = map[string]interface{}{
		"min_lock_demand": minLockDemand,
	}

	return response, nil
}
