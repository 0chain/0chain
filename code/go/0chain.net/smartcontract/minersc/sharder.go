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

func (msc *MinerSmartContract) doesSharderExist(pkey datastore.Key, statectx c_state.StateContextI) bool {
	mbits, err := statectx.GetTrieNode(pkey)
	if err != nil {
		Logger.Warn("unexpected error", zap.Error(err))
	}
	if mbits != nil {
		return true
	}
	return false
}

//AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddSharder(t *transaction.Transaction, input []byte, gn *globalNode, statectx c_state.StateContextI) (string, error) {
	Logger.Info("try to add sharder", zap.Any("txn", t))
	allShardersList, err := msc.getShardersList(statectx, AllShardersKey)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.New("add_sharder_failed - Failed to get miner list" + err.Error())
	}
	msc.verifySharderState(statectx, AllShardersKey, "Checking allminerslist in the beginning")

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

	_, err = msc.getSharderNode(datastore.Key(ADDRESS+newSharder.ID), newSharder.ID, statectx)
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
func (msc *MinerSmartContract) verifySharderState(statectx c_state.StateContextI, key datastore.Key, msg string) {
	allSharderList, err := msc.getShardersList(statectx, key)
	if err != nil {
		Logger.Info(msg + " getShardersList_failed - Failed to retrieve existing miners list")
		return
	}
	if allSharderList == nil || len(allSharderList.Nodes) == 0 {
		Logger.Info(msg + " allSharderList is empty")
		return
	}

	Logger.Info(msg)
	for _, sharder := range allSharderList.Nodes {
		Logger.Info("allSharderList", zap.String("url", sharder.N2NHost), zap.String("ID", sharder.ID))
	}

}

func (msc *MinerSmartContract) getShardersList(statectx c_state.StateContextI, key datastore.Key) (*MinerNodes, error) {
	allMinersList := &MinerNodes{}
	allMinersBytes, err := statectx.GetTrieNode(key)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("getShardersList_failed",
			fmt.Sprintf("Failed to retrieve existing sharders list: %v", err))
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	err = allMinersList.Decode(allMinersBytes.Encode())
	if err != nil {
		return nil, err
	}
	return allMinersList, nil
}

func (msc *MinerSmartContract) getSharderNode(key datastore.Key, id string, balances c_state.StateContextI) (*MinerNode, error) {
	mn := NewMinerNode()
	mn.ID = id
	ms, err := balances.GetTrieNode(key)
	if err == util.ErrValueNotPresent {
		return mn, err
	} else if err != nil {
		return nil, err
	}
	err = mn.Decode(ms.Encode())
	if err != nil {
		return nil, err
	}
	return mn, nil
}

func (msc *MinerSmartContract) sharderKeep(t *transaction.Transaction, input []byte, gn *globalNode,
	balances c_state.StateContextI) (result string, err2 error) {
	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return "", err
	}
	if pn.Phase != Contribute {
		return "", common.NewError("sharder_keep_failed", "this is not the correct phase to contribute (sharder keep)")
	}

	sharderKeepList, err := msc.getShardersList(balances, ShardersKeepKey)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.New("sharder_keep_failed - Failed to get miner list" + err.Error())
	}
	msc.verifySharderState(balances, ShardersKeepKey, "Checking sharderKeepList in the beginning")

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

	//check new sharder
	allShardersList, err := msc.getShardersList(balances, AllShardersKey)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.New("sharder_keep_failed - Failed to get miner list" + err.Error())
	}
	if allShardersList.FindNodeById(newSharder.ID) == nil {
		return "", common.NewErrorf("failed to add sharder", "unknown sharder: %v", newSharder.ID)
	}

	if sharderKeepList.FindNodeById(newSharder.ID) != nil {
		return "", common.NewErrorf("failed to add sharder", "sharder already exists: %v", newSharder.ID)
	}

	sharderKeepList.Nodes = append(sharderKeepList.Nodes, newSharder)
	if _, err := balances.InsertTrieNode(ShardersKeepKey, sharderKeepList); err != nil {
		return "", err
	}
	msc.verifyMinerState(balances, "Checking allsharderslist afterInsert")
	buff := newSharder.Encode()
	return string(buff), nil
}
