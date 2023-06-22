package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

func emitUpdateBlobber(sn *StorageNode, bi *BlobberOfferStake, sp *stakePool, balances cstate.StateContextI) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}
	data := &event.Blobber{
		BaseURL:       sn.BaseURL,
		Latitude:      sn.Geolocation.Latitude,
		Longitude:     sn.Geolocation.Longitude,
		ReadPrice:     sn.Terms.ReadPrice,
		WritePrice:    sn.Terms.WritePrice,
		MinLockDemand: sn.Terms.MinLockDemand,

		Capacity:    sn.Capacity,
		Allocated:   bi.Allocated,
		SavedData:   sn.SavedData,
		IsAvailable: sn.IsAvailable,
		Provider: event.Provider{
			ID:              sn.ID,
			DelegateWallet:  sn.StakePoolSettings.DelegateWallet,
			NumDelegates:    sn.StakePoolSettings.MaxNumDelegates,
			ServiceCharge:   sn.StakePoolSettings.ServiceChargeRatio,
			LastHealthCheck: sn.LastHealthCheck,
			TotalStake:      staked,
		},
		OffersTotal: sp.TotalOffers,
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, sn.ID, data)
	return nil
}

func emitAddBlobber(sn *StorageNode, idx int32, bi *BlobberOfferStake, sp *stakePool, balances cstate.StateContextI) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}

	data := &event.Blobber{
		BaseURL:       sn.BaseURL,
		Latitude:      sn.Geolocation.Latitude,
		Longitude:     sn.Geolocation.Longitude,
		ReadPrice:     sn.Terms.ReadPrice,
		WritePrice:    sn.Terms.WritePrice,
		MinLockDemand: sn.Terms.MinLockDemand,

		Capacity:    sn.Capacity,
		Allocated:   bi.Allocated,
		SavedData:   sn.SavedData,
		IsAvailable: true,
		Provider: event.Provider{
			ID:              sn.ID,
			Index:           idx,
			DelegateWallet:  sn.StakePoolSettings.DelegateWallet,
			NumDelegates:    sn.StakePoolSettings.MaxNumDelegates,
			ServiceCharge:   sn.StakePoolSettings.ServiceChargeRatio,
			LastHealthCheck: sn.LastHealthCheck,
			TotalStake:      staked,
			Rewards: event.ProviderRewards{
				ProviderID:   sn.ID,
				Rewards:      sp.Reward,
				TotalRewards: sp.Reward,
			},
		},

		OffersTotal: bi.TotalOffers,

		CreationRound: balances.GetBlock().Round,
	}

	balances.EmitEvent(event.TypeStats, event.TagAddBlobber, sn.ID, data)
	return nil
}

// func emitUpdateBlobberAllocatedSavedHealth(sn *StorageNode, balances cstate.StateContextI) {
func emitUpdateBlobberAllocatedSavedHealth(id string, lhc common.Timestamp, alloced, savedData int64, balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberAllocatedSavedHealth, id, event.Blobber{
		Provider: event.Provider{
			ID:              id,
			LastHealthCheck: lhc,
		},
		Allocated: alloced,
		SavedData: savedData,
	})
}

func emitBlobberHealthCheck(sn *StorageNode, downtime uint64, balances cstate.StateContextI) {
	data := dbs.DbHealthCheck{
		ID:              sn.ID,
		LastHealthCheck: sn.LastHealthCheck,
		Downtime:        downtime,
	}

	balances.EmitEvent(event.TypeStats, event.TagBlobberHealthCheck, sn.ID, data)
}
