package storagesc

import (
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/provider"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

func providerType(p provider.ProviderI) spenum.Provider {
	switch p.(type) {
	case *ValidationNode:
		return spenum.Validator
	case *StorageNode:
		return spenum.Blobber
	default:
		return spenum.Unknown
	}
}

func emitUpdateProvider(p provider.ProviderI, sp *stakePool, balances cstate.StateContextI) error {
	switch pType := p.(type) {
	case *ValidationNode:
		validator := p.(*ValidationNode)
		return validator.EmitUpdate(&sp.StakePool, balances)
	case *StorageNode:
		blobber := p.(*StorageNode)
		return blobber.EmitUpdate(sp, balances)
	default:
		return fmt.Errorf("unreconised provider type %v", pType)
	}
}

func (sn *StorageNode) EmitAdd(balances cstate.StateContextI) {
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
		IsShutDown:      sn.IsShutDown(),
		IsKilled:        sn.IsKilled(),

		DelegateWallet: sn.StakePoolSettings.DelegateWallet,
		MinStake:       sn.StakePoolSettings.MinStake,
		MaxStake:       sn.StakePoolSettings.MaxStake,
		NumDelegates:   sn.StakePoolSettings.MaxNumDelegates,
		ServiceCharge:  sn.StakePoolSettings.ServiceChargeRatio,
	}
	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteBlobber, sn.ID, data)
}

func (sn *StorageNode) EmitUpdate(sp *stakePool, balances cstate.StateContextI) error {
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
			"is_killed":          sn.IsKilled(),
			"is_shut_down":       sn.IsShutDown(),
			"delegate_wallet":    sn.StakePoolSettings.DelegateWallet,
			"min_stake":          int64(sn.StakePoolSettings.MinStake),
			"max_stake":          int64(sn.StakePoolSettings.MaxStake),
			"num_delegates":      sn.StakePoolSettings.MaxNumDelegates,
			"service_charge":     sn.StakePoolSettings.ServiceChargeRatio,
			"saved_data":         sn.SavedData,
		},
	}
	if sp != nil {
		stake, err := sp.stake()
		if err != nil {
			return err
		}
		data.Updates["offers_total"] = sp.TotalOffers
		data.Updates["unstake_total"] = sp.TotalUnStake
		data.Updates["stake"] = stake
		data.Updates["reward"] = sp.Reward
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, sn.ID, data)
	return nil
}
