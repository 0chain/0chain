package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
)

type SimpleNodeResponse struct {
	SimpleNode
	RoundLastUpdated int64 `json:"round_last_updated"`
}

type DelegatePoolResponse struct {
	stakepool.DelegatePool
	RoundLastUpdated int64 `json:"round_last_updated"`
}

type StakePoolResponse struct {
	Pools    map[string]*DelegatePoolResponse `json:"pools"`
	Reward   currency.Coin                    `json:"rewards"`
	Settings stakepool.Settings               `json:"settings"`
	Minter   cstate.ApprovedMinter            `json:"minter"`
}

type MinerNodeResponse struct {
	*SimpleNodeResponse `json:"simple_miner"`
	*StakePoolResponse  `json:"stake_pool"`
}

func minerTableToMinerNode(edbMiner event.Miner, delegates []event.DelegatePool) MinerNodeResponse {
	var status = node.NodeStatusInactive
	if edbMiner.Active {
		status = node.NodeStatusActive
	}
	msn := SimpleNodeResponse{
		SimpleNode: SimpleNode{
			ID:          edbMiner.ID,
			N2NHost:     edbMiner.N2NHost,
			Host:        edbMiner.Host,
			Port:        edbMiner.Port,
			Path:        edbMiner.Path,
			PublicKey:   edbMiner.PublicKey,
			ShortName:   edbMiner.ShortName,
			BuildTag:    edbMiner.BuildTag,
			TotalStaked: edbMiner.Provider.TotalStake,
			Delete:      edbMiner.Delete,
			Geolocation: SimpleNodeGeolocation{
				Latitude:  edbMiner.Latitude,
				Longitude: edbMiner.Longitude,
			},
			NodeType:        NodeTypeMiner,
			LastHealthCheck: edbMiner.LastHealthCheck,
			Status:          status,
		},
		RoundLastUpdated: edbMiner.RoundLastUpdated,
	}

	mn := MinerNodeResponse{
		SimpleNodeResponse: &msn,
		StakePoolResponse: &StakePoolResponse{
			Reward: edbMiner.Rewards.Rewards,
			Settings: stakepool.Settings{
				DelegateWallet:     edbMiner.DelegateWallet,
				ServiceChargeRatio: edbMiner.ServiceCharge,
				MaxNumDelegates:    edbMiner.Provider.NumDelegates,
				MinStake:           edbMiner.MinStake,
				MaxStake:           edbMiner.MaxStake,
			},
		},
	}
	if len(delegates) == 0 {
		return mn
	}
	mn.StakePoolResponse.Pools = make(map[string]*DelegatePoolResponse)
	for _, delegate := range delegates {
		mn.StakePoolResponse.Pools[delegate.PoolID] = &DelegatePoolResponse{
			DelegatePool: stakepool.DelegatePool{
				Balance:      delegate.Balance,
				Reward:       delegate.Reward,
				Status:       spenum.PoolStatus(delegate.Status),
				RoundCreated: delegate.RoundCreated,
				DelegateID:   delegate.DelegateID,
			},
			RoundLastUpdated: delegate.RoundPoolLastUpdated,
		}
	}
	return mn
}

func minerNodeToMinerTable(mn *MinerNode) event.Miner {
	return event.Miner{
		N2NHost:   mn.N2NHost,
		Host:      mn.Host,
		Port:      mn.Port,
		Path:      mn.Path,
		PublicKey: mn.PublicKey,
		ShortName: mn.ShortName,
		BuildTag:  mn.BuildTag,
		Delete:    mn.Delete,
		Provider: event.Provider{
			ID:             mn.ID,
			TotalStake:     mn.TotalStaked,
			DelegateWallet: mn.Settings.DelegateWallet,
			ServiceCharge:  mn.Settings.ServiceChargeRatio,
			NumDelegates:   mn.Settings.MaxNumDelegates,
			MinStake:       mn.Settings.MinStake,
			MaxStake:       mn.Settings.MaxStake,
			Rewards: event.ProviderRewards{
				ProviderID:   mn.ID,
				Rewards:      mn.Reward,
				TotalRewards: mn.Reward,
			},
		},
		LastHealthCheck: mn.LastHealthCheck,

		Active:    mn.Status == node.NodeStatusActive,
		Longitude: mn.Geolocation.Longitude,
		Latitude:  mn.Geolocation.Latitude,
	}
}

func emitAddMiner(mn *MinerNode, balances cstate.StateContextI) {
	logging.Logger.Info("emitting add or overwrite miner event")
	balances.EmitEvent(event.TypeStats, event.TagAddMiner, mn.ID, minerNodeToMinerTable(mn))
}

func emitUpdateMiner(mn *MinerNode, balances cstate.StateContextI, updateStatus bool) error {

	logging.Logger.Info("emitting update miner event")

	dbUpdates := dbs.DbUpdates{
		Id: mn.ID,
		Updates: map[string]interface{}{
			"n2n_host":          mn.N2NHost,
			"host":              mn.Host,
			"port":              mn.Port,
			"path":              mn.Path,
			"public_key":        mn.PublicKey,
			"short_name":        mn.ShortName,
			"build_tag":         mn.BuildTag,
			"total_stake":       mn.TotalStaked,
			"delete":            mn.Delete,
			"delegate_wallet":   mn.Settings.DelegateWallet,
			"service_charge":    mn.Settings.ServiceChargeRatio,
			"num_delegates":     mn.Settings.MaxNumDelegates,
			"min_stake":         mn.Settings.MinStake,
			"max_stake":         mn.Settings.MaxStake,
			"last_health_check": mn.LastHealthCheck,
			"longitude":         mn.SimpleNode.Geolocation.Longitude,
			"latitude":          mn.SimpleNode.Geolocation.Latitude,
		},
	}

	if updateStatus {
		dbUpdates.Updates["active"] = mn.Status == node.NodeStatusActive
	}

	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, mn.ID, dbUpdates)
	return nil
}

func emitDeleteMiner(id string, balances cstate.StateContextI) error {

	logging.Logger.Info("emitting delete miner event")

	balances.EmitEvent(event.TypeStats, event.TagDeleteMiner, id, id)
	return nil
}
