package minersc

import (
	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"errors"
	"fmt"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) doesMinerExist(pkey datastore.Key, statectx c_state.StateContextI) bool {
	mbits, err := statectx.GetTrieNode(pkey)
	if err != nil {
		Logger.Error("GetTrieNode from state context", zap.Error(err))
		return false
	}
	if mbits != nil {
		return true
	}
	return false
}

//AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddMiner(t *transaction.Transaction, input []byte, gn *globalNode, statectx c_state.StateContextI) (string, error) {
	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()

	Logger.Info("try to add miner", zap.Any("txn", t))
	allMinersList, err := msc.getMinersList(statectx)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.New("add_miner_failed - Failed to get miner list" + err.Error())
	}
	msc.verifyMinerState(statectx, "Checking allminerslist in the beginning")

	newMiner := NewMinerNode()
	err = newMiner.Decode(input)
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))
		return "", err
	}
	//log.Println(newMiner)
	Logger.Info("The new miner info", zap.String("base URL", newMiner.N2NHost), zap.String("ID", newMiner.ID), zap.String("pkey", newMiner.PublicKey), zap.Any("mscID", msc.ID))
	Logger.Info("MinerNode", zap.Any("node", newMiner))
	if newMiner.PublicKey == "" || newMiner.ID == "" {
		Logger.Error("public key or ID is empty")
		return "", errors.New("PublicKey or the ID is empty. Cannot proceed")
	}

	_, err = msc.getMinerNode(newMiner.ID, statectx)
	if err != util.ErrValueNotPresent {
		return "", common.NewError("failed to add miner", "miner already exists")
	}

	pool := sci.NewDelegatePool()
	transfer, _, err := pool.DigPool(t.Hash, t)
	if err != nil {
		return "", common.NewError("failed to add miner", fmt.Sprintf("error digging delegate pool: %v", err.Error()))
	}
	statectx.AddTransfer(transfer)
	newMiner.Pending[t.Hash] = pool
	allMinersList.Nodes = append(allMinersList.Nodes, newMiner)
	statectx.InsertTrieNode(AllMinersKey, allMinersList)
	statectx.InsertTrieNode(newMiner.getKey(), newMiner)
	msc.verifyMinerState(statectx, "Checking allminerslist afterInsert")

	buff := newMiner.Encode()
	return string(buff), nil
}

//------------- local functions ---------------------
func (msc *MinerSmartContract) verifyMinerState(statectx c_state.StateContextI, msg string) {
	allMinersList, err := msc.getMinersList(statectx)
	if err != nil {
		Logger.Info(msg + " getMinersList_failed - Failed to retrieve existing miners list")
		return
	}
	if allMinersList == nil || len(allMinersList.Nodes) == 0 {
		Logger.Info(msg + " allminerslist is empty")
		return
	}

	Logger.Info(msg)
	for _, miner := range allMinersList.Nodes {
		Logger.Info("allminerslist", zap.String("url", miner.N2NHost), zap.String("ID", miner.ID))
	}

}

func (msc *MinerSmartContract) GetMinersList(statectx c_state.StateContextI) (*MinerNodes, error) {
	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()
	return msc.getMinersList(statectx)
}

func (msc *MinerSmartContract) getMinersList(statectx c_state.StateContextI) (*MinerNodes, error) {
	allMinersList := &MinerNodes{}
	allMinersBytes, err := statectx.GetTrieNode(AllMinersKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, errors.New("getMinersList_failed - Failed to retrieve existing miners list")
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	allMinersList.Decode(allMinersBytes.Encode())
	return allMinersList, nil
}

func (msc *MinerSmartContract) getMinerNode(id string, balances c_state.StateContextI) (*MinerNode, error) {
	mn := NewMinerNode()
	mn.ID = id
	ms, err := balances.GetTrieNode(mn.getKey())
	if err == util.ErrValueNotPresent {
		return mn, err
	} else if err != nil {
		return nil, err
	}
	mn.Decode(ms.Encode())
	return mn, err
}
