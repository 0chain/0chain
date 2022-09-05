package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"

	"0chain.net/smartcontract/dbs/event"
)

func emitAddOrOverwriteBlobber(
	sn *StorageNode, sp *stakePool, balances cstate.StateContextI,
) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}
	data := &event.Blobber{
		BlobberID:        sn.ID,
		BaseURL:          sn.BaseURL,
		Latitude:         sn.Geolocation.Latitude,
		Longitude:        sn.Geolocation.Longitude,
		ReadPrice:        sn.Terms.ReadPrice,
		WritePrice:       sn.Terms.WritePrice,
		MinLockDemand:    sn.Terms.MinLockDemand,
		MaxOfferDuration: sn.Terms.MaxOfferDuration.Nanoseconds(),

		Capacity:        sn.Capacity,
		Allocated:       sn.Allocated,
		SavedData:       sn.SavedData,
		LastHealthCheck: int64(sn.LastHealthCheck),

		DelegateWallet: sn.StakePoolSettings.DelegateWallet,
		MinStake:       sn.StakePoolSettings.MinStake,
		MaxStake:       sn.StakePoolSettings.MaxStake,
		NumDelegates:   sn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  sn.StakePoolSettings.ServiceChargeRatio,

		OffersTotal:  sp.TotalOffers,
		UnstakeTotal: sp.TotalUnStake,
		Reward:       sp.Reward,
		TotalStake:   staked,

		Name:        sn.Information.Name,
		WebsiteUrl:  sn.Information.WebsiteUrl,
		Description: sn.Information.Description,
		LogoUrl:     sn.Information.LogoUrl,
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteBlobber, sn.ID, data)
	return nil
}

func emitAddBlobber(sn *StorageNode, sp *stakePool, balances cstate.StateContextI) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}

	data := &event.Blobber{
		BlobberID:        sn.ID,
		BaseURL:          sn.BaseURL,
		Latitude:         sn.Geolocation.Latitude,
		Longitude:        sn.Geolocation.Longitude,
		ReadPrice:        sn.Terms.ReadPrice,
		WritePrice:       sn.Terms.WritePrice,
		MinLockDemand:    sn.Terms.MinLockDemand,
		MaxOfferDuration: sn.Terms.MaxOfferDuration.Nanoseconds(),

		Capacity:        sn.Capacity,
		Allocated:       sn.Allocated,
		SavedData:       sn.SavedData,
		LastHealthCheck: int64(sn.LastHealthCheck),

		DelegateWallet: sn.StakePoolSettings.DelegateWallet,
		MinStake:       sn.StakePoolSettings.MinStake,
		MaxStake:       sn.StakePoolSettings.MaxStake,
		NumDelegates:   sn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  sn.StakePoolSettings.ServiceChargeRatio,

		OffersTotal:  sp.TotalOffers,
		UnstakeTotal: sp.TotalUnStake,
		Reward:       sp.Reward,
		TotalStake:   staked,

		Name:        sn.Information.Name,
		WebsiteUrl:  sn.Information.WebsiteUrl,
		Description: sn.Information.Description,
		LogoUrl:     sn.Information.LogoUrl,
	}

	balances.EmitEvent(event.TypeStats, event.TagAddBlobber, sn.ID, data)
	return nil
}

func emitUpdateBlobber(sn *StorageNode, balances cstate.StateContextI) error {
	data := &dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"base_url":           sn.BaseURL,
			"latitude":           sn.Geolocation.Latitude,
			"longitude":          sn.Geolocation.Longitude,
			"read_price":         int64(sn.Terms.ReadPrice),
			"write_price":        int64(sn.Terms.WritePrice),
			"min_lock_demand":    sn.Terms.MinLockDemand,
			"max_offer_duration": sn.Terms.MaxOfferDuration.Nanoseconds(),
			"capacity":           sn.Capacity,
			"allocated":          sn.Allocated,
			"last_health_check":  int64(sn.LastHealthCheck),
			"delegate_wallet":    sn.StakePoolSettings.DelegateWallet,
			"min_stake":          int64(sn.StakePoolSettings.MinStake),
			"max_stake":          int64(sn.StakePoolSettings.MaxStake),
			"num_delegates":      sn.StakePoolSettings.MaxNumDelegates,
			"service_charge":     sn.StakePoolSettings.ServiceChargeRatio,
			"saved_data":         sn.SavedData,
		},
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, sn.ID, data)
	return nil
}
