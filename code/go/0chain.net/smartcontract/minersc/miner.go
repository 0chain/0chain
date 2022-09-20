package minersc

import (
	"fmt"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

func doesMinerExist(pkey datastore.Key,
	balances cstate.CommonStateContextI) (bool, error) {

	mn := NewMinerNode()
	err := balances.GetTrieNode(pkey, mn)
	switch err {
	case nil:
		return true, nil
	case util.ErrValueNotPresent:
		return false, nil
	default:
		return false, err
	}
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

	if newMiner.Settings.DelegateWallet == "" {
		newMiner.Settings.DelegateWallet = newMiner.ID
	}

	newMiner.LastHealthCheck = t.CreationDate

	logging.Logger.Info("add_miner: The new miner info",
		zap.String("base URL", newMiner.N2NHost),
		zap.String("ID", newMiner.ID),
		zap.String("pkey", newMiner.PublicKey),
		zap.Any("mscID", msc.ID),
		zap.String("delegate_wallet", newMiner.Settings.DelegateWallet),
		zap.Float64("service_charge", newMiner.Settings.ServiceChargeRatio),
		zap.Int("number_of_delegates", newMiner.Settings.MaxNumDelegates),
		zap.Int64("min_stake", int64(newMiner.Settings.MinStake)),
		zap.Int64("max_stake", int64(newMiner.Settings.MaxStake)),
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

		err = emitAddMiner(newMiner, balances)
		if err != nil {
			return "", common.NewErrorf("add_miner",
				"insert new miner: %v", err)
		}

		update = true
	}

	exist, err := doesMinerExist(newMiner.GetKey(), balances)
	if err != nil {
		return "", common.NewErrorf("add_miner", "error checking miner existence: %v", err)
	}

	if !exist {
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
	var nodeType spenum.Provider
	switch deleteNode.NodeType {
	case NodeTypeMiner:
		nodeType = spenum.Miner
	case NodeTypeSharder:
		nodeType = spenum.Sharder
	default:
		return nil, fmt.Errorf("unrecognised node type: %v", deleteNode.NodeType.String())
	}

	usp, err := stakepool.GetUserStakePools(nodeType, deleteNode.Settings.DelegateWallet, balances)
	if err != nil {
		return nil, fmt.Errorf("can't get user pools list: %v", err)
	}

	for key, pool := range deleteNode.Pools {
		switch pool.Status {
		case spenum.Pending:
			_, err := deleteNode.UnlockPool(
				pool.DelegateID, nodeType, key, usp, balances)
			if err != nil {
				return nil, fmt.Errorf("error emptying delegate pool: %v", err)
			}
		case spenum.Active:
			pool.Status = spenum.Deleting
		case spenum.Deleting:
		case spenum.Deleted:
		default:
			return nil, fmt.Errorf(
				"unrecognised stakepool status: %v", pool.Status.String())
		}
	}

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
			if err != nil {
				return
			}

			err = emitDeleteMiner(mn.ID, balances)
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

	if mn.LastSettingUpdateRound > 0 && balances.GetBlock().Round-mn.LastSettingUpdateRound < gn.CooldownPeriod {
		return "", common.NewError("update_miner_settings", "block round is in cooldown period")
	}

	if mn.Delete {
		return "", common.NewError("update_miner_settings", "can't update settings of miner being deleted")
	}

	if mn.Settings.DelegateWallet != t.ClientID {
		logging.Logger.Debug("delegate wallet is not equal to one set in config", zap.String("delegate", t.ClientID), zap.String("config", mn.Settings.DelegateWallet))
		return "", common.NewError("update_miner_settings", "access denied")
	}

	mn.Settings.ServiceChargeRatio = update.Settings.ServiceChargeRatio
	mn.Settings.MaxNumDelegates = update.Settings.MaxNumDelegates
	mn.Settings.MinStake = update.Settings.MinStake
	mn.Settings.MaxStake = update.Settings.MaxStake

	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("update_miner_settings", "saving: %v", err)
	}

	if err = emitUpdateMiner(mn, balances, false); err != nil {
		return "", common.NewErrorf("update_miner_settings", "saving: %v", err)
	}

	return string(mn.Encode()), nil
}

//------------- local functions ---------------------
// TODO: remove this or return error and do real checking
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

func (msc *MinerSmartContract) getMinersList(balances cstate.QueryStateContextI) (
	all *MinerNodes, err error) {

	lockAllMiners.Lock()
	defer lockAllMiners.Unlock()
	return getMinersList(balances)
}

func getMinerNode(id string, state cstate.CommonStateContextI) (*MinerNode, error) {

	mn := NewMinerNode()
	mn.ID = id
	err := state.GetTrieNode(mn.GetKey(), mn)
	if err != nil {
		return nil, err
	}

	return mn, nil
}

func validateNodeSettings(node *MinerNode, gn *GlobalNode, opcode string) error {
	if node.Settings.ServiceChargeRatio < 0 {
		return common.NewErrorf(opcode,
			"invalid negative service charge: %v", node.Settings.ServiceChargeRatio)
	}

	if node.Settings.ServiceChargeRatio > gn.MaxCharge {
		return common.NewErrorf(opcode,
			"max_charge is greater than allowed by SC: %v > %v",
			node.Settings.ServiceChargeRatio, gn.MaxCharge)
	}

	if node.Settings.MaxNumDelegates <= 0 {
		return common.NewErrorf(opcode,
			"invalid non-positive number_of_delegates: %v", node.Settings.MaxNumDelegates)
	}

	if node.Settings.MaxNumDelegates > gn.MaxDelegates {
		return common.NewErrorf(opcode,
			"number_of_delegates greater than max_delegates of SC: %v > %v",
			node.Settings.MaxNumDelegates, gn.MaxDelegates)
	}

	if node.Settings.MinStake < gn.MinStake {
		return common.NewErrorf(opcode,
			"min_stake is less than allowed by SC: %v > %v",
			node.Settings.MinStake, gn.MinStake)
	}

	if node.Settings.MinStake > node.Settings.MaxStake {
		return common.NewErrorf(opcode,
			"invalid node request results in min_stake greater than max_stake: %v > %v", node.Settings.MinStake, node.Settings.MaxStake)
	}

	if node.Settings.MaxStake > gn.MaxStake {
		return common.NewErrorf(opcode,
			"max_stake is greater than allowed by SC: %v > %v",
			node.Settings.MaxStake, gn.MaxStake)
	}

	return nil
}
