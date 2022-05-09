package storagesc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"

	"0chain.net/smartcontract/dbs/event"
)

func emitAddOrOverwriteBlobber(
	sn *StorageNode, sp *stakePool, balances cstate.StateContextI,
) error {
	data, err := json.Marshal(&event.Blobber{
		BlobberID:               sn.ID,
		BaseURL:                 sn.BaseURL,
		Latitude:                sn.Geolocation.Latitude,
		Longitude:               sn.Geolocation.Longitude,
		ReadPrice:               int64(sn.Terms.ReadPrice),
		WritePrice:              int64(sn.Terms.WritePrice),
		MinLockDemand:           sn.Terms.MinLockDemand,
		MaxOfferDuration:        sn.Terms.MaxOfferDuration.String(),
		ChallengeCompletionTime: int64(sn.Terms.ChallengeCompletionTime),

		Capacity:        sn.Capacity,
		Used:            sn.Used,
		SavedData:       sn.SavedData,
		LastHealthCheck: int64(sn.LastHealthCheck),

		DelegateWallet: sn.StakePoolSettings.DelegateWallet,
		MinStake:       int64(sn.StakePoolSettings.MinStake),
		MaxStake:       int64(sn.StakePoolSettings.MaxStake),
		NumDelegates:   sn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  sn.StakePoolSettings.ServiceCharge,

		OffersTotal:  int64(sp.TotalOffers),
		UnstakeTotal: int64(sp.TotalUnStake),
		Reward:       int64(sp.Reward),
		TotalStake:   int64(sp.stake()),

		Name:        sn.Information.Name,
		WebsiteUrl:  sn.Information.WebsiteUrl,
		Description: sn.Information.Description,
		LogoUrl:     sn.Information.LogoUrl,
	})
	if err != nil {
		return fmt.Errorf("marshalling blobber: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteBlobber, sn.ID, string(data))
	return nil
}

func emitUpdateBlobber(sn *StorageNode, balances cstate.StateContextI) error {
	data, err := json.Marshal(&dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"base_url":                  sn.BaseURL,
			"latitude":                  sn.Geolocation.Latitude,
			"longitude":                 sn.Geolocation.Longitude,
			"read_price":                int64(sn.Terms.ReadPrice),
			"write_price":               int64(sn.Terms.WritePrice),
			"min_lock_demand":           sn.Terms.MinLockDemand,
			"max_offer_duration":        sn.Terms.MaxOfferDuration.String(),
			"challenge_completion_time": int64(sn.Terms.ChallengeCompletionTime),
			"capacity":                  sn.Capacity,
			"used":                      sn.Used,
			"last_health_check":         int64(sn.LastHealthCheck),
			"delegate_wallet":           sn.StakePoolSettings.DelegateWallet,
			"min_stake":                 int64(sn.StakePoolSettings.MinStake),
			"max_stake":                 int64(sn.StakePoolSettings.MaxStake),
			"num_delegates":             sn.StakePoolSettings.MaxNumDelegates,
			"service_charge":            sn.StakePoolSettings.ServiceCharge,
			"saved_data":                sn.SavedData,
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, sn.ID, string(data))
	return nil
}
