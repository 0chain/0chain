package minersc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
)

func sharderTableToSharderNode(edbSharder event.Sharder) MinerNode {

	var status = node.NodeStatusInactive
	if edbSharder.Active {
		status = node.NodeStatusActive
	}
	msn := SimpleNode{
		ID:          edbSharder.SharderID,
		N2NHost:     edbSharder.N2NHost,
		Host:        edbSharder.Host,
		Port:        edbSharder.Port,
		Path:        edbSharder.Path,
		PublicKey:   edbSharder.PublicKey,
		Description: edbSharder.Description,
		BuildTag:    edbSharder.BuildTag,
		TotalStaked: edbSharder.TotalStaked,
		Delete:      edbSharder.Delete,

		LastHealthCheck: edbSharder.LastHealthCheck,
		Geolocation: SimpleNodeGeolocation{
			Latitude:  edbSharder.Latitude,
			Longitude: edbSharder.Longitude,
		},
		NodeType: NodeTypeSharder,
		Status:   status,
	}

	return MinerNode{
		SimpleNode: &msn,
		StakePool: &stakepool.StakePool{
			Reward: edbSharder.Rewards,
			Settings: stakepool.Settings{
				DelegateWallet:     edbSharder.DelegateWallet,
				ServiceChargeRatio: edbSharder.ServiceCharge,
				MaxNumDelegates:    edbSharder.NumberOfDelegates,
				MinStake:           edbSharder.MinStake,
				MaxStake:           edbSharder.MaxStake,
			},
		},
	}

}

func sharderNodeToSharderTable(sn *MinerNode) event.Sharder {

	return event.Sharder{
		SharderID:         sn.ID,
		N2NHost:           sn.N2NHost,
		Host:              sn.Host,
		Port:              sn.Port,
		Path:              sn.Path,
		PublicKey:         sn.PublicKey,
		Description:         sn.Description,
		BuildTag:          sn.BuildTag,
		TotalStaked:       sn.TotalStaked,
		Delete:            sn.Delete,
		DelegateWallet:    sn.Settings.DelegateWallet,
		ServiceCharge:     sn.Settings.ServiceChargeRatio,
		NumberOfDelegates: sn.Settings.MaxNumDelegates,
		MinStake:          sn.Settings.MinStake,
		MaxStake:          sn.Settings.MaxStake,
		LastHealthCheck:   sn.LastHealthCheck,
		Rewards:           sn.Reward,
		Active:            sn.Status == node.NodeStatusActive,
		Longitude:         sn.Geolocation.Longitude,
		Latitude:          sn.Geolocation.Latitude,
	}
}

func emitAddSharder(sn *MinerNode, balances cstate.StateContextI) error {

	data, err := json.Marshal(sharderNodeToSharderTable(sn))
	if err != nil {
		return fmt.Errorf("marshalling sharder: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddSharder, sn.ID, string(data))

	return nil
}

func emitAddOrOverwriteSharder(sn *MinerNode, balances cstate.StateContextI) error {

	data, err := json.Marshal(sharderNodeToSharderTable(sn))
	if err != nil {
		return fmt.Errorf("marshalling sharder: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteSharder, sn.ID, string(data))

	return nil
}

func emitUpdateSharder(sn *MinerNode, balances cstate.StateContextI, updateStatus bool) error {

	dbUpdates := dbs.DbUpdates{
		Id: sn.ID,
		Updates: map[string]interface{}{
			"n2n_host":            sn.N2NHost,
			"host":                sn.Host,
			"port":                sn.Port,
			"path":                sn.Path,
			"public_key":          sn.PublicKey,
			"description":          sn.Description,
			"build_tag":           sn.BuildTag,
			"total_staked":        sn.TotalStaked,
			"delete":              sn.Delete,
			"delegate_wallet":     sn.Settings.DelegateWallet,
			"service_charge":      sn.Settings.ServiceChargeRatio,
			"number_of_delegates": sn.Settings.MaxNumDelegates,
			"min_stake":           sn.Settings.MinStake,
			"max_stake":           sn.Settings.MaxStake,
			"last_health_check":   sn.LastHealthCheck,
			"longitude":           sn.SimpleNode.Geolocation.Longitude,
			"latitude":            sn.SimpleNode.Geolocation.Latitude,
			"rewards":             sn.Reward,
		},
	}

	if updateStatus {
		dbUpdates.Updates["active"] = sn.Status == node.NodeStatusActive
	}

	data, err := json.Marshal(dbUpdates)
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateSharder, sn.ID, string(data))
	return nil
}

func emitDeleteSharder(id string, balances cstate.StateContextI) error {

	balances.EmitEvent(event.TypeStats, event.TagDeleteSharder, id, id)
	return nil
}
