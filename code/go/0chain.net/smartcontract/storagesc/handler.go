package storagesc

import (
	"context"
	"encoding/json"
	"net/url"

	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values) (interface{}, error) {
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	allocationBytes, err := ssc.DB.GetNode(allocationObj.GetKey())
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(allocationBytes, allocationObj)
	return allocationObj, err
}

func (ssc *StorageSmartContract) LatestReadMarkerHandler(ctx context.Context, params url.Values) (interface{}, error) {
	clientID := params.Get("client")
	blobberID := params.Get("blobber")
	var commitRead ReadConnection
	commitRead.ReadMarker = &ReadMarker{BlobberID: blobberID, ClientID: clientID}

	commitReadBytes, err := ssc.DB.GetNode(commitRead.GetKey())
	if err != nil {
		return nil, err
	}
	if commitReadBytes == nil {
		return make(map[string]string), nil
	}
	err = commitRead.Decode(commitReadBytes)

	return commitRead.ReadMarker, err

}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values) (interface{}, error) {
	blobberID := params.Get("blobber")
	var blobberChallengeObj BlobberChallenge
	blobberChallengeObj.BlobberID = blobberID

	blobberChallengeBytes, err := ssc.DB.GetNode(blobberChallengeObj.GetKey())
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB. "+err.Error())
	}
	blobberChallengeObj.Decode(blobberChallengeBytes)

	// for k, v := range blobberChallengeObj.ChallengeMap {
	// 	if v.Response != nil {
	// 		delete(blobberChallengeObj.ChallengeMap, k)
	// 	}
	// }

	return &blobberChallengeObj, err
}
