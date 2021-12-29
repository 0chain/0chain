package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

func minerTableToMinerNode(edbMiner *event.Miner) *MinerNode {

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
		Stat: Stat{
			GeneratorRewards: edbMiner.Rewards,
			GeneratorFees:    edbMiner.Fees,
		},
		LastHealthCheck: edbMiner.LastHealthCheck,
	}

	return &MinerNode{
		SimpleNode: &msn,
	}

}

func minerNodeToMinerTable(mn *MinerNode) event.Miner {

	return event.Miner{
		Model:             gorm.Model{},
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
		Rewards:           mn.Stat.GeneratorRewards,
		Fees:              mn.Stat.GeneratorFees,
		Longitude:         0,
		Latitude:          0,
	}
}

func emitAddMiner(mn *MinerNode, balances cstate.StateContextI) error {

	data, err := json.Marshal(minerNodeToMinerTable(mn))
	if err != nil {
		return fmt.Errorf("marshalling miner: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddMiner, mn.ID, string(data))

	return nil
}

func emitAddOrOverwriteMiner(mn *MinerNode, balances cstate.StateContextI) error {

	data, err := json.Marshal(minerNodeToMinerTable(mn))
	if err != nil {
		return fmt.Errorf("marshalling miner: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddOrOverwriteMiner, mn.ID, string(data))

	return nil
}

func emitUpdateMiner(mn *MinerNode, balances cstate.StateContextI) error {

	data, err := json.Marshal(dbs.DbUpdates{
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
			"rewards":             mn.SimpleNode.Stat.GeneratorRewards,
			"fees":                mn.SimpleNode.Stat.GeneratorFees,
			"longitude":           0,
			"latitude":            0,
		},
	})
	if err != nil {
		return fmt.Errorf("marshalling update: %v", err)
	}
	balances.EmitEvent(event.TypeStats, event.TagUpdateMiner, mn.ID, string(data))
	return nil
}
