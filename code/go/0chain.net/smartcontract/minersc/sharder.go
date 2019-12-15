package minersc

import (
	"errors"
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) doesSharderExist(pkey datastore.Key, statectx c_state.StateContextI) bool {
	mbits, _ := statectx.GetTrieNode(pkey)
	if mbits != nil {
		return true
	}
	return false
}

//AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddSharder(t *transaction.Transaction, input []byte, statectx c_state.StateContextI) (string, error) {
	Logger.Info("try to add sharder", zap.Any("txn", t))
	allShardersList, err := msc.getShardersList(statectx)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.New("add_sharder_failed - Failed to get miner list" + err.Error())
	}
	msc.verifySharderState(statectx, "Checking allminerslist in the beginning")

	newSharder := NewMinerNode()
	err = newSharder.Decode(input)
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))

		return "", err
	}
	Logger.Info("The new sharder info", zap.String("base URL", newSharder.N2NHost), zap.String("ID", newSharder.ID), zap.String("pkey", newSharder.PublicKey), zap.Any("mscID", msc.ID))
	Logger.Info("SharderNode", zap.Any("node", newSharder))
	if newSharder.PublicKey == "" || newSharder.ID == "" {
		Logger.Error("public key or ID is empty")
		return "", errors.New("PublicKey or the ID is empty. Cannot proceed")
	}

	_, err = msc.getSharderNode(newSharder.ID, statectx)
	if err != util.ErrValueNotPresent {
		return "", common.NewError("failed to add sharder", "sharder already exists")
	}

	pool := sci.NewDelegatePool()
	transfer, _, err := pool.DigPool(t.Hash, t)
	if err != nil {
		return "", common.NewError("failed to add sharder", fmt.Sprintf("error digging delegate pool: %v", err.Error()))
	}
	statectx.AddTransfer(transfer)
	newSharder.Pending[t.Hash] = pool
	allShardersList.Nodes = append(allShardersList.Nodes, newSharder)
	statectx.InsertTrieNode(AllShardersKey, allShardersList)
	statectx.InsertTrieNode(newSharder.getKey(), newSharder)
	msc.verifyMinerState(statectx, "Checking allsharderslist afterInsert")

	buff := newSharder.Encode()
	return string(buff), nil
}

//------------- local functions ---------------------
func (msc *MinerSmartContract) verifySharderState(statectx c_state.StateContextI, msg string) {
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

func (msc *MinerSmartContract) getShardersList(statectx c_state.StateContextI) (*MinerNodes, error) {
	allMinersList := &MinerNodes{}
	allMinersBytes, err := statectx.GetTrieNode(AllShardersKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, errors.New("getMinersList_failed - Failed to retrieve existing miners list")
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	allMinersList.Decode(allMinersBytes.Encode())
	return allMinersList, nil
}

func (msc *MinerSmartContract) getSharderNode(id string, balances c_state.StateContextI) (*MinerNode, error) {
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
