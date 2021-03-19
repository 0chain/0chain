package minersc

import (
	"errors"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) doesMinerExist(pkey datastore.Key,
	balances cstate.StateContextI) bool {

	mbits, err := balances.GetTrieNode(pkey)
	if err != nil && err != util.ErrValueNotPresent {
		Logger.Error("GetTrieNode from state context", zap.Error(err),
			zap.String("key", pkey))
		return false
	}
	if mbits != nil {
		return true
	}
	return false
}

// AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddMiner(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
		resp string, err error) {

	var newMiner = NewConsensusNode()
	if err = newMiner.Decode(inputData); err != nil {
		return "", common.NewErrorf("add_miner_failed",
			"decoding request: %v", err)
	}

	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()

	Logger.Info("add_miner: try to add miner", zap.Any("txn", t))

	var all *ConsensusNodes
	if all, err = msc.getMinersList(balances); err != nil {
		Logger.Error("add_miner: Error in getting list from the DB",
			zap.Error(err))
		return "", common.NewErrorf("add_miner_failed",
			"failed to get miner list: %v", err)
	}
	msc.verifyMinerState(balances,
		"add_miner: checking all miners list in the beginning")

	if newMiner.DelegateWallet == "" {
		newMiner.DelegateWallet = newMiner.ID
	}

	newMiner.LastHealthCheck = t.CreationDate

	Logger.Info("add_miner: The new miner info",
		zap.String("base URL", newMiner.N2NHost),
		zap.String("ID", newMiner.ID),
		zap.String("pkey", newMiner.PublicKey),
		zap.Any("mscID", msc.ID),
		zap.String("delegate_wallet", newMiner.DelegateWallet),
		zap.Float64("service_charge", newMiner.ServiceCharge),
		zap.Int("number_of_delegates", newMiner.NumberOfDelegates),
		zap.Int64("min_stake", int64(newMiner.MinStake)),
		zap.Int64("max_stake", int64(newMiner.MaxStake)),
	)
	Logger.Info("add_miner: ConsensusNode", zap.Any("node", newMiner))

	if newMiner.PublicKey == "" || newMiner.ID == "" {
		Logger.Error("add_miner: public key or ID is empty")
		return "", common.NewError("add_miner_failed",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	if newMiner.ServiceCharge < 0 {
		return "", common.NewErrorf("add_miner_failed",
			"invalid negative service charge: %v", newMiner.ServiceCharge)
	}

	if newMiner.ServiceCharge > gn.MaxCharge {
		return "", common.NewErrorf("add_miner_failed",
			"max_charge is greater than allowed by SC: %v > %v",
			newMiner.ServiceCharge, gn.MaxCharge)
	}

	if newMiner.NumberOfDelegates < 0 {
		return "", common.NewErrorf("add_miner_failed",
			"invalid negative number_of_delegates: %v", newMiner.NumberOfDelegates)
	}

	if newMiner.NumberOfDelegates > gn.MaxDelegates {
		return "", common.NewErrorf("add_miner_failed",
			"number_of_delegates greater than max_delegates of SC: %v > %v",
			newMiner.NumberOfDelegates, gn.MaxDelegates)
	}

	if newMiner.MinStake < gn.MinStake {
		return "", common.NewErrorf("add_miner_failed",
			"min_stake is less than allowed by SC: %v < %v",
			newMiner.MinStake, gn.MinStake)
	}

	if newMiner.MaxStake > gn.MaxStake {
		return "", common.NewErrorf("add_miner_failed",
			"max_stake is greater than allowed by SC: %v > %v",
			newMiner.MaxStake, gn.MaxStake)
	}

	if msc.doesMinerExist(newMiner.getKey(), balances) {
		return "", common.NewError("add_miner_failed",
			"miner already exists")
	}

	newMiner.NodeType = NodeTypeMiner // set node type

	// add to all miners list
	all.Nodes = append(all.Nodes, newMiner.SimpleNode)
	if _, err = balances.InsertTrieNode(AllMinersKey, all); err != nil {
		return "", common.NewErrorf("add_miner_failed",
			"saving all miners list: %v", err)
	}

	// set node type -- miner
	if err = newMiner.save(balances); err != nil {
		return "", common.NewError("add_miner_failed", err.Error())
	}

	msc.verifyMinerState(balances,
		"add_miner: Checking all miners list afterInsert")

	resp = string(newMiner.Encode())
	return
}

func (msc *MinerSmartContract) UpdateSettings(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
		resp string, err error) {

	var update = NewConsensusNode()
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
			"service_charge is greater than allowed by SC: %v > %v",
			update.ServiceCharge, gn.MaxCharge)
	}

	if update.NumberOfDelegates < 0 {
		return "", common.NewErrorf("update_settings",
			"invalid negative number_of_delegates: %v", update.NumberOfDelegates)
	}

	if update.NumberOfDelegates > gn.MaxDelegates {
		return "", common.NewErrorf("add_miner_failed",
			"number_of_delegates greater than max_delegates of SC: %v > %v",
			update.NumberOfDelegates, gn.MaxDelegates)
	}

	if update.MinStake < gn.MinStake {
		return "", common.NewErrorf("update_settings",
			"min_stake is less than allowed by SC: %v < %v",
			update.MinStake, gn.MinStake)
	}

	if update.MaxStake > gn.MaxStake {
		return "", common.NewErrorf("update_settings",
			"max_stake is greater than allowed by SC: %v > %v",
			update.MaxStake, gn.MaxStake)
	}

	var mn *ConsensusNode
	mn, err = msc.getConsensusNode(update.ID, balances)
	if err != nil {
		return "", common.NewError("update_settings", err.Error())
	}

	if mn.DelegateWallet != t.ClientID {
		return "", common.NewError("update_setings", "access denied")
	}

	mn.ServiceCharge = update.ServiceCharge
	mn.NumberOfDelegates = update.NumberOfDelegates
	mn.MinStake = update.MinStake
	mn.MaxStake = update.MaxStake

	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("update_setings", "saving: %v", err)
	}

	return string(mn.Encode()), nil
}

