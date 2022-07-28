package minersc

import (
	"errors"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) UpdateSharderSettings(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var update = NewMinerNode()
	if err = update.Decode(inputData); err != nil {
		return "", common.NewErrorf("update_sharder_settings",
			"decoding request: %v", err)
	}

	err = validateNodeSettings(update, gn, "update_sharder_settings")
	if err != nil {
		return "", err
	}

	var sn *MinerNode
	sn, err = msc.getSharderNode(update.ID, balances)
	if err != nil {
		return "", common.NewError("update_sharder_settings", err.Error())
	}

	if sn.LastSettingUpdateRound > 0 && balances.GetBlock().Round-sn.LastSettingUpdateRound < gn.CooldownPeriod {
		return "", common.NewError("update_miner_settings", "block round is in cooldown period")
	}

	if sn.Delete {
		return "", common.NewError("update_settings", "can't update settings of sharder being deleted")
	}
	if sn.Settings.DelegateWallet != t.ClientID {
		return "", common.NewError("update_sharder_settings", "access denied")
	}

	sn.Settings.ServiceChargeRatio = update.Settings.ServiceChargeRatio
	sn.Settings.MaxNumDelegates = update.Settings.MaxNumDelegates
	sn.Settings.MinStake = update.Settings.MinStake
	sn.Settings.MaxStake = update.Settings.MaxStake

	if err = sn.save(balances); err != nil {
		return "", common.NewErrorf("update_sharder_settings", "saving: %v", err)
	}

	if err = emitUpdateSharder(sn, balances, false); err != nil {
		return "", common.NewErrorf("update_sharder_settings", "saving(event): %v", err)
	}

	return string(sn.Encode()), nil
}

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

	if newSharder.Settings.DelegateWallet == "" {
		newSharder.Settings.DelegateWallet = newSharder.ID
	}

	newSharder.LastHealthCheck = t.CreationDate

	logging.Logger.Info("The new sharder info",
		zap.String("base URL", newSharder.N2NHost),
		zap.String("ID", newSharder.ID),
		zap.String("pkey", newSharder.PublicKey),
		zap.Any("mscID", msc.ID),
		zap.String("delegate_wallet", newSharder.Settings.DelegateWallet),
		zap.Float64("service_charge", newSharder.Settings.ServiceChargeRatio),
		zap.Int("number_of_delegates", newSharder.Settings.MaxNumDelegates),
		zap.Int64("min_stake", int64(newSharder.Settings.MinStake)),
		zap.Int64("max_stake", int64(newSharder.Settings.MaxStake)))

	logging.Logger.Info("SharderNode", zap.Any("node", newSharder))

	if newSharder.PublicKey == "" || newSharder.ID == "" {
		logging.Logger.Error("public key or ID is empty")
		return "", common.NewError("add_sharder",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	err = validateNodeSettings(newSharder, gn, "add_sharder")
	if err != nil {
		return "", common.NewErrorf("add_sharder", "validate node setting failed: %v", zap.Error(err))
	}

	existing, err := msc.getSharderNode(newSharder.ID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("add_sharder", "unexpected error: %v", err)
	}

	// if found
	if err == nil {
		// and found in all
		if allSharders.FindNodeById(newSharder.ID) != nil {
			logging.Logger.Info("add_sharder: found node by id")
			return string(newSharder.Encode()), nil
		}
		// otherwise the sharder has saved by block sharders reward
		newSharder.Reward = existing.Reward
	}

	newSharder.NodeType = NodeTypeSharder // set node type

	if err = quickFixDuplicateHosts(newSharder, allSharders.Nodes); err != nil {
		return "", common.NewError("add_sharder", err.Error())
	}

	allSharders.Nodes = append(allSharders.Nodes, newSharder)

	// save the added sharder
	_, err = balances.InsertTrieNode(newSharder.GetKey(), newSharder)
	if err != nil {
		return "", common.NewErrorf("add_sharder", "saving sharder: %v", err)
	}

	err = emitAddSharder(newSharder, balances)
	if err != nil {
		return "", common.NewErrorf("add_sharder", "saving sharder(event): %v", err)
	}

	// save all sharders list
	if err = updateAllShardersList(balances, allSharders); err != nil {
		return "", common.NewErrorf("add_sharder", "saving all sharders list: %v", err)
	}

	msc.verifyMinerState(balances, "checking all sharders list after insert")

	return string(newSharder.Encode()), nil
}

// DeleteSharder Function to handle removing a sharder from the chain
func (msc *MinerSmartContract) DeleteSharder(
	_ *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var err error
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

	updatedSn, err := msc.deleteNode(gn, sn, balances)
	if err != nil {
		return "", common.NewError("delete_sharder", err.Error())
	}

	if err = msc.deleteSharderFromViewChange(updatedSn, balances); err != nil {
		return "", common.NewError("delete_sharder", err.Error())
	}

	return "", nil
}

func (msc *MinerSmartContract) deleteSharderFromViewChange(sn *MinerNode, balances cstate.StateContextI) error {
	pn, err := GetPhaseNode(balances)
	if err != nil {
		return err
	}

	if pn.Phase == Unknown {
		return common.NewError("failed to delete from view change", "phase is unknown")
	}

	if pn.Phase == Wait {
		return common.NewError("failed to delete from view change", "magic block has already been created for next view change")
	}

	sharders, err := getShardersKeepList(balances)
	if err != nil {
		logging.Logger.Error("delete_sharder_from_view_change: Error in getting list from the DB",
			zap.Error(err))
		return common.NewErrorf("delete_sharder_from_view_change",
			"failed to get sharders list: %v", err)
	}
	for i, v := range sharders.Nodes {
		if v.ID == sn.ID {
			sharders.Nodes = append(sharders.Nodes[:i], sharders.Nodes[i+1:]...)

			if err = emitDeleteSharder(sn.ID, balances); err != nil {
				return err
			}
			break
		}
	}

	_, err = balances.InsertTrieNode(ShardersKeepKey, sharders)
	return err
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
	balances cstate.StateContextI) (*MinerNode, error) {

	sn := NewMinerNode()
	sn.ID = sid
	err := balances.GetTrieNode(sn.GetKey(), sn)
	if err != nil {
		return nil, err
	}

	return sn, nil
}

func (msc *MinerSmartContract) sharderKeep(_ *transaction.Transaction,
	input []byte, _ *GlobalNode, balances cstate.StateContextI) (
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
		zap.Any("mscID", msc.ID),
		zap.Int64("pn_start_round", pn.StartRound),
		zap.String("phase", pn.Phase.String()))

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
