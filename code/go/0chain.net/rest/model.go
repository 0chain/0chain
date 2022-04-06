package rest

import (
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/storagesc"
)

// swagger:model intMap
type intMap map[string]int64

// swagger:model StorageNode
type StorageNode storagesc.StorageNode

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
