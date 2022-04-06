package rest

import (
	"time"

	"0chain.net/core/datastore"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/storagesc"
)

//go:generate msgp -io=false -tests=false -unexported=true -v

// swagger:model storageNode
type storageNode storagesc.StorageNode

// swagger:model storageNode
type storageStakePool struct {
	stakepool.StakePool
	// TotalOffers represents tokens required by currently
	// open offers of the blobber. It's allocation_id -> {lock, expire}
	TotalOffers state.Balance `json:"total_offers"`
	// Total amount to be un staked
	TotalUnStake state.Balance `json:"total_un_stake"`
}

// stake pool key for the storage SC and  blobber
func storageStakePoolKey(blobberID string) datastore.Key {
	return datastore.Key(storagesc.ADDRESS + ":stakepool:" + blobberID)
}

func (sp storageStakePool) get(
	blobberID datastore.Key, srh StorageRestHandler,
) error {
	return srh.GetTrieNode(storageStakePoolKey(blobberID), &sp)
}

func blobberTableToStorageNode(blobber event.Blobber) (storageNode, error) {
	maxOfferDuration, err := time.ParseDuration(blobber.MaxOfferDuration)
	if err != nil {
		return storageNode{}, err
	}
	challengeCompletionTime, err := time.ParseDuration(blobber.ChallengeCompletionTime)
	if err != nil {
		return storageNode{}, err
	}
	return storageNode{
		ID:      blobber.BlobberID,
		BaseURL: blobber.BaseURL,
		Geolocation: storagesc.StorageNodeGeolocation{
			Latitude:  blobber.Latitude,
			Longitude: blobber.Longitude,
		},
		Terms: storagesc.Terms{
			ReadPrice:               state.Balance(blobber.ReadPrice),
			WritePrice:              state.Balance(blobber.WritePrice),
			MinLockDemand:           blobber.MinLockDemand,
			MaxOfferDuration:        maxOfferDuration,
			ChallengeCompletionTime: challengeCompletionTime,
		},
		Capacity:        blobber.Capacity,
		Used:            blobber.Used,
		LastHealthCheck: common.Timestamp(blobber.LastHealthCheck),
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  blobber.DelegateWallet,
			MinStake:        state.Balance(blobber.MinStake),
			MaxStake:        state.Balance(blobber.MaxStake),
			MaxNumDelegates: blobber.NumDelegates,
			ServiceCharge:   blobber.ServiceCharge,
		},
		Information: storagesc.Info{
			Name:        blobber.Name,
			WebsiteUrl:  blobber.WebsiteUrl,
			LogoUrl:     blobber.LogoUrl,
			Description: blobber.Description,
		},
	}, nil
}
