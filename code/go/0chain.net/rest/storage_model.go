package rest

import (
	"time"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/datastore"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/storagesc"
)

// swagger:model readMarkersCount
type readMarkersCount struct {
	ReadMarkersCount int64 `json:"read_markers_count"`
}

// swagger:model storageStakePool
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

// swagger:model storageNodeResponse
type storageNodesResponse struct {
	Nodes []storageNodeResponse
}

// StorageNode represents Blobber configurations.
// swagger:model storageNodeResponse
type storageNodeResponse struct {
	storagesc.StorageNode
	TotalStake int64 `json:"total_stake"`
}

func blobberTableToStorageNode(blobber event.Blobber) (storageNodeResponse, error) {
	maxOfferDuration, err := time.ParseDuration(blobber.MaxOfferDuration)
	if err != nil {
		return storageNodeResponse{}, err
	}
	challengeCompletionTime, err := time.ParseDuration(blobber.ChallengeCompletionTime)
	if err != nil {
		return storageNodeResponse{}, err
	}
	return storageNodeResponse{
		StorageNode: storagesc.StorageNode{
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
		},
		TotalStake: blobber.TotalStake,
	}, nil
}

// swagger:model userPoolStat
type userPoolStat struct {
	Pools map[datastore.Key][]*storagesc.DelegatePoolStat `json:"pools"`
}

// swagger:model stakePoolStat
type stakePoolStat struct {
	ID      string        `json:"pool_id"` // pool ID
	Balance state.Balance `json:"balance"` // total balance
	Unstake state.Balance `json:"unstake"` // total unstake amount

	Free       int64         `json:"free"`        // free staked space
	Capacity   int64         `json:"capacity"`    // blobber bid
	WritePrice state.Balance `json:"write_price"` // its write price

	OffersTotal  state.Balance `json:"offers_total"` //
	UnstakeTotal state.Balance `json:"unstake_total"`
	// delegate pools
	Delegate []storagesc.DelegatePoolStat `json:"delegate"`
	Penalty  state.Balance                `json:"penalty"` // total for all
	// rewards
	Rewards state.Balance `json:"rewards"`

	// Settings of the stake pool
	Settings stakepool.StakePoolSettings `json:"settings"`
}

func spStats(
	blobber event.Blobber,
	delegatePools []event.DelegatePool,
) *stakePoolStat {
	stat := new(stakePoolStat)
	stat.ID = blobber.BlobberID
	stat.UnstakeTotal = state.Balance(blobber.UnstakeTotal)
	stat.Capacity = blobber.Capacity
	stat.WritePrice = state.Balance(blobber.WritePrice)
	stat.OffersTotal = state.Balance(blobber.OffersTotal)
	stat.Delegate = make([]storagesc.DelegatePoolStat, 0, len(delegatePools))
	stat.Settings = stakepool.StakePoolSettings{
		DelegateWallet:  blobber.DelegateWallet,
		MinStake:        state.Balance(blobber.MinStake),
		MaxStake:        state.Balance(blobber.MaxStake),
		MaxNumDelegates: blobber.NumDelegates,
		ServiceCharge:   blobber.ServiceCharge,
	}
	stat.Rewards = state.Balance(blobber.Reward)
	for _, dp := range delegatePools {
		dpStats := storagesc.DelegatePoolStat{
			ID:           dp.PoolID,
			Balance:      state.Balance(dp.Balance),
			DelegateID:   dp.DelegateID,
			Rewards:      state.Balance(dp.Reward),
			Status:       spenum.PoolStatus(dp.Status).String(),
			TotalReward:  state.Balance(dp.TotalReward),
			TotalPenalty: state.Balance(dp.TotalPenalty),
			RoundCreated: dp.RoundCreated,
		}
		stat.Balance += dpStats.Balance
		stat.Delegate = append(stat.Delegate, dpStats)
	}
	return stat
}
