package minersc

import (
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
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

func (msc *MinerSmartContract) UpdateSharderSettings(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var update = NewMinerNode()
	if err = update.Decode(inputData); err != nil {
		return "", common.NewErrorf("update_settings",
			"decoding request: %v", err)
	}

	if update.ServiceCharge < 0 {
		return "", common.NewErrorf("update_settings",
			"invalid negative service charge: %v", update.ServiceCharge)
	}

	if update.ServiceCharge > gn.MaxCharge {
		return "", common.NewErrorf("update_settings",
			"max_charge is greater than allowed by SC: %v > %v",
			update.ServiceCharge, gn.MaxCharge)
	}

	if update.NumberOfDelegates < 0 {
		return "", common.NewErrorf("update_settings",
			"invalid negative number_of_delegates: %v", update.ServiceCharge)
	}

	if update.NumberOfDelegates > gn.MaxDelegates {
		return "", common.NewErrorf("add_miner_failed",
			"number_of_delegates greater than max_delegates of SC: %v > %v",
			update.ServiceCharge, gn.MaxDelegates)
	}

	if update.MinStake < gn.MinStake {
		return "", common.NewErrorf("update_settings",
			"min_stake is less than allowed by SC: %v > %v",
			update.MinStake, gn.MinStake)
	}

	if update.MaxStake < gn.MaxStake {
		return "", common.NewErrorf("update_settings",
			"max_stake is greater than allowed by SC: %v > %v",
			update.MaxStake, gn.MaxStake)
	}

	var sn *MinerNode
	sn, err = msc.getSharderNode(update.ID, balances)
	if err != nil {
		return "", common.NewError("update_settings", err.Error())
	}
	if sn.Delete {
		return "", common.NewError("update_settings", "can't update settings of sharder being deleted")
	}
	if sn.DelegateWallet != t.ClientID {
		return "", common.NewError("update_setings", "access denied")
	}

	sn.ServiceCharge = update.ServiceCharge
	sn.NumberOfDelegates = update.NumberOfDelegates
	sn.MinStake = update.MinStake
	sn.MaxStake = update.MaxStake

	if err = sn.save(balances); err != nil {
		return "", common.NewErrorf("update_setings", "saving: %v", err)
	}

	return string(sn.Encode()), nil
}

// DeleteSharder Function to handle removing a sharder from the chain
func (msc *MinerSmartContract) DeleteSharder(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var deleteSharder = NewMinerNode()
	if err = deleteSharder.Decode(inputData); err != nil {
		return "", common.NewErrorf("delete_sharder",
			"decoding request: %v", err)
	}

	var sn *MinerNode
	sn, err = msc.getSharderNode(deleteSharder.ID, balances)
	if err != nil {
		return "", common.NewError("delete_sharder", err.Error())
	}
	sn.Delete = true

	// deleting pending pools
	for key, pool := range sn.Pending {
		var un *UserNode
		if un, err = msc.getUserNode(pool.DelegateID, balances); err != nil {
			return "", common.NewErrorf("delete_sharder",
				"getting user node: %v", err)
		}

		var transfer *state.Transfer
		transfer, resp, err = pool.EmptyPool(msc.ID, pool.DelegateID, nil)
		if err != nil {
			return "", common.NewErrorf("delete_sharder",
				"error emptying delegate pool: %v", err)
		}

		if err = balances.AddTransfer(transfer); err != nil {
			return "", common.NewErrorf("delete_sharder",
				"adding transfer: %v", err)
		}

		delete(un.Pools, key)
		delete(sn.Pending, key)

		if err = un.save(balances); err != nil {
			return "", common.NewError("delete_sharder", err.Error())
		}
	}

	// deleting active pools
	for key, pool := range sn.Active {
		if pool.Status == DELETING {
			continue
		}

		pool.Status = DELETING // mark as deleting
		pool.TokenLockInterface = &ViewChangeLock{
			Owner:               pool.DelegateID,
			DeleteViewChangeSet: true,
			DeleteVC:            gn.ViewChange,
		}
		sn.Deleting[key] = pool // add to deleting
	}

	if err = msc.deleteSharderFromViewChange(sn, balances); err != nil {
		return "", err
	}

	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()

	var all *MinerNodes
	if all, err = msc.getShardersList(balances, AllShardersKey); err != nil {
		Logger.Error("delete_sharder: Error in getting list from the DB",
			zap.Error(err))
		return "", common.NewErrorf("delete_sharder",
			"failed to get sharder list: %v", err)
	}
	msc.verifySharderState(balances, AllShardersKey,
		"delete_sharder: checking all sharders list in the beginning")

	for i, v := range all.Nodes {
		if v.ID == sn.ID {
			all.Nodes[i] = sn
			break
		}
	}

	if _, err = balances.InsertTrieNode(AllShardersKey, all); err != nil {
		return "", common.NewErrorf("delete_sharder",
			"saving all sharders list: %v", err)
	}

	// set node type -- miner
	if err = sn.save(balances); err != nil {
		return "", common.NewError("delete_sharder", err.Error())
	}

	msc.verifySharderState(balances, AllShardersKey,
		"delete_sharder: Checking all sharders list afterInsert")

	resp = string(sn.Encode())
	return
}

func (msc *MinerSmartContract) deleteSharderFromViewChange(sn *MinerNode, balances cstate.StateContextI) (err error) {
	var pn *PhaseNode
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return
	}
	if pn.Phase == Unknown {
		err = common.NewError("failed to delete from view change", "phase is unknown")
		return
	}
	if pn.Phase != Wait {
		sharders := &MinerNodes{}
		if sharders, err = msc.getShardersList(balances, ShardersKeepKey); err != nil {
			Logger.Error("delete_sharder_from_view_change: Error in getting list from the DB",
				zap.Error(err))
			return common.NewErrorf("delete_sharder_from_view_change",
				"failed to get sharders list: %v", err)
		}
		for i, v := range sharders.Nodes {
			if v.ID == sn.ID {
				sharders.Nodes = append(sharders.Nodes[:i], sharders.Nodes[i+1:]...)
				break
			}
		}
		if _, err = balances.InsertTrieNode(ShardersKeepKey, sharders); err != nil {
			return
		}
	} else {
		err = common.NewError("failed to delete from view change", "magic block has already been created for next view change")
		return
	}
	return
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
		return nil, err
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
