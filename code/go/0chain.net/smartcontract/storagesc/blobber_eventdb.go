package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/dto"
)

func emitUpdateBlobber(sn *dto.StorageDtoNode, sp *stakePool, balances cstate.StateContextI) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}
	data := &event.Blobber{
		BaseURL:   getString(sn.BaseURL),
		Capacity:  getInt64(sn.Capacity),
		Allocated: getInt64(sn.Allocated),
		SavedData: getInt64(sn.SavedData),
		Provider: event.Provider{
			ID:              sn.ID,
			DelegateWallet:  sp.Settings.DelegateWallet,
			NumDelegates:    sp.Settings.MaxNumDelegates,
			ServiceCharge:   sp.Settings.ServiceChargeRatio,
			LastHealthCheck: sn.LastHealthCheck,
			TotalStake:      staked,
		},
		OffersTotal: sp.TotalOffers,
	}
	if sn.Geolocation != nil {
		if sn.Geolocation.Latitude != nil {
			data.Latitude = *sn.Geolocation.Latitude
		}
		if sn.Geolocation.Longitude != nil {
			data.Longitude = *sn.Geolocation.Longitude
		}

	}

	if sn.Terms != nil {
		if sn.Terms.ReadPrice != nil {
			data.ReadPrice = *sn.Terms.ReadPrice
		}
		if sn.Terms.WritePrice != nil {
			data.WritePrice = *sn.Terms.WritePrice
		}
	}

	if sn.NotAvailable != nil {
		data.NotAvailable = *sn.NotAvailable
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, sn.ID, data)
	return nil
}

func getString(pointerValue *string) string {
	if pointerValue != nil {
		return *pointerValue
	}
	return ""
}

func getInt64(pointerValue *int64) int64 {
	if pointerValue != nil {
		return *pointerValue
	}
	return 0
}

func emitAddBlobber(sn *StorageNode, sp *stakePool, balances cstate.StateContextI) error {
	staked, err := sp.stake()
	if err != nil {
		return err
	}

	data := &event.Blobber{
		BaseURL:    sn.BaseURL,
		Latitude:   sn.Geolocation.Latitude,
		Longitude:  sn.Geolocation.Longitude,
		ReadPrice:  sn.Terms.ReadPrice,
		WritePrice: sn.Terms.WritePrice,

		Capacity:     sn.Capacity,
		Allocated:    sn.Allocated,
		SavedData:    sn.SavedData,
		NotAvailable: false,
		Provider: event.Provider{
			ID:              sn.ID,
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

		OffersTotal: sp.TotalOffers,

		CreationRound: balances.GetBlock().Round,
	}

	balances.EmitEvent(event.TypeStats, event.TagAddBlobber, sn.ID, data)
	return nil
}

func emitUpdateBlobberAllocatedSavedHealth(sn *StorageNode, balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobberAllocatedSavedHealth, sn.ID, event.Blobber{
		Provider: event.Provider{
			ID:              sn.ID,
			LastHealthCheck: sn.LastHealthCheck,
		},
		Allocated: sn.Allocated,
		SavedData: sn.SavedData,
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
