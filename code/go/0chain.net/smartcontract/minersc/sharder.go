package minersc

import (
	"0chain.net/chaincore/state"
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

// AddSharder function to handle miner register
func (msc *MinerSmartContract) AddSharder(
	t *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {

	logging.Logger.Info("add_sharder", zap.Any("txn", t))

	var newSharder = NewMinerNode()
	if err = newSharder.Decode(input); err != nil {
		logging.Logger.Error("Error in decoding the input", zap.Error(err))
		return "", common.NewErrorf("add_sharder", "decoding request: %v", err)
	}

	if err = newSharder.Validate(); err != nil {
		return "", common.NewErrorf("add_sharder", "invalid input: %v", err)
	}

	allSharders, err := getAllShardersList(balances)
	if err != nil {
		logging.Logger.Error("add_sharder: failed to get sharders list", zap.Error(err))
		return "", common.NewErrorf("add_sharder", "getting all sharders list: %v", err)
	}

	verifyAllShardersState(balances, "Checking all sharders list in the beginning")

	if newSharder.DelegateWallet == "" {
		newSharder.DelegateWallet = newSharder.ID
	}

	newSharder.LastHealthCheck = t.CreationDate

	logging.Logger.Info("The new sharder info",
		zap.String("base URL", newSharder.N2NHost),
		zap.String("ID", newSharder.ID),
		zap.String("pkey", newSharder.PublicKey),
		zap.Any("mscID", msc.ID),
		zap.String("delegate_wallet", newSharder.DelegateWallet),
		zap.Float64("service_charge", newSharder.ServiceCharge),
		zap.Int("number_of_delegates", newSharder.NumberOfDelegates),
		zap.Int64("min_stake", int64(newSharder.MinStake)),
		zap.Int64("max_stake", int64(newSharder.MaxStake)))

	logging.Logger.Info("SharderNode", zap.Any("node", newSharder))

	if newSharder.PublicKey == "" || newSharder.ID == "" {
		logging.Logger.Error("public key or ID is empty")
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

	existing, err := msc.getSharderNode(newSharder.ID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("add_sharder", "unexpected error: %v", err)
	}

	// if found
	if err == nil {
		// and found in all
		if allSharders.FindNodeById(newSharder.ID) != nil {
			return string(newSharder.Encode()), nil
		}
		// otherwise the sharder has saved by block sharders reward
		newSharder.Stat.SharderRewards = existing.Stat.SharderRewards
	}

	newSharder.NodeType = NodeTypeSharder // set node type

	if err = quickFixDuplicateHosts(newSharder, allSharders.Nodes); err != nil {
		return "", common.NewError("add_sharder", err.Error())
	}

	allSharders.Nodes = append(allSharders.Nodes, newSharder)

	// save the added sharder
	_, err = balances.InsertTrieNode(newSharder.getKey(), newSharder)
	if err != nil {
		return "", common.NewErrorf("add_sharder", "saving sharder: %v", err)
	}

	// save all sharders list
	if err = updateAllShardersList(balances, allSharders); err != nil {
		return "", common.NewErrorf("add_sharder", "saving all sharders list: %v", err)
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

	// set node type -- miner
	if err = sn.save(balances); err != nil {
		return "", common.NewError("delete_sharder", err.Error())
	}

	resp = string(sn.Encode())
	return
}

func (msc *MinerSmartContract) deleteSharderFromViewChange(sn *MinerNode, balances cstate.StateContextI) (err error) {
	var pn *PhaseNode
	if pn, err = GetPhaseNode(balances); err != nil {
		return
	}
	if pn.Phase == Unknown {
		err = common.NewError("failed to delete from view change", "phase is unknown")
		return
	}
	if pn.Phase != Wait {
		sharders := &MinerNodes{}
		if sharders, err = getShardersKeepList(balances); err != nil {
			logging.Logger.Error("delete_sharder_from_view_change: Error in getting list from the DB",
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
func verifyAllShardersState(balances cstate.StateContextI, msg string) {
	shardersList, err := getAllShardersList(balances)
	if err != nil {
		logging.Logger.Error("verify_all_sharder_state_failed", zap.Error(err))
		return
	}

	if shardersList == nil || len(shardersList.Nodes) == 0 {
		logging.Logger.Info(msg + " shardersList is empty")
		return
	}

	logging.Logger.Info(msg)
	for _, sharder := range shardersList.Nodes {
		logging.Logger.Info("shardersList", zap.String("url", sharder.N2NHost), zap.String("ID", sharder.ID))
	}
}

func verifyShardersKeepState(balances cstate.StateContextI, msg string) {
	shardersList, err := getShardersKeepList(balances)
	if err != nil {
		logging.Logger.Error("verify_sharder_keep_state_failed", zap.Error(err))
		return
	}

	if shardersList == nil || len(shardersList.Nodes) == 0 {
		logging.Logger.Info(msg + " shardersList is empty")
		return
	}

	logging.Logger.Info(msg)
	for _, sharder := range shardersList.Nodes {
		logging.Logger.Info("shardersList", zap.String("url", sharder.N2NHost), zap.String("ID", sharder.ID))
	}
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

	pn, err := GetPhaseNode(balances)
	if err != nil {
		return "", err
	}
	if pn.Phase != Contribute {
		return "", common.NewError("sharder_keep",
			"this is not the correct phase to contribute (sharder keep)")
	}

	newSharder := NewMinerNode()
	err = newSharder.Decode(input)
	if err != nil {
		logging.Logger.Error("Error in decoding the input", zap.Error(err))
		return "", err
	}

	if err = newSharder.Validate(); err != nil {
		return "", common.NewErrorf("sharder_keep", "invalid input: %v", err)
	}

	sharderKeepList, err := getShardersKeepList(balances)
	if err != nil {
		logging.Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", common.NewErrorf("sharder_keep",
			"Failed to get miner list: %v", err)
	}
	verifyShardersKeepState(balances, "Checking sharderKeepList in the beginning")

	logging.Logger.Info("The new sharder info",
		zap.String("base URL", newSharder.N2NHost),
		zap.String("ID", newSharder.ID),
		zap.String("pkey", newSharder.PublicKey),
		zap.Any("mscID", msc.ID))
	logging.Logger.Info("SharderNode", zap.Any("node", newSharder))
	if newSharder.PublicKey == "" || newSharder.ID == "" {
		logging.Logger.Error("public key or ID is empty")
		return "", errors.New("PublicKey or the ID is empty. Cannot proceed")
	}

	//check new sharder
	allShardersList, err := getAllShardersList(balances)
	if err != nil {
		logging.Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", common.NewErrorf("sharder_keep",
			"Failed to get miner list: %v", err)
	}
	if allShardersList.FindNodeById(newSharder.ID) == nil {
		return "", common.NewErrorf("sharder_keep", "unknown sharder: %v", newSharder.ID)
	}

	if sharderKeepList.FindNodeById(newSharder.ID) != nil {
		// do not return error for sharder already exist,
		logging.Logger.Debug("Add sharder already exists", zap.String("ID", newSharder.ID))
		return string(newSharder.Encode()), nil
	}

	sharderKeepList.Nodes = append(sharderKeepList.Nodes, newSharder)
	if err := updateShardersKeepList(balances, sharderKeepList); err != nil {
		return "", err
	}
	msc.verifyMinerState(balances, "Checking allsharderslist afterInsert")
	buff := newSharder.Encode()
	return string(buff), nil
}
