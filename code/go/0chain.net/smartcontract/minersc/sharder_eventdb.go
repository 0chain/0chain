package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
)

func sharderTableToSharderNode(edbSharder event.Sharder, delegates []event.DelegatePool) MinerNodeResponse {
	var status = node.NodeStatusInactive
	if edbSharder.Active {
		status = node.NodeStatusActive
	}
	msn := SimpleNodeResponse{
		SimpleNode: SimpleNode{
			ID:          edbSharder.ID,
			N2NHost:     edbSharder.N2NHost,
			Host:        edbSharder.Host,
			Port:        edbSharder.Port,
			Path:        edbSharder.Path,
			PublicKey:   edbSharder.PublicKey,
			ShortName:   edbSharder.ShortName,
			BuildTag:    edbSharder.BuildTag,
			TotalStaked: edbSharder.TotalStake,
			Delete:      edbSharder.Delete,

			LastHealthCheck: edbSharder.LastHealthCheck,
			Geolocation: SimpleNodeGeolocation{
				Latitude:  edbSharder.Latitude,
				Longitude: edbSharder.Longitude,
			},
			NodeType: NodeTypeSharder,
			Status:   status,
		},
		RoundLastUpdated: edbSharder.RoundLastUpdated,
	}

	sn := MinerNodeResponse{
		SimpleNodeResponse: &msn,
		StakePoolResponse: &StakePoolResponse{
			Reward: edbSharder.Rewards.Rewards,
			Settings: stakepool.Settings{
				DelegateWallet:     edbSharder.DelegateWallet,
				ServiceChargeRatio: edbSharder.ServiceCharge,
				MaxNumDelegates:    edbSharder.Provider.NumDelegates,
				MinStake:           edbSharder.MinStake,
				MaxStake:           edbSharder.MaxStake,
			},
		},
	}
	if len(delegates) == 0 {
		return sn
	}
	sn.StakePoolResponse.Pools = make(map[string]*DelegatePoolResponse)
	for _, delegate := range delegates {
		sn.StakePoolResponse.Pools[delegate.PoolID] = &DelegatePoolResponse{
			DelegatePool: stakepool.DelegatePool{
				Balance:      delegate.Balance,
				Reward:       delegate.Reward,
				Status:       spenum.PoolStatus(delegate.Status),
				RoundCreated: delegate.RoundCreated,
				DelegateID:   delegate.DelegateID,
			},
			RoundLastUpdated: delegate.RoundLastUpdated,
		}
	}
	return sn

}

func sharderNodeToSharderTable(sn *MinerNode) event.Sharder {
	return event.Sharder{
		N2NHost:   sn.N2NHost,
		Host:      sn.Host,
		Port:      sn.Port,
		Path:      sn.Path,
		PublicKey: sn.PublicKey,
		ShortName: sn.ShortName,
		BuildTag:  sn.BuildTag,
		Delete:    sn.Delete,
		Provider: event.Provider{
			ID:             sn.ID,
			TotalStake:     sn.TotalStaked,
			DelegateWallet: sn.Settings.DelegateWallet,
			ServiceCharge:  sn.Settings.ServiceChargeRatio,
			NumDelegates:   sn.Settings.MaxNumDelegates,
			MinStake:       sn.Settings.MinStake,
			MaxStake:       sn.Settings.MaxStake,
			Rewards: event.ProviderRewards{
				ProviderID:   sn.ID,
				Rewards:      sn.Reward,
				TotalRewards: sn.Reward,
			},
		},
		LastHealthCheck: sn.LastHealthCheck,

		Active:    sn.Status == node.NodeStatusActive,
		Longitude: sn.Geolocation.Longitude,
		Latitude:  sn.Geolocation.Latitude,
	}
}

func emitAddSharder(sn *MinerNode, balances cstate.StateContextI) {
	balances.EmitEvent(event.TypeStats, event.TagAddSharder, sn.ID, sharderNodeToSharderTable(sn))
}

func emitUpdateSharder(sn *MinerNode, balances cstate.StateContextI, updateStatus bool) error {
	dbUpdates := dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"n2n_host":          sn.N2NHost,
			"host":              sn.Host,
			"port":              sn.Port,
			"path":              sn.Path,
			"public_key":        sn.PublicKey,
			"short_name":        sn.ShortName,
			"build_tag":         sn.BuildTag,
			"total_stake":       sn.TotalStaked,
			"delete":            sn.Delete,
			"delegate_wallet":   sn.Settings.DelegateWallet,
			"service_charge":    sn.Settings.ServiceChargeRatio,
			"num_delegates":     sn.Settings.MaxNumDelegates,
			"min_stake":         sn.Settings.MinStake,
			"max_stake":         sn.Settings.MaxStake,
			"last_health_check": sn.LastHealthCheck,
			"longitude":         sn.SimpleNode.Geolocation.Longitude,
			"latitude":          sn.SimpleNode.Geolocation.Latitude,
		},
	}

	if updateStatus {
		dbUpdates.Updates["active"] = sn.Status == node.NodeStatusActive
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateSharder, sn.ID, dbUpdates)
	logging.Logger.Warn("emit sharder - update sharder")
	return nil
}

func emitDeleteSharder(id string, balances cstate.StateContextI) error {
	balances.EmitEvent(event.TypeStats, event.TagDeleteSharder, id, id)
	return nil
}
