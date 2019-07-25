package storagesc

import (
	"context"
	// "encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) GetAllocationsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	clientID := params.Get("client")
	allocations, err := ssc.getAllocationsList(clientID, balances)
	if err != nil {
		return nil, err
	}
	result := make([]*StorageAllocation, 0)
	for _, allocationID := range allocations.List {
		allocationObj := &StorageAllocation{}
		allocationObj.ID = allocationID

		allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
		if err != nil {
			continue
		}
		allocationObj.Decode(allocationBytes.Encode())
		result = append(result, allocationObj)
	}
	return result, nil
}

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	allocationBytes, err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID))
	if err != nil {
		return nil, err
	}
	allocationObj.Decode(allocationBytes.Encode())
	return allocationObj, err
}

func (ssc *StorageSmartContract) LatestReadMarkerHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	clientID := params.Get("client")
	blobberID := params.Get("blobber")
	commitRead := &ReadConnection{}
	commitRead.ReadMarker = &ReadMarker{BlobberID: blobberID, ClientID: clientID}

	commitReadBytes, err := balances.GetTrieNode(commitRead.GetKey(ssc.ID))
	if err != nil {
		return nil, err
	}
	if commitReadBytes == nil {
		return make(map[string]string), nil
	}
	commitRead.Decode(commitReadBytes.Encode())

	return commitRead.ReadMarker, err

}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	blobberID := params.Get("blobber")
	blobberChallengeObj := &BlobberChallenge{}
	blobberChallengeObj.BlobberID = blobberID
	blobberChallengeObj.Challenges = make([]*StorageChallenge, 0)

	blobberChallengeBytes, err := balances.GetTrieNode(blobberChallengeObj.GetKey(ssc.ID))
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB. "+err.Error())
	}
	blobberChallengeObj.Decode(blobberChallengeBytes.Encode())

	// for k, v := range blobberChallengeObj.ChallengeMap {
	// 	if v.Response != nil {
	// 		delete(blobberChallengeObj.ChallengeMap, k)
	// 	}
	// }

	return &blobberChallengeObj, err
}
