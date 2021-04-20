package minersc

import (
	"0chain.net/smartcontract"
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

// AddSharder function to handle miner register
func (msc *MinerSmartContract) AddSharder(t *transaction.Transaction,
	input []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	Logger.Info("try to add sharder", zap.Any("txn", t))
	var all *MinerNodes
	if all, err = msc.getShardersList(balances, AllShardersKey); err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", common.NewErrorf("add_sharder",
			"getting miner list: %v", err)
	}

	msc.verifySharderState(balances, AllShardersKey,
		"Checking all sharders list in the beginning")

	var newSharder = NewMinerNode()
	if err = newSharder.Decode(input); err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))
		return "", common.NewErrorf("add_sharder", "decoding request: %v", err)
	}

	if newSharder.DelegateWallet == "" {
		newSharder.DelegateWallet = newSharder.ID
	}

	newSharder.LastHealthCheck = t.CreationDate

	Logger.Info("The new sharder info",
		zap.String("base URL", newSharder.N2NHost),
		zap.String("ID", newSharder.ID),
		zap.String("pkey", newSharder.PublicKey),
		zap.Any("mscID", msc.ID),
		zap.String("delegate_wallet", newSharder.DelegateWallet),
		zap.Float64("service_charge", newSharder.ServiceCharge),
		zap.Int("number_of_delegates", newSharder.NumberOfDelegates),
		zap.Int64("min_stake", int64(newSharder.MinStake)),
		zap.Int64("max_stake", int64(newSharder.MaxStake)))

	Logger.Info("SharderNode", zap.Any("node", newSharder))

	if newSharder.PublicKey == "" || newSharder.ID == "" {
		Logger.Error("public key or ID is empty")
		return "", common.NewError("add_sharder",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	if newSharder.NumberOfDelegates < 0 {
		return "", common.NewErrorf("add_sharder",
			"invalid negative number_of_delegates: %v",
			newSharder.ServiceCharge)
	}

	if newSharder.MinStake < gn.MinStake {
		return "", common.NewErrorf("add_sharder",
			"min_stake is less than allowed by SC: %v > %v",
			newSharder.MinStake, gn.MinStake)
	}

	if newSharder.MaxStake < gn.MaxStake {
		return "", common.NewErrorf("add_sharder",
			"max_stake is greater than allowed by SC: %v > %v",
			newSharder.MaxStake, gn.MaxStake)
	}

	var existing *MinerNode
	existing, err = msc.getSharderNode(newSharder.ID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("add_sharder", "unexpected error: %v", err)
	}

	// if found
	if err == nil {
		// and found in all
		if all.FindNodeById(newSharder.ID) != nil {
			return "", common.NewError("add_sharder", "sharder already exists")
		}
		// otherwise the sharder has saved by block sharders reward
		newSharder.Stat.SharderRewards = existing.Stat.SharderRewards
	}

	newSharder.NodeType = NodeTypeSharder // set node type

	// add to all
	all.Nodes = append(all.Nodes, newSharder)
	// save the added sharder
	_, err = balances.InsertTrieNode(newSharder.getKey(), newSharder)
	if err != nil {
		return "", common.NewErrorf("add_sharder",
			"saving sharder: %v", err)
	}
	// save all sharders list
	if _, err = balances.InsertTrieNode(AllShardersKey, all); err != nil {
		return "", common.NewErrorf("add_sharder",
			"saving all sharders list: %v", err)
	}

	msc.verifyMinerState(balances, "checking all sharders list after insert")

	return string(newSharder.Encode()), nil
}

//------------- local functions ---------------------
func (msc *MinerSmartContract) verifySharderState(balances cstate.StateContextI, key datastore.Key, msg string) {
	allSharderList, err := msc.getShardersList(balances, key)
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

func (msc *MinerSmartContract) getShardersList(balances cstate.StateContextI,
	key datastore.Key) (*MinerNodes, error) {

	allMinersList := &MinerNodes{}
	allMinersBytes, err := balances.GetTrieNode(key)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, common.NewError("getShardersList_failed",
			fmt.Sprintf("Failed to retrieve existing sharders list: %v", err))
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	err = allMinersList.Decode(allMinersBytes.Encode())
	if err != nil {
		return nil, smartcontract.NewError(smartcontract.DecodingErr, err)
	}
	return allMinersList, nil
}

func (msc *MinerSmartContract) getSharderNode(sid string,
	balances cstate.StateContextI) (sn *MinerNode, err error) {

	var ss util.Serializable
	ss, err = balances.GetTrieNode(getSharderKey(sid))
	if err != nil && err != util.ErrValueNotPresent {
		return // unexpected error
	}

	sn = NewMinerNode()
	sn.ID = sid

	if err == util.ErrValueNotPresent {
		return // with error ErrValueNotPresent (that's very stupid)
	}

	if err = sn.Decode(ss.Encode()); err != nil {
		return nil, fmt.Errorf("invalid state: decoding sharder: %v", err)
	}

	return // got it!
}

func (msc *MinerSmartContract) sharderKeep(t *transaction.Transaction,
	input []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err2 error) {

	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return "", err
	}
	if pn.Phase != Contribute {
		return "", common.NewError("sharder_keep_failed",
			"this is not the correct phase to contribute (sharder keep)")
	}

	sharderKeepList, err := msc.getShardersList(balances, ShardersKeepKey)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", common.NewErrorf("sharder_keep_failed",
			"Failed to get miner list: %v", err)
	}
	msc.verifySharderState(balances, ShardersKeepKey, "Checking sharderKeepList in the beginning")

	newSharder := NewMinerNode()
	err = newSharder.Decode(input)
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))

		return "", err
	}
	Logger.Info("The new sharder info",
		zap.String("base URL", newSharder.N2NHost),
		zap.String("ID", newSharder.ID),
		zap.String("pkey", newSharder.PublicKey),
		zap.Any("mscID", msc.ID))
	Logger.Info("SharderNode", zap.Any("node", newSharder))
	if newSharder.PublicKey == "" || newSharder.ID == "" {
		Logger.Error("public key or ID is empty")
		return "", errors.New("PublicKey or the ID is empty. Cannot proceed")
	}

	//check new sharder
	allShardersList, err := msc.getShardersList(balances, AllShardersKey)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", common.NewErrorf("sharder_keep_failed",
			"Failed to get miner list: %v", err)
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
