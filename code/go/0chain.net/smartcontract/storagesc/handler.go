package storagesc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/datastore"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

func (ssc *StorageSmartContract) GetAllocationsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	clientID := params.Get("client")
	allocations, err := ssc.getAllocationsList(clientID, balances)
	if err != nil {
		return nil, common.NewErrInternal("can't get allocation list", err.Error())
	}
	result := make([]*StorageAllocation, 0)
	for _, allocationID := range allocations.List {
		allocationObj := &StorageAllocation{}
		allocationObj.ID = allocationID

		err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID), allocationObj)
		switch err {
		case nil:
			result = append(result, allocationObj)
		case util.ErrValueNotPresent:
			continue
		default:
			msg := fmt.Sprintf("can't decode allocation with id '%s'", allocationID)
			return nil, common.NewErrInternal(msg, err.Error())
		}
	}
	return result, nil
}

func (ssc *StorageSmartContract) GetAllocationMinLockHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	var err error
	var creationDate = common.Timestamp(time.Now().Unix())

	allocData := params.Get("allocation_data")
	var request newAllocationRequest
	if err = request.decode([]byte(allocData)); err != nil {
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

	var sa = request.storageAllocation()

	blobberNodes, bSize, err := ssc.selectBlobbers(
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

type userPoolStat struct {
	Pools map[datastore.Key][]*DelegatePoolStat `json:"pools"`
}

func (ssc *StorageSmartContract) getUserStakePoolStatHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	clientID := datastore.Key(params.Get("client_id"))

	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	pools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Blobber))
	if err != nil {
		return nil, errors.New("blobber not found in event database")
	}

	var ups = new(userPoolStat)
	ups.Pools = make(map[datastore.Key][]*DelegatePoolStat)
	for _, pool := range pools {
		var dps = DelegatePoolStat{
			ID:           pool.PoolID,
			Balance:      state.Balance(pool.Balance),
			DelegateID:   pool.DelegateID,
			Rewards:      state.Balance(pool.Reward),
			TotalPenalty: state.Balance(pool.TotalPenalty),
			TotalReward:  state.Balance(pool.TotalReward),
			Status:       spenum.PoolStatus(pool.Status).String(),
			RoundCreated: pool.RoundCreated,
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dps)
	}

	return ups, nil
}

func (ssc *StorageSmartContract) GetBlockByHashHandler(_ context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	hash := params.Get("block_hash")
	if len(hash) == 0 {
		return nil, fmt.Errorf("cannot find valid block hash: %v", hash)
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	block, err := balances.GetEventDB().GetBlocksByHash(hash)
	return &block, err
}

func (ssc *StorageSmartContract) GetBlocksHandler(_ context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	block, err := balances.GetEventDB().GetBlocks()
	return &block, err
}

func (ssc *StorageSmartContract) GetTotalData(_ context.Context, balances cstate.StateContextI) (int64, error) {
	if ssc != nil {
		storageNodes, err := ssc.getBlobbersList(balances)
		if err != nil {
			return 0, fmt.Errorf("error from getBlobbersList in GetTotalData: %v", err)
		}

		var totalSavedData int64
		for _, sn := range storageNodes.Nodes {
			totalSavedData += sn.SavedData
		}

		return totalSavedData, nil
	}

	return 0, fmt.Errorf("storageSmartContract is nil")
}
