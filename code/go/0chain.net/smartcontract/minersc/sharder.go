package minersc

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"

	commonsc "0chain.net/smartcontract/common"
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

	var sn *MinerNode
	sn, err = msc.getSharderNode(update.ID, balances)
	if err != nil {
		return "", common.NewError("update_sharder_settings", err.Error())
	}

	if sn.LastSettingUpdateRound > 0 && balances.GetBlock().Round-sn.LastSettingUpdateRound < gn.CooldownPeriod {
		return "", common.NewError("update_sharder_settings", "block round is in cooldown period")
	}

	if sn.Delete {
		return "", common.NewError("update_sharder_settings", "can't update settings of sharder being deleted")
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

// AddSharder function to handle sharder register
func (msc *MinerSmartContract) AddSharder(
	t *transaction.Transaction,
	input []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {

	var newSharder = NewMinerNode()
	if err = newSharder.Decode(input); err != nil {
		logging.Logger.Error("Error in decoding the input", zap.Error(err))
		return "", common.NewErrorf("add_sharder", "decoding request: %v", err)
	}

	if err = newSharder.Validate(); err != nil {
		return "", common.NewErrorf("add_sharder", "invalid input: %v", err)
	}

	magicBlockSharders := balances.GetChainCurrentMagicBlock().Sharders
	if !magicBlockSharders.HasNode(newSharder.ID) {
		logging.Logger.Error("add_sharder: Error in Adding a new sharder: Not in magic block", zap.String("SharderID", newSharder.ID))
		return "", common.NewErrorf("add_sharder",
			"failed to add new sharder: Not in magic block")
	}

	if config.Development() && newSharder.Settings.DelegateWallet == "" {
		newSharder.Settings.DelegateWallet = newSharder.ID
	}

	newSharder.LastHealthCheck = t.CreationDate

	logging.Logger.Info("The new sharder info",
		zap.String("base URL", newSharder.N2NHost),
		zap.String("ID", newSharder.ID),
		zap.String("pkey", newSharder.PublicKey),
		zap.String("mscID", msc.ID),
		zap.String("delegate_wallet", newSharder.Settings.DelegateWallet),
		zap.Float64("service_charge", newSharder.Settings.ServiceChargeRatio),
		zap.Int("number_of_delegates", newSharder.Settings.MaxNumDelegates),
		zap.Int64("min_stake", int64(newSharder.Settings.MinStake)),
		zap.Int64("max_stake", int64(newSharder.Settings.MaxStake)))

	if newSharder.PublicKey == "" || newSharder.ID == "" {
		logging.Logger.Error("public key or ID is empty")
		return "", common.NewError("add_sharder",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	// Check delegate wallet differs from operationl wallet
	if err := commonsc.ValidateDelegateWallet(newSharder.PublicKey, newSharder.Settings.DelegateWallet); err != nil {
		return "", err
	}

	err = validateNodeSettings(newSharder, gn, "add_sharder")
	if err != nil {
		return "", common.NewErrorf("add_sharder", "validate node setting failed: %v", zap.Error(err))
	}

	newSharder.NodeType = NodeTypeSharder // set node type
	newSharder.ProviderType = spenum.Sharder
	exist, err := msc.getSharderNode(newSharder.ID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("add_sharder", "unexpected error: %v", err)
	}

	if exist != nil {
		logging.Logger.Info("add_sharder: sharder already exist", zap.String("ID", newSharder.ID))
		return string(newSharder.Encode()), nil
	}

	if err = insertNodeN2NHost(balances, ADDRESS, newSharder); err != nil {
		return "", common.NewError("add_sharder", err.Error())
	}

	nodeIDs, err := getNodeIDs(balances, AllShardersKey)
	if err != nil {
		return "", common.NewErrorf("add_sharder", "could not get sharder ids: %v", err)
	}

	nodeIDs = append(nodeIDs, newSharder.ID)
	if err := nodeIDs.save(balances, AllShardersKey); err != nil {
		return "", common.NewErrorf("add_sharder", "save harder to list failed: %v", err)
	}

	if err := newSharder.save(balances); err != nil {
		return "", common.NewErrorf("add_sharder", "save sharder failed: %v", err)
	}

	emitAddSharder(newSharder, balances)
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

	sharderIDs, err := getNodeIDs(balances, ShardersKeepKey)
	if err != nil {
		logging.Logger.Error("delete_sharder_from_view_change: Error in getting list from the DB", zap.Error(err))
		return common.NewErrorf("delete_sharder_from_view_change", "failed to get sharder ids: %v", err)
	}

	for i, id := range sharderIDs {
		if id == sn.ID {
			sharderIDs = append(sharderIDs[:i], sharderIDs[i+1:]...)

			emitDeleteSharder(sn.ID, balances)
			break
		}
	}

	return sharderIDs.save(balances, ShardersKeepKey)
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

func (_ *MinerSmartContract) getSharderNode(
	sid string,
	balances cstate.StateContextI,
) (*MinerNode, error) {
	return getSharderNode(sid, balances)
}

func getSharderNode(
	sid string,
	balances cstate.CommonStateContextI,
) (*MinerNode, error) {
	sn := NewMinerNode()
	sn.ID = sid
	err := balances.GetTrieNode(sn.GetKey(), sn)
	if err != nil {
		return nil, err
	}
	if sn.ProviderType != spenum.Sharder {
		return nil, fmt.Errorf("provider is %s should be %s", sn.ProviderType, spenum.Blobber)
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

	logging.Logger.Info("The new sharder info",
		zap.String("base URL", newSharder.N2NHost),
		zap.String("ID", newSharder.ID),
		zap.String("pkey", newSharder.PublicKey),
		zap.String("mscID", msc.ID),
		zap.Int64("pn_start_round", pn.StartRound),
		zap.String("phase", pn.Phase.String()))

	if newSharder.PublicKey == "" || newSharder.ID == "" {
		logging.Logger.Error("public key or ID is empty")
		return "", errors.New("PublicKey or the ID is empty. Cannot proceed")
	}

	//check new sharder
	_, err = getSharderNode(newSharder.ID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		return "", common.NewErrorf("sharder_keep", "unknown sharder: %v", newSharder.ID)
	default:
		return "", common.NewErrorf("sharder_keep", "failed to check sharder existence: %v", err)
	}

	keepNodeIDs, err := getNodeIDs(balances, ShardersKeepKey)
	if err != nil {
		return "", common.NewErrorf("sharder_keep",
			"failed to get keep sharder ids: %v", err)
	}

	if keepNodeIDs.find(newSharder.ID) {
		// do not return error for sharder already exist,
		logging.Logger.Debug("Add sharder already exists", zap.String("ID", newSharder.ID))
		return string(newSharder.Encode()), nil
	}

	keepNodeIDs = append(keepNodeIDs, newSharder.ID)
	if err := keepNodeIDs.save(balances, ShardersKeepKey); err != nil {
		return "", common.NewErrorf("sharder_keep",
			"failed to save keep sharder ids: %v", err)
	}

	buff := newSharder.Encode()
	return string(buff), nil
}
