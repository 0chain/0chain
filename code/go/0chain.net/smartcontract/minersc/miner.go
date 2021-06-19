package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) doesMinerExist(pkey datastore.Key,
	balances cstate.StateContextI) bool {

	mbits, err := balances.GetTrieNode(pkey)
	if err != nil && err != util.ErrValueNotPresent {
		logging.Logger.Error("GetTrieNode from state context", zap.Error(err),
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

	var newMiner = NewMinerNode()
	if err = newMiner.Decode(inputData); err != nil {
		return "", common.NewErrorf("add_miner",
			"decoding request: %v", err)
	}

	if err = newMiner.Validate(); err != nil {
		return "", common.NewErrorf("add_miner", "invalid input: %v", err)
	}

	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()

	logging.Logger.Info("add_miner: try to add miner", zap.Any("txn", t))

	allMiners, err := getMinersList(balances)
	if err != nil {
		logging.Logger.Error("add_miner: Error in getting list from the DB",
			zap.Error(err))
		return "", common.NewErrorf("add_miner",
			"failed to get miner list: %v", err)
	}

	msc.verifyMinerState(balances,
		"add_miner: checking all miners list in the beginning")

	if newMiner.DelegateWallet == "" {
		newMiner.DelegateWallet = newMiner.ID
	}

	newMiner.LastHealthCheck = t.CreationDate

	logging.Logger.Info("add_miner: The new miner info",
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
	logging.Logger.Info("add_miner: MinerNode", zap.Any("node", newMiner))

	if newMiner.PublicKey == "" || newMiner.ID == "" {
		logging.Logger.Error("add_miner: public key or ID is empty")
		return "", common.NewError("add_miner",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	if newMiner.ServiceCharge < 0 {
		return "", common.NewErrorf("add_miner",
			"invalid negative service charge: %v", newMiner.ServiceCharge)
	}

	if newMiner.ServiceCharge > gn.MaxCharge {
		return "", common.NewErrorf("add_miner",
			"max_charge is greater than allowed by SC: %v > %v",
			newMiner.ServiceCharge, gn.MaxCharge)
	}

	if newMiner.NumberOfDelegates < 0 {
		return "", common.NewErrorf("add_miner",
			"invalid negative number_of_delegates: %v", newMiner.ServiceCharge)
	}

	if newMiner.NumberOfDelegates > gn.MaxDelegates {
		return "", common.NewErrorf("add_miner",
			"number_of_delegates greater than max_delegates of SC: %v > %v",
			newMiner.ServiceCharge, gn.MaxDelegates)
	}

	if newMiner.MinStake < gn.MinStake {
		return "", common.NewErrorf("add_miner",
			"min_stake is less than allowed by SC: %v > %v",
			newMiner.MinStake, gn.MinStake)
	}

	if newMiner.MaxStake < gn.MaxStake {
		return "", common.NewErrorf("add_miner",
			"max_stake is greater than allowed by SC: %v > %v",
			newMiner.MaxStake, gn.MaxStake)
	}

	newMiner.NodeType = NodeTypeMiner // set node type

	if err = quickFixDuplicateHosts(newMiner, allMiners.Nodes); err != nil {
		return "", common.NewError("add_miner", err.Error())
	}

	allMap := make(map[string]struct{}, len(allMiners.Nodes))
	for _, n := range allMiners.Nodes {
		allMap[n.getKey()] = struct{}{}
	}

	var update bool
	if _, ok := allMap[newMiner.getKey()]; !ok {
		allMiners.Nodes = append(allMiners.Nodes, newMiner)

		if err = updateMinersList(balances, allMiners); err != nil {
			return "", common.NewErrorf("add_miner",
				"saving all miners list: %v", err)
		}
		update = true
	}

	if !msc.doesMinerExist(newMiner.getKey(), balances) {
		if err = newMiner.save(balances); err != nil {
			return "", common.NewError("add_miner", err.Error())
		}

		msc.verifyMinerState(balances, "add_miner: Checking all miners list afterInsert")

		update = true
	}

	if !update {
		logging.Logger.Debug("Add miner already exists", zap.String("ID", newMiner.ID))
	}

	return string(newMiner.Encode()), nil
}

func (msc *MinerSmartContract) UpdateSettings(t *transaction.Transaction,
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
		return "", common.NewErrorf("add_miner",
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

	var mn *MinerNode
	mn, err = getMinerNode(update.ID, balances)
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

//------------- local functions ---------------------
func (msc *MinerSmartContract) verifyMinerState(balances cstate.StateContextI,
	msg string) {

	allMinersList, err := getMinersList(balances)
	if err != nil {
		logging.Logger.Info(msg + " (verifyMinerState) getMinersList_failed - " +
			"Failed to retrieve existing miners list: " + err.Error())
		return
	}
	if allMinersList == nil || len(allMinersList.Nodes) == 0 {
		logging.Logger.Info(msg + " allminerslist is empty")
		return
	}

	logging.Logger.Info(msg)
	for _, miner := range allMinersList.Nodes {
		logging.Logger.Info("allminerslist",
			zap.String("url", miner.N2NHost),
			zap.String("ID", miner.ID))
	}

}

func (msc *MinerSmartContract) GetMinersList(balances cstate.StateContextI) (
	all *MinerNodes, err error) {

	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()
	return getMinersList(balances)
}

// getMinerNode
func getMinerNode(id string, state cstate.StateContextI) (*MinerNode, error) {
	getFromNodeFunc := func() (*MinerNode, error) {
		mn := NewMinerNode()
		mn.ID = id

		ms, err := state.GetTrieNode(mn.getKey())
		if err != nil {
			return nil, err
		}

		if err := mn.Decode(ms.Encode()); err != nil {
			return nil, err
		}

		return mn, nil
	}

	getFromMinersList := func() (*MinerNode, error) {
		allMiners, err := getMinersList(state)
		if err != nil {
			return nil, err
		}

		for _, node := range allMiners.Nodes {
			if node.ID == id {
				return node, nil
			}
		}

		return nil, util.ErrValueNotPresent
	}

	getFuncs := []func() (*MinerNode, error){
		getFromNodeFunc,
		getFromMinersList,
	}

	var err error
	var mn *MinerNode
	for _, fn := range getFuncs {
		var node *MinerNode
		node, err = fn()
		if err == nil {
			return node, nil
		}

		switch err {
		case util.ErrNodeNotFound, util.ErrValueNotPresent:
			mn = NewMinerNode()
			mn.ID = id
			continue
		default:
			return nil, err
		}
	}

	return mn, err
}
