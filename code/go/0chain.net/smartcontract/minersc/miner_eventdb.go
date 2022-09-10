package minersc

import (
	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

func minerTableToMinerNode(edbMiner event.Miner) MinerNode {
	var status = node.NodeStatusInactive
	if edbMiner.Active {
		status = node.NodeStatusActive
	}
	msn := SimpleNode{
		ID:          edbMiner.MinerID,
		N2NHost:     edbMiner.N2NHost,
		Host:        edbMiner.Host,
		Port:        edbMiner.Port,
		Path:        edbMiner.Path,
		PublicKey:   edbMiner.PublicKey,
		ShortName:   edbMiner.ShortName,
		BuildTag:    edbMiner.BuildTag,
		TotalStaked: edbMiner.TotalStaked,
		Delete:      edbMiner.Delete,
		Geolocation: SimpleNodeGeolocation{
			Latitude:  edbMiner.Latitude,
			Longitude: edbMiner.Longitude,
		},
		NodeType:        NodeTypeMiner,
		LastHealthCheck: edbMiner.LastHealthCheck,
		Status:          status,
	}

	return MinerNode{
		SimpleNode: &msn,
		StakePool: &stakepool.StakePool{
			Reward: edbMiner.Rewards.Rewards,
			Settings: stakepool.Settings{
				DelegateWallet:     edbMiner.DelegateWallet,
				ServiceChargeRatio: edbMiner.ServiceCharge,
				MaxNumDelegates:    edbMiner.NumberOfDelegates,
				MinStake:           edbMiner.MinStake,
				MaxStake:           edbMiner.MaxStake,
			},
		},
	}

}

func minerNodeToMinerTable(mn *MinerNode) event.Miner {
	return event.Miner{
		MinerID:           mn.ID,
		N2NHost:           mn.N2NHost,
		Host:              mn.Host,
		Port:              mn.Port,
		Path:              mn.Path,
		PublicKey:         mn.PublicKey,
		ShortName:         mn.ShortName,
		BuildTag:          mn.BuildTag,
		TotalStaked:       mn.TotalStaked,
		Delete:            mn.Delete,
		DelegateWallet:    mn.Settings.DelegateWallet,
		ServiceCharge:     mn.Settings.ServiceChargeRatio,
		NumberOfDelegates: mn.Settings.MaxNumDelegates,
		MinStake:          mn.Settings.MinStake,
		MaxStake:          mn.Settings.MaxStake,
		LastHealthCheck:   mn.LastHealthCheck,
		Rewards: event.ProviderRewards{
			ProviderID:   mn.ID,
			Rewards:      mn.Reward,
			TotalRewards: mn.Reward,
		},
		Active:    mn.Status == node.NodeStatusActive,
		Longitude: mn.Geolocation.Longitude,
		Latitude:  mn.Geolocation.Latitude,
	}
}

//func emitAddMiner(mn *MinerNode, balances cstate.StateContextI) error {
//
//	logging.Logger.Info("emitting add miner event")
//
//	balances.EmitEvent(event.TypeStats, event.TagAddMiner, mn.ID, minerNodeToMinerTable(mn))
//
//	return nil
//}

func emitAddOrOverwriteMiner(mn *MinerNode, balances cstate.StateContextI) error {

	logging.Logger.Info("emitting add or overwrite miner event")

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteMiner, mn.ID, minerNodeToMinerTable(mn))

	return nil
}

func emitUpdateMiner(mn *MinerNode, balances cstate.StateContextI, updateStatus bool) error {

	logging.Logger.Info("emitting update miner event")

	dbUpdates := dbs.DbUpdates{
		Id: mn.ID,
		Updates: map[string]interface{}{
			"n2n_host":            mn.N2NHost,
			"host":                mn.Host,
			"port":                mn.Port,
			"path":                mn.Path,
			"public_key":          mn.PublicKey,
			"short_name":          mn.ShortName,
			"build_tag":           mn.BuildTag,
			"total_staked":        mn.TotalStaked,
			"delete":              mn.Delete,
			"delegate_wallet":     mn.Settings.DelegateWallet,
			"service_charge":      mn.Settings.ServiceChargeRatio,
			"number_of_delegates": mn.Settings.MaxNumDelegates,
			"min_stake":           mn.Settings.MinStake,
			"max_stake":           mn.Settings.MaxStake,
			"last_health_check":   mn.LastHealthCheck,
			"longitude":           mn.SimpleNode.Geolocation.Longitude,
			"latitude":            mn.SimpleNode.Geolocation.Latitude,
			"rewards":             mn.Reward,
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
