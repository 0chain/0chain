package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (msc *MinerSmartContract) minerHealthCheck(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	if err := minersPartitions.updateNode(balances, GetNodeKey(t.ClientID), func(m *MinerNode) error {
		m.LastHealthCheck = t.CreationDate
		resp = string(m.Encode())
		return nil
	}); err != nil {
		return "", common.NewError("miner_health_check_failed", err.Error())
	}

	return resp, nil
}

func (msc *MinerSmartContract) sharderHealthCheck(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	if err := shardersPartitions.updateNode(balances, GetNodeKey(t.ClientID), func(n *MinerNode) error {
		n.LastHealthCheck = t.CreationDate
		resp = string(n.Encode())
		return nil
	}); err != nil {
		return "", common.NewError("sharder_health_check_failed", err.Error())
	}

	return resp, nil
}
