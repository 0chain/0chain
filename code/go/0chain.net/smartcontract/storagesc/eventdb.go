package storagesc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/core/common"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs"

	"0chain.net/smartcontract/dbs/event"
)

func blobberTableToStorageNode(blobber event.Blobber) (StorageNode, error) {
	maxOfferDuration, err := time.ParseDuration(blobber.MaxOfferDuration)
	if err != nil {
		return StorageNode{}, err
	}
	challengeCompletionTime, err := time.ParseDuration(blobber.ChallengeCompletionTime)
	if err != nil {
		return StorageNode{}, err
	}
	return StorageNode{
		ID:      blobber.BlobberID,
		BaseURL: blobber.BaseURL,
		Geolocation: StorageNodeGeolocation{
			Latitude:  blobber.Latitude,
			Longitude: blobber.Longitude,
		},
		Terms: Terms{
			ReadPrice:               state.Balance(blobber.ReadPrice),
			WritePrice:              state.Balance(blobber.WritePrice),
			MinLockDemand:           blobber.MinLockDemand,
			MaxOfferDuration:        maxOfferDuration,
			ChallengeCompletionTime: challengeCompletionTime,
		},
		Capacity:        blobber.Capacity,
		Used:            blobber.Used,
		LastHealthCheck: common.Timestamp(blobber.LastHealthCheck),
		StakePoolSettings: stakePoolSettings{
			DelegateWallet: blobber.DelegateWallet,
			MinStake:       state.Balance(blobber.MinStake),
			MaxStake:       state.Balance(blobber.MaxStake),
			NumDelegates:   blobber.NumDelegates,
			ServiceCharge:  blobber.ServiceCharge,
		},
	}, nil
}

func emitAddBlobber(sn *StorageNode, balances cstate.StateContextI) error {
	data, err := json.Marshal(&event.Blobber{
		BlobberID:               sn.ID,
		BaseURL:                 sn.BaseURL,
		Latitude:                sn.Geolocation.Latitude,
		Longitude:               sn.Geolocation.Longitude,
		ReadPrice:               int64(sn.Terms.ReadPrice),
		WritePrice:              int64(sn.Terms.WritePrice),
		MinLockDemand:           sn.Terms.MinLockDemand,
		MaxOfferDuration:        sn.Terms.MaxOfferDuration.String(),
		ChallengeCompletionTime: sn.Terms.ChallengeCompletionTime.String(),
		Capacity:                sn.Capacity,
		Used:                    sn.Used,
		LastHealthCheck:         int64(sn.LastHealthCheck),
		DelegateWallet:          sn.StakePoolSettings.DelegateWallet,
		MinStake:                int64(sn.StakePoolSettings.MaxStake),
		MaxStake:                int64(sn.StakePoolSettings.MaxStake),
		NumDelegates:            sn.StakePoolSettings.NumDelegates,
		ServiceCharge:           sn.StakePoolSettings.ServiceCharge,
	})
	if err != nil {
		return fmt.Errorf("marshalling blobber: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagAddBlobber, sn.ID, string(data))
	return nil
}

func emitUpdateBlobber(sn *StorageNode, balances cstate.StateContextI) error {
	data, err := json.Marshal(&dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"url":                       sn.BaseURL,
			"latitude":                  sn.Geolocation.Latitude,
			"longitude":                 sn.Geolocation.Longitude,
			"read_price":                int64(sn.Terms.ReadPrice),
			"write_price":               int64(sn.Terms.WritePrice),
			"min_lock_demand":           sn.Terms.MinLockDemand,
			"max_offer_duration":        sn.Terms.MaxOfferDuration.String(),
			"challenge_completion_time": sn.Terms.ChallengeCompletionTime.String(),
			"capacity":                  sn.Capacity,
			"used":                      sn.Used,
			"last_health_check":         int64(sn.LastHealthCheck),
			"delegate_wallet":           sn.StakePoolSettings.DelegateWallet,
			"min_stake":                 int64(sn.StakePoolSettings.MaxStake),
			"max_stake":                 int64(sn.StakePoolSettings.MaxStake),
			"num_delegates":             sn.StakePoolSettings.NumDelegates,
			"service_charge":            sn.StakePoolSettings.ServiceCharge,
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, sn.ID, string(data))
	return nil
}
