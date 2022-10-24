package minersc

import (
	"errors"
	"fmt"

	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

// GetNodeKey converts node ID to node Key
func GetNodeKey(id string) datastore.Key {
	return ADDRESS + id
}

// AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddMiner(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {
	newMiner := NewMinerNode()
	if err = newMiner.Decode(inputData); err != nil {
		return "", common.NewErrorf("add_miner",
			"decoding request: %v", err)
	}

	newMiner.NodeType = NodeTypeMiner // set node type
	newMiner.LastHealthCheck = t.CreationDate

	if err = newMiner.Validate(); err != nil {
		return "", common.NewErrorf("add_miner", "invalid input: %v", err)
	}

	if newMiner.PublicKey == "" || newMiner.ID == "" {
		logging.Logger.Error("public key or ID is empty")
		return "", common.NewError("add_miner",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	if newMiner.Settings.DelegateWallet == "" {
		newMiner.Settings.DelegateWallet = newMiner.ID
	}

	err = validateNodeSettings(newMiner, gn, "add_miner")
	if err != nil {
		return "", err
	}

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

	// TODO: save host:port to separte MPT nodes for duplication checking
	//if err = quickFixDuplicateHosts(newMiner, allMiners.Nodes); err != nil {
	//	return "", common.NewError("add_miner", err.Error())
	//}

	if err := minersPartitions.add(balances, newMiner); err != nil {
		if partitions.ErrItemExist(err) {
			logging.Logger.Debug("add_miner - miner already exist", zap.String("ID", newMiner.ID))
			// return new miner encoded string to make it align with the old logic
			return string(newMiner.Encode()), nil
		}

		return "", common.NewError("add_miner", err.Error())
	}

	err = emitAddOrOverwriteMiner(newMiner, balances)
	if err != nil {
		return "", common.NewErrorf("add_miner",
			"insert new miner: %v", err)
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

	if err := minersPartitions.updateNode(balances, GetNodeKey(deleteMiner.ID), func(mn *MinerNode) error {
		updatedMn, err := msc.deleteNode(mn, balances)
		if err != nil {
			return err
		}

		if err = msc.deleteMinerFromViewChange(updatedMn, balances); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", common.NewErrorf("delete_miner", err.Error())
	}

	return "", nil
}

func (msc *MinerSmartContract) deleteNode(
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

	update := NewMinerNode()
	if err = update.Decode(inputData); err != nil {
		return "", common.NewErrorf("update_miner_settings",
			"decoding request: %v", err)
	}

	err = validateNodeSettings(update, gn, "update_miner_settings")
	if err != nil {
		return "", err
	}

	if err := minersPartitions.updateNode(balances, GetNodeKey(update.ID), func(mn *MinerNode) error {
		if mn.LastSettingUpdateRound > 0 && balances.GetBlock().Round-mn.LastSettingUpdateRound < gn.CooldownPeriod {
			return errors.New("block round is in cool down period")
		}

		if mn.Delete {
			return errors.New("can't update settings of miner being deleted")
		}

		if mn.Settings.DelegateWallet != t.ClientID {
			logging.Logger.Debug("delegate wallet is not equal to one set in config",
				zap.String("delegate", t.ClientID),
				zap.String("config", mn.Settings.DelegateWallet))
			return errors.New("access denied")
		}

		mn.Settings.ServiceChargeRatio = update.Settings.ServiceChargeRatio
		mn.Settings.MaxNumDelegates = update.Settings.MaxNumDelegates
		mn.Settings.MinStake = update.Settings.MinStake
		mn.Settings.MaxStake = update.Settings.MaxStake

		emitUpdateMiner(mn, balances, false)
		resp = string(mn.Encode())
		return nil
	}); err != nil {
		return "", common.NewErrorf("update_miner_settings", err.Error())
	}

	return resp, nil
}

// ------------- local functions ---------------------
// TODO: remove this or return error and do real checking
//func (msc *MinerSmartContract) verifyMinerState(allMinersList *MinerNodes, balances cstate.StateContextI,
//	msg string) {
//	if allMinersList == nil || len(allMinersList.Nodes) == 0 {
//		logging.Logger.Info(msg + " allminerslist is empty")
//		return
//	}
//}

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
