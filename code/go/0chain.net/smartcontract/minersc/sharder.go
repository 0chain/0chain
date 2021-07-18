package minersc

import (
	"github.com/0chain/gosdk/core/common/errors"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
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
		return "", errors.Newf("add_sharder", "decoding request: %v", err)
	}

	if err = newSharder.Validate(); err != nil {
		return "", errors.Newf("add_sharder", "invalid input: %v", err)
	}

	allSharders, err := getAllShardersList(balances)
	if err != nil {
		logging.Logger.Error("add_sharder: failed to get sharders list", zap.Error(err))
		return "", errors.Newf("add_sharder", "getting all sharders list: %v", err)
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
		return "", errors.New("add_sharder",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	if newSharder.NumberOfDelegates < 0 {
		return "", errors.Newf("add_sharder",
			"invalid negative number_of_delegates: %v",
			newSharder.ServiceCharge)
	}

	if newSharder.MinStake < gn.MinStake {
		return "", errors.Newf("add_sharder",
			"min_stake is less than allowed by SC: %v > %v",
			newSharder.MinStake, gn.MinStake)
	}

	if newSharder.MaxStake < gn.MaxStake {
		return "", errors.Newf("add_sharder",
			"max_stake is greater than allowed by SC: %v > %v",
			newSharder.MaxStake, gn.MaxStake)
	}

	existing, err := msc.getSharderNode(newSharder.ID, balances)
	if err != nil && err != util.ErrValueNotPresent() {
		return "", errors.Newf("add_sharder", "unexpected error: %v", err)
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
		return "", errors.Wrap(err, errors.New("add_sharder", ""))
	}

	allSharders.Nodes = append(allSharders.Nodes, newSharder)

	// save the added sharder
	_, err = balances.InsertTrieNode(newSharder.getKey(), newSharder)
	if err != nil {
		return "", errors.Newf("add_sharder", "saving sharder: %v", err)
	}

	// save all sharders list
	if err = updateAllShardersList(balances, allSharders); err != nil {
		return "", errors.Newf("add_sharder", "saving all sharders list: %v", err)
	}

	msc.verifyMinerState(balances, "checking all sharders list after insert")

	return string(newSharder.Encode()), nil
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
	if err != nil && err != util.ErrValueNotPresent() {
		return // unexpected error
	}

	sn = NewMinerNode()
	sn.ID = sid

	if err == util.ErrValueNotPresent() {
		return // with error ErrValueNotPresent (that's very stupid)
	}

	if err = sn.Decode(ss.Encode()); err != nil {
		return nil, errors.Newf("", "invalid state: decoding sharder: %v", err)
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
		return "", errors.New("sharder_keep",
			"this is not the correct phase to contribute (sharder keep)")
	}

	newSharder := NewMinerNode()
	err = newSharder.Decode(input)
	if err != nil {
		logging.Logger.Error("Error in decoding the input", zap.Error(err))
		return "", err
	}

	if err = newSharder.Validate(); err != nil {
		return "", errors.Newf("sharder_keep", "invalid input: %v", err)
	}

	sharderKeepList, err := getShardersKeepList(balances)
	if err != nil {
		logging.Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.Newf("sharder_keep",
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
		return "", errors.Newf("sharder_keep",
			"Failed to get miner list: %v", err)
	}
	if allShardersList.FindNodeById(newSharder.ID) == nil {
		return "", errors.Newf("sharder_keep", "unknown sharder: %v", newSharder.ID)
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
