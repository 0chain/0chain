package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

func minerTableToMinerNode(edbMiner event.Miner) MinerNode {

	var isMinerActive = node.NodeStatusActive
	if !edbMiner.Active {
		isMinerActive = node.NodeStatusInactive
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
		TotalStaked:       edbMiner.TotalStaked,
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
		Active:          isMinerActive,
	}

	return MinerNode{
		SimpleNode: &msn,
	}

}

func minerNodeToMinerTable(m *MinerNode) event.Miner {

	return event.Miner{
		Model:             gorm.Model{},
		MinerID:           m.ID,
		N2NHost:           m.N2NHost,
		Host:              m.Host,
		Port:              m.Port,
		Path:              m.Path,
		PublicKey:         m.PublicKey,
		ShortName:         m.ShortName,
		BuildTag:          m.BuildTag,
		TotalStaked:       m.TotalStaked,
		Delete:            m.Delete,
		DelegateWallet:    m.DelegateWallet,
		ServiceCharge:     m.ServiceCharge,
		NumberOfDelegates: m.NumberOfDelegates,
		MinStake:          m.MinStake,
		MaxStake:          m.MaxStake,
		LastHealthCheck:   m.LastHealthCheck,
		Rewards:           m.Stat.GeneratorRewards,
		Fees:              m.Stat.GeneratorFees,
		TotalStake:        state.Balance(m.TotalStaked),
		Active:            m.SimpleNode.Active == node.NodeStatusActive,
		Longitude:         0,
		Latitude:          0,
	}
}

func emitAddMiner(m *MinerNode, balances cstate.StateContextI) error {

	data, err := json.Marshal(minerNodeToMinerTable(m))
	if err != nil {
		return fmt.Errorf("marshalling miner: %v", err)
	}

	balances.EmitEvent(event.TypeStats, event.TagAddMiner, m.ID, string(data))

	return nil
}