func (msc *MinerSmartContract) GetMinersList(balances cstate.StateContextI) (
	all *ConsensusNodes, err error) {

	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()
	return msc.getMinersList(balances)
}

func (msc *MinerSmartContract) getMinersList(balances cstate.StateContextI) (
	all *ConsensusNodes, err error) {

	all = new(ConsensusNodes)
	allMinersBytes, err := balances.GetTrieNode(AllMinersKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, errors.New("get_miners_list_failed - " +
			"failed to retrieve existing miners list: " + err.Error())
	}
	if allMinersBytes == nil {
		return all, nil
	}
	all.Decode(allMinersBytes.Encode())
	return all, nil
}

func (msc *MinerSmartContract) verifyMinerState(balances cstate.StateContextI,
	msg string) {

	allMinersList, err := msc.getMinersList(balances)
	if err != nil {
		Logger.Info(msg + " (verifyMinerState) getMinersList_failed - " +
			"Failed to retrieve existing miners list: " + err.Error())
		return
	}
	if allMinersList == nil || len(allMinersList.Nodes) == 0 {
		Logger.Info(msg + " allminerslist is empty")
		return
	}

	Logger.Info(msg)
	for _, miner := range allMinersList.Nodes {
		Logger.Info("allminerslist",
			zap.String("url", miner.N2NHost),
			zap.String("ID", miner.ID))
	}
}

func (msc *MinerSmartContract) getConsensusNode(id string,
	balances cstate.StateContextI) (*ConsensusNode, error) {

	node := NewConsensusNode()
	node.ID = id

	trieNode, err := balances.GetTrieNode(node.getKey())
	if err == util.ErrValueNotPresent {
		return node, err
	} else if err != nil {
		return nil, err
	}

	if err := node.Decode(trieNode.Encode()); err != nil {
		return nil, err
	}
	return node, nil
}
