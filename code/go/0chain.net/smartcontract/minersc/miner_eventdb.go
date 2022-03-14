package minersc

import (
	"encoding/json"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
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
		ID:                edbMiner.MinerID,
		N2NHost:           edbMiner.N2NHost,
		Host:              edbMiner.Host,
		Port:              edbMiner.Port,
		Path:              edbMiner.Path,
		PublicKey:         edbMiner.PublicKey,
		ShortName:         edbMiner.ShortName,
		BuildTag:          edbMiner.BuildTag,
		TotalStaked:       int64(edbMiner.TotalStaked),
		Delete:            edbMiner.Delete,
		DelegateWallet:    edbMiner.DelegateWallet,
		ServiceCharge:     edbMiner.ServiceCharge,
		NumberOfDelegates: edbMiner.NumberOfDelegates,
		MinStake:          edbMiner.MinStake,
		MaxStake:          edbMiner.MaxStake,
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
		TotalStaked:       state.Balance(mn.TotalStaked),
		Delete:            mn.Delete,
		DelegateWallet:    mn.DelegateWallet,
		ServiceCharge:     mn.ServiceCharge,
		NumberOfDelegates: mn.NumberOfDelegates,
		MinStake:          mn.MinStake,
		MaxStake:          mn.MaxStake,
		LastHealthCheck:   mn.LastHealthCheck,
		Rewards:           mn.Reward,
		Active:            mn.Status == node.NodeStatusActive,
		Longitude:         mn.Geolocation.Longitude,
		Latitude:          mn.Geolocation.Latitude,
	}
}

func emitAddMiner(mn *MinerNode, balances cstate.StateContextI) error {

	logging.Logger.Info("emitting add miner event")

	data, err := json.Marshal(minerNodeToMinerTable(mn))
	if err != nil {
		return fmt.Errorf("marshalling miner: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddMiner, mn.ID, string(data))

	return nil
}

func emitAddOrOverwriteMiner(mn *MinerNode, balances cstate.StateContextI) error {

	logging.Logger.Info("emitting add or overwrite miner event")

	data, err := json.Marshal(minerNodeToMinerTable(mn))
	if err != nil {
		return fmt.Errorf("marshalling miner: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteMiner, mn.ID, string(data))

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
			"delegate_wallet":     mn.DelegateWallet,
			"service_charge":      mn.ServiceCharge,
			"number_of_delegates": mn.NumberOfDelegates,
			"min_stake":           mn.MinStake,
			"max_stake":           mn.MaxStake,
			"last_health_check":   mn.LastHealthCheck,
			"rewards":             mn.Reward,
			"longitude":           0,
			"latitude":            0,
		},
	}

	if updateStatus {
		dbUpdates.Updates["active"] = mn.Status == node.NodeStatusActive
	}

	data, err := json.Marshal(dbUpdates)
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, mn.ID, string(data))
	return nil
}

func emitDeleteMiner(id string, balances cstate.StateContextI) error {

	logging.Logger.Info("emitting delete miner event")

	balances.EmitEvent(event.TypeStats, event.TagDeleteMiner, id, id)
	return nil
}
