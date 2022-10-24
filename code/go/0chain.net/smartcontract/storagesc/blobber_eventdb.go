package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

func emitAddOrOverwriteBlobber(sn *StorageNode, sp *stakePool, balances cstate.StateContextI) error {
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
		TotalStake:   staked,
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, sn.ID, data)
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
		Rewards: event.ProviderRewards{
			ProviderID:   sn.ID,
			Rewards:      sp.Reward,
			TotalRewards: sp.Reward,
		},
		TotalStake: staked,
	}

	balances.EmitEvent(event.TypeStats, event.TagAddBlobber, sn.ID, data)
	return nil
}

func emitUpdateBlobber(sn *StorageNode, balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberAllocatedHealth, sn.ID, event.Blobber{
		BlobberID:       sn.ID,
		Allocated:       sn.Allocated,
		LastHealthCheck: int64(sn.LastHealthCheck),
	})
}
