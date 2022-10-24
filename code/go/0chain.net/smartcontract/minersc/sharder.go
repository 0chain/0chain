package minersc

import (
	"errors"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/partitions"

	"github.com/0chain/common/core/logging"
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

	if err := shardersPartitions.update(balances, update.ID, func(sn *MinerNode) error {
		if sn.LastSettingUpdateRound > 0 && balances.GetBlock().Round-sn.LastSettingUpdateRound < gn.CooldownPeriod {
			return errors.New("block round is in cooldown period")
		}

		if sn.Delete {
			return errors.New("can't update settings of sharder being deleted")
		}
		if sn.Settings.DelegateWallet != t.ClientID {
			return errors.New("access denied")
		}

		sn.Settings.ServiceChargeRatio = update.Settings.ServiceChargeRatio
		sn.Settings.MaxNumDelegates = update.Settings.MaxNumDelegates
		sn.Settings.MinStake = update.Settings.MinStake
		sn.Settings.MaxStake = update.Settings.MaxStake

		emitUpdateSharder(sn, balances, false)

		resp = string(sn.Encode())
		return nil
	}); err != nil {
		return "", common.NewErrorf("update_sharder_settings", err.Error())
	}

	return resp, nil
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

	newSharder.LastHealthCheck = t.CreationDate
	newSharder.NodeType = NodeTypeSharder // set node type

	if newSharder.Settings.DelegateWallet == "" {
		newSharder.Settings.DelegateWallet = newSharder.ID
	}

	if newSharder.PublicKey == "" || newSharder.ID == "" {
		logging.Logger.Error("public key or ID is empty")
		return "", common.NewError("add_sharder",
			"PublicKey or the ID is empty. Cannot proceed")
	}

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

	err = validateNodeSettings(newSharder, gn, "add_sharder")
	if err != nil {
		return "", common.NewErrorf("add_sharder", "validate node setting failed: %v", err)
	}

	// TODO: save host:port for duplication checking
	//if err = quickFixDuplicateHosts(newSharder, allSharders.Nodes); err != nil {
	//	return "", common.NewError("add_sharder", err.Error())
	//}

	if err := shardersPartitions.add(balances, newSharder); err != nil {
		if partitions.ErrItemExist(err) {
			logging.Logger.Info("add_sharder: sharder already exist", zap.String("ID", newSharder.ID))
			return string(newSharder.Encode()), nil
		}
		return "", common.NewErrorf("add_sharder", "adding sharder failed: %v", err)
	}

	emitAddOrOverwriteSharder(newSharder, balances)

	return string(newSharder.Encode()), nil
}

// DeleteSharder Function to handle removing a sharder from the chain
func (msc *MinerSmartContract) DeleteSharder(
	_ *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var deleteSharder = NewMinerNode()
	if err := deleteSharder.Decode(inputData); err != nil {
		return "", common.NewErrorf("delete_sharder",
			"decoding request: %v", err)
	}

	if err := shardersPartitions.update(balances, deleteSharder.ID, func(sn *MinerNode) error {
		updatedSn, err := msc.deleteNode(sn, balances)
		if err != nil {
			return err
		}

		if err = msc.deleteSharderFromViewChange(updatedSn, balances); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", common.NewErrorf("delete_sharder", err.Error())
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
			"failed to get sharders keep list: %v", err)
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

// ------------- local functions ---------------------
//func verifyAllShardersState(balances cstate.StateContextI, msg string) {
//	shardersList, err := getAllShardersList(balances)
//	if err != nil {
//		logging.Logger.Error("verify_all_sharder_state_failed", zap.Error(err))
//		return
//	}
//
//	if shardersList == nil || len(shardersList.Nodes) == 0 {
//		logging.Logger.Info(msg + " shardersList is empty")
//		return
//	}
//
//	logging.Logger.Info(msg)
//	for _, sharder := range shardersList.Nodes {
//		logging.Logger.Info("shardersList", zap.String("url", sharder.N2NHost), zap.String("ID", sharder.ID))
//	}
//}

//func verifyShardersKeepState(balances cstate.StateContextI, msg string) {
//	shardersList, err := getShardersKeepList(balances)
//	if err != nil {
//		logging.Logger.Error("verify_sharder_keep_state_failed", zap.Error(err))
//		return
//	}
//
//	if shardersList == nil || len(shardersList.Nodes) == 0 {
//		logging.Logger.Info(msg + " shardersList is empty")
//		return
//	}
//
//	logging.Logger.Info(msg)
//	for _, sharder := range shardersList.Nodes {
//		logging.Logger.Info("shardersList", zap.String("url", sharder.N2NHost), zap.String("ID", sharder.ID))
//	}
//}

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
	//verifyShardersKeepState(balances, "Checking sharderKeepList in the beginning")

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
	found, err := shardersPartitions.exist(balances, newSharder.GetKey())
	if err != nil {
		return "", common.NewErrorf("sharder_keep", "failed to get sharder: %v, %v", newSharder.ID, err)
	}

	if !found {
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

	buff := newSharder.Encode()
	return string(buff), nil
}
