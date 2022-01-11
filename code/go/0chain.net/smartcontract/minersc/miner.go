package minersc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
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
		logging.Logger.Error("public key or ID is empty")
		return "", common.NewError("add_miner",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	err = validateNodeSettings(newMiner, gn, "add_miner")
	if err != nil {
		return "", err
	}

	newMiner.NodeType = NodeTypeMiner // set node type

	if err = quickFixDuplicateHosts(newMiner, allMiners.Nodes); err != nil {
		return "", common.NewError("add_miner", err.Error())
	}

	allMap := make(map[string]struct{}, len(allMiners.Nodes))
	for _, n := range allMiners.Nodes {
		allMap[n.GetKey()] = struct{}{}
	}

	var update bool
	if _, ok := allMap[newMiner.GetKey()]; !ok {
		allMiners.Nodes = append(allMiners.Nodes, newMiner)

		if err = updateMinersList(balances, allMiners); err != nil {
			return "", common.NewErrorf("add_miner",
				"saving all miners list: %v", err)
		}
		update = true
	}

	if !msc.doesMinerExist(newMiner.GetKey(), balances) {
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

// deleteMiner Function to handle removing a miner from the chain
func (msc *MinerSmartContract) DeleteMiner(
	_ *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	var err error
	var deleteMiner = NewMinerNode()
	if err = deleteMiner.Decode(inputData); err != nil {
		return "", common.NewErrorf("delete_miner",
			"decoding request: %v", err)
	}

	var mn *MinerNode
	mn, err = getMinerNode(deleteMiner.ID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		mn = NewMinerNode()
		mn.ID = deleteMiner.ID
	default:
		return "", common.NewError("delete_miner", err.Error())
	}

	updatedMn, err := msc.deleteNode(gn, mn, balances)
	if err != nil {
		return "", common.NewError("delete_miner", err.Error())
	}

	if err = msc.deleteMinerFromViewChange(updatedMn, balances); err != nil {
		return "", common.NewError("delete_miner", err.Error())
	}

	return "", nil
}

func (msc *MinerSmartContract) deleteNode(
	gn *GlobalNode,
	deleteNode *MinerNode,
	balances cstate.StateContextI,
) (*MinerNode, error) {
	var err error
	deleteNode.Delete = true

	// deleting pending pools
	for key, pool := range deleteNode.Pending {
		var un *UserNode
		if un, err = msc.getUserNode(pool.DelegateID, balances); err != nil {
			return nil, fmt.Errorf("getting user node: %v", err)
		}

		var transfer *state.Transfer
		transfer, _, err = pool.EmptyPool(msc.ID, pool.DelegateID, nil)
		if err != nil {
			return nil, fmt.Errorf("error emptying delegate pool: %v", err)
		}

		if err = balances.AddTransfer(transfer); err != nil {
			return nil, fmt.Errorf("adding transfer: %v", err)
		}

		if err := un.deletePool(deleteNode.ID, key); err != nil {
			return nil, fmt.Errorf("deleting pool: %v", err)
		}
		delete(deleteNode.Pending, key)

		if err = un.save(balances); err != nil {
			return nil, fmt.Errorf("saving user node%s: %v", un.ID, err)
		}
	}

	// deleting active pools
	for key, pool := range deleteNode.Active {
		if pool.Status == DELETING {
			continue
		}

		pool.Status = DELETING // mark as deleting
		pool.TokenLockInterface = &ViewChangeLock{
			Owner:               pool.DelegateID,
			DeleteViewChangeSet: true,
			DeleteVC:            gn.ViewChange,
		}
		deleteNode.Deleting[key] = pool // add to deleting
	}

	// set node type -- miner
	if err = deleteNode.save(balances); err != nil {
		return nil, fmt.Errorf("saving node %v", err.Error())
	}

	return deleteNode, nil
}

func (msc *MinerSmartContract) deleteMinerFromViewChange(mn *MinerNode, balances cstate.StateContextI) (err error) {
	var pn *PhaseNode
	if pn, err = GetPhaseNode(balances); err != nil {
		return
	}
	if pn.Phase == Unknown {
		err = common.NewError("failed to delete from view change", "phase is unknown")
		return
	}
	if pn.Phase != Wait {
		var dkgMiners *DKGMinerNodes
		if dkgMiners, err = getDKGMinersList(balances); err != nil {
			return
		}
		if _, ok := dkgMiners.SimpleNodes[mn.ID]; ok {
			delete(dkgMiners.SimpleNodes, mn.ID)
			_, err = balances.InsertTrieNode(DKGMinersKey, dkgMiners)
		}
	} else {
		err = common.NewError("failed to delete from view change", "magic block has already been created for next view change")
		return
	}
	return
}

func (msc *MinerSmartContract) UpdateMinerSettings(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var update = NewMinerNode()
	if err = update.Decode(inputData); err != nil {
		return "", common.NewErrorf("update_miner_settings",
			"decoding request: %v", err)
	}

	err = validateNodeSettings(update, gn, "update_miner_settings")
	if err != nil {
		return "", err
	}

	var mn *MinerNode
	mn, err = getMinerNode(update.ID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		mn = NewMinerNode()
		mn.ID = update.ID
	default:
		return "", common.NewError("update_miner_settings", err.Error())
	}

	if mn.Delete {
		return "", common.NewError("update_settings", "can't update settings of miner being deleted")
	}

	if mn.DelegateWallet != t.ClientID {
		return "", common.NewError("update_miner_settings", "access denied")
	}

	mn.ServiceCharge = update.ServiceCharge
	mn.NumberOfDelegates = update.NumberOfDelegates
	mn.MinStake = update.MinStake
	mn.MaxStake = update.MaxStake

	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("update_miner_settings", "saving: %v", err)
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
}

func (msc *MinerSmartContract) GetMinersList(balances cstate.StateContextI) (
	all *MinerNodes, err error) {

	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()
	return getMinersList(balances)
}

// getMinerNode
func getMinerNode(id string, state cstate.StateContextI) (*MinerNode, error) {
	mn := NewMinerNode()
	mn.ID = id
	ms, err := state.GetTrieNode(mn.GetKey())
	if err != nil {
		return nil, err
	}

	if err := mn.Decode(ms.Encode()); err != nil {
		return nil, err
	}

	return mn, nil
}

func validateNodeSettings(node *MinerNode, gn *GlobalNode, opcode string) error {
	if node.ServiceCharge < 0 {
		return common.NewErrorf(opcode,
			"invalid negative service charge: %v", node.ServiceCharge)
	}

	if node.ServiceCharge > gn.MaxCharge {
		return common.NewErrorf(opcode,
			"max_charge is greater than allowed by SC: %v > %v",
			node.ServiceCharge, gn.MaxCharge)
	}

	if node.NumberOfDelegates <= 0 {
		return common.NewErrorf(opcode,
			"invalid non-positive number_of_delegates: %v", node.NumberOfDelegates)
	}

	if node.NumberOfDelegates > gn.MaxDelegates {
		return common.NewErrorf(opcode,
			"number_of_delegates greater than max_delegates of SC: %v > %v",
			node.NumberOfDelegates, gn.MaxDelegates)
	}

	if node.MinStake < gn.MinStake {
		return common.NewErrorf(opcode,
			"min_stake is less than allowed by SC: %v > %v",
			node.MinStake, gn.MinStake)
	}

	if node.MinStake < 0 || node.MaxStake < 0 {
		return common.NewErrorf(opcode,
			"invalid negative min_stake: %v or max_stake: %v", node.MinStake, node.MaxStake)
	}

	if node.MinStake > node.MaxStake {
		return common.NewErrorf(opcode,
			"invalid node request results in min_stake greater than max_stake: %v > %v", node.MinStake, node.MaxStake)
	}

	if node.MaxStake > gn.MaxStake {
		return common.NewErrorf(opcode,
			"max_stake is greater than allowed by SC: %v > %v",
			node.MaxStake, gn.MaxStake)
	}

	return nil
}
