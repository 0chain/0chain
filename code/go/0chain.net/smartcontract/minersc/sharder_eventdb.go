package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"
)

func sharderTableToSharderNode(edbSharder *event.Sharder) *MinerNode {

	var status = node.NodeStatusInactive
	if edbSharder.Active {
		status = node.NodeStatusActive
	}
	msn := SimpleNode{
		ID:                edbSharder.SharderID,
		N2NHost:           edbSharder.N2NHost,
		Host:              edbSharder.Host,
		Port:              edbSharder.Port,
		Path:              edbSharder.Path,
		PublicKey:         edbSharder.PublicKey,
		ShortName:         edbSharder.ShortName,
		BuildTag:          edbSharder.BuildTag,
		TotalStaked:       int64(edbSharder.TotalStaked),
		Delete:            edbSharder.Delete,
		DelegateWallet:    edbSharder.DelegateWallet,
		ServiceCharge:     edbSharder.ServiceCharge,
		NumberOfDelegates: edbSharder.NumberOfDelegates,
		MinStake:          edbSharder.MinStake,
		MaxStake:          edbSharder.MaxStake,
		Stat: Stat{
			GeneratorRewards: edbSharder.Rewards,
			GeneratorFees:    edbSharder.Fees,
		},
		LastHealthCheck: edbSharder.LastHealthCheck,
		Status:          status,
	}

	return &MinerNode{
		SimpleNode: &msn,
	}

}

func sharderNodeToSharderTable(sh *MinerNode) event.Sharder {

	return event.Sharder{
		SharderID:         sh.ID,
		N2NHost:           sh.N2NHost,
		Host:              sh.Host,
		Port:              sh.Port,
		Path:              sh.Path,
		PublicKey:         sh.PublicKey,
		ShortName:         sh.ShortName,
		BuildTag:          sh.BuildTag,
		TotalStaked:       state.Balance(sh.TotalStaked),
		Delete:            sh.Delete,
		DelegateWallet:    sh.DelegateWallet,
		ServiceCharge:     sh.ServiceCharge,
		NumberOfDelegates: sh.NumberOfDelegates,
		MinStake:          sh.MinStake,
		MaxStake:          sh.MaxStake,
		LastHealthCheck:   sh.LastHealthCheck,
		Rewards:           sh.Stat.GeneratorRewards,
		Fees:              sh.Stat.GeneratorFees,
		Active:            sh.Status == node.NodeStatusActive,
		Longitude:         0,
		Latitude:          0,
	}
}

func emitAddSharder(sh *MinerNode, balances cstate.StateContextI) error {

	data, err := json.Marshal(sharderNodeToSharderTable(sh))
	if err != nil {
		return fmt.Errorf("marshalling sharder: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddSharder, sh.ID, string(data))

	return nil
}

func emitAddOrOverwriteSharder(sh *MinerNode, balances cstate.StateContextI, active bool) error {

	data, err := json.Marshal(sharderNodeToSharderTable(sh))
	if err != nil {
		return fmt.Errorf("marshalling sharder: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteSharder, sh.ID, string(data))

	return nil
}

func emitUpdateSharder(sh *MinerNode, balances cstate.StateContextI, updateStatus bool) error {

	dbUpdates := dbs.DbUpdates{
		Id: sh.ID,
		Updates: map[string]interface{}{
			"n2n_host":            sh.N2NHost,
			"host":                sh.Host,
			"port":                sh.Port,
			"path":                sh.Path,
			"public_key":          sh.PublicKey,
			"short_name":          sh.ShortName,
			"build_tag":           sh.BuildTag,
			"total_staked":        sh.TotalStaked,
			"delete":              sh.Delete,
			"delegate_wallet":     sh.DelegateWallet,
			"service_charge":      sh.ServiceCharge,
			"number_of_delegates": sh.NumberOfDelegates,
			"min_stake":           sh.MinStake,
			"max_stake":           sh.MaxStake,
			"last_health_check":   sh.LastHealthCheck,
			"rewards":             sh.SimpleNode.Stat.GeneratorRewards,
			"fees":                sh.SimpleNode.Stat.GeneratorFees,
			"longitude":           0,
			"latitude":            0,
		},
	}

	if updateStatus {
		dbUpdates.Updates["active"] = sh.Status == node.NodeStatusActive
	}

	data, err := json.Marshal(dbUpdates)
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateSharder, sh.ID, string(data))
	return nil
}

func emitDeleteSharder(id string, balances cstate.StateContextI) error {

	balances.EmitEvent(event.TypeStats, event.TagDeleteSharder, id, id)
	return nil
}
