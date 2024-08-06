package minersc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/dto"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	commonsc "0chain.net/smartcontract/common"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

//nolint:unused
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

	newMiner.Settings.MinStake = gn.MinStakePerDelegate
	magicBlockMiners := balances.GetChainCurrentMagicBlock().Miners

	if magicBlockMiners == nil {
		return "", common.NewError("add_miner", "magic block miners nil")
	}

	if !magicBlockMiners.HasNode(newMiner.ID) {

		logging.Logger.Error("add_miner: Error in Adding a new miner: Not in magic block")
		return "", common.NewErrorf("add_miner",
			"failed to add new miner: Not in magic block")
	}

	newMiner.LastHealthCheck = t.CreationDate

	logging.Logger.Info("add_miner: The new miner info",
		zap.String("base URL", newMiner.N2NHost),
		zap.String("ID", newMiner.ID),
		zap.String("pkey", newMiner.PublicKey),
		zap.String("mscID", msc.ID),
		zap.String("delegate_wallet", newMiner.Settings.DelegateWallet),
		zap.Float64("service_charge", newMiner.Settings.ServiceChargeRatio),
		zap.Int("num_delegates", newMiner.Settings.MaxNumDelegates),
	)

	if newMiner.PublicKey == "" || newMiner.ID == "" {
		logging.Logger.Error("public key or ID is empty")
		return "", common.NewError("add_miner",
			"PublicKey or the ID is empty. Cannot proceed")
	}

	// Check delegate wallet is not the same as operational wallet (PUK)
	if err := commonsc.ValidateDelegateWallet(newMiner.PublicKey, newMiner.Settings.DelegateWallet); err != nil {
		return "", err
	}

	err = validateNodeSettings(newMiner, gn, "add_miner")
	if err != nil {
		return "", err
	}

	newMiner.NodeType = NodeTypeMiner // set node type
	newMiner.ProviderType = spenum.Miner

	exist, err := getMinerNode(newMiner.ID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("add_miner", "could not get miner: %v", err)
	}

	if exist != nil {
		logging.Logger.Info("add_miner: miner already exists", zap.String("ID", newMiner.ID))
		return string(newMiner.Encode()), nil
	}

	if err = insertNodeN2NHost(balances, ADDRESS, newMiner); err != nil {
		return "", common.NewError("add_miner", err.Error())
	}

	nodeIDs, err := getNodeIDs(balances, AllMinersKey)
	if err != nil {
		return "", common.NewErrorf("add_miner", "could not get miner ids: %v", err)
	}

	nodeIDs = append(nodeIDs, newMiner.ID)
	if err := nodeIDs.save(balances, AllMinersKey); err != nil {
		return "", common.NewErrorf("add_miner", "save miner to list failed: %v", err)
	}

	if err := newMiner.save(balances); err != nil {
		return "", common.NewErrorf("add_miner", "save failed: %v", err)
	}

	emitAddMiner(newMiner, balances)

	return string(newMiner.Encode()), nil
}

// DeleteMiner Function to handle removing a miner from the chain
func (msc *MinerSmartContract) DeleteMiner(
	_ *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (string, error) {
	// actErr := cstate.WithActivation(balances, "ares", func() error {
	// 	return nil
	// }, func() error {
	// 	return errors.New("delete miner is disabled")
	// })
	// if actErr != nil {
	// 	return "", actErr
	// }

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

	lfmb := balances.GetLastestFinalizedMagicBlock()
	cloneMB := lfmb.MagicBlock.Clone()
	cloneMB.Miners.Delete(mn.ID)
	cloneMB.Mpks.Delete(mn.ID)
	cloneMB.ShareOrSigns.Delete(mn.ID)

	cloneMB.PreviousMagicBlockHash = lfmb.MagicBlock.Hash
	cloneMB.MagicBlockNumber = lfmb.MagicBlockNumber + 1
	nvcPeriod := PhaseRounds[Wait]
	cloneMB.StartingRound = ((balances.GetBlock().Round)/nvcPeriod + 1) * nvcPeriod

	dkgMiners := NewDKGMinerNodes()
	dkgMiners.calculateTKN(gn, cloneMB.Miners.Size())
	cloneMB.T = dkgMiners.T
	cloneMB.K = dkgMiners.K
	cloneMB.N = dkgMiners.N
	cloneMB.Hash = cloneMB.GetHash()
	logging.Logger.Debug("delete miner, new TKN:",
		zap.Int("T", cloneMB.T),
		zap.Int("K", cloneMB.K),
		zap.Int("N", cloneMB.N),
		zap.Int64("next vc", cloneMB.StartingRound),
		zap.Int("MB miner size", cloneMB.Miners.Size()))

	// msc.createMagicBlock()
	if err := updateMagicBlock(balances, cloneMB); err != nil {
		return "", common.NewError("delete_miner could not update magic block", err.Error())
	}

	debugMB, err := getMagicBlock(balances)
	if err != nil {
		return "", common.NewError("delete_miner could not get magic block", err.Error())
	}

	logging.Logger.Debug("delete miner, get magic block:", zap.Any("miner size", debugMB.Miners.Size()))

	gn.ViewChange = cloneMB.StartingRound
	if err := gn.save(balances); err != nil {
		return "", common.NewError("delete_miner could not save global node", err.Error())
	}

	return "delete miner successfully", nil
}

func (msc *MinerSmartContract) deleteNode(
	gn *GlobalNode,
	deleteNode *MinerNode,
	balances cstate.StateContextI,
) (*MinerNode, error) {
	// var err error
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

	logging.Logger.Debug("delete node",
		zap.String("node type", nodeType.String()),
		zap.String("id", deleteNode.ID))

	// err = saveDeleteNodeID(balances, nodeType, deleteNode.ID)
	// if err != nil {
	// 	return nil, err
	// }

	orderedPoolIds := deleteNode.OrderedPoolIds()
	for _, key := range orderedPoolIds {
		pool := deleteNode.Pools[key]
		switch pool.Status {
		case spenum.Active:
			pool.Status = spenum.Deleted
			_, err := deleteNode.UnlockPool(
				pool.DelegateID, nodeType, pool.DelegateID, balances)
			if err != nil {
				return nil, fmt.Errorf("error emptying delegate pool: %v", err)
			}
		case spenum.Deleted:
		default:
			return nil, fmt.Errorf(
				"unrecognised stakepool status: %v", pool.Status.String())
		}
	}

	// if err = deleteNode.save(balances); err != nil {
	// 	return nil, fmt.Errorf("saving node %v", err.Error())
	// }

	return deleteNode, nil
}

func (msc *MinerSmartContract) deleteMinerFromViewChange(mn *MinerNode, balances cstate.StateContextI) (err error) {
	var pn *PhaseNode
	if pn, err = GetPhaseNode(balances); err != nil {
		logging.Logger.Error("could not get phase node", zap.Error(err))
		return
	}
	if pn.Phase == Unknown {
		err = common.NewError("failed to delete from view change", "phase is unknown")
		return
	}
	if pn.Phase != Wait {
		var dkgMiners *DKGMinerNodes
		if dkgMiners, err = getDKGMinersList(balances); err != nil {
			logging.Logger.Error("delete_miner_from_view_change: Error in getting list from the DB", zap.Error(err))
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

	requiredUpdateInMinerNode := dto.NewMinerDtoNode()
	if err = json.Unmarshal(inputData, &requiredUpdateInMinerNode); err != nil {
		return "", common.NewErrorf("update_miner_settings",
			"decoding request: %v", err)
	}

	err = validateNodeUpdateSettings(requiredUpdateInMinerNode, gn, "update_miner_settings")
	if err != nil {
		return "", err
	}

	var mn *MinerNode
	mn, err = getMinerNode(requiredUpdateInMinerNode.ID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		mn = NewMinerNode()
		mn.ID = requiredUpdateInMinerNode.ID
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

	// only update when there were values sent
	if requiredUpdateInMinerNode.StakePool.StakePoolSettings.ServiceChargeRatio != nil {
		mn.Settings.ServiceChargeRatio = *requiredUpdateInMinerNode.StakePoolSettings.ServiceChargeRatio
	}

	if requiredUpdateInMinerNode.StakePool.StakePoolSettings.MaxNumDelegates != nil {
		mn.Settings.MaxNumDelegates = *requiredUpdateInMinerNode.StakePoolSettings.MaxNumDelegates
	}

	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("update_miner_settings", "saving: %v", err)
	}

	if err = emitUpdateMiner(mn, balances, false); err != nil {
		return "", common.NewErrorf("update_miner_settings", "saving: %v", err)
	}

	return string(mn.Encode()), nil
}

// ------------- local functions ---------------------

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
	if mn.ProviderType != spenum.Miner {
		return nil, fmt.Errorf("provider is %s should be %s", mn.ProviderType, spenum.Miner)
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

	return nil
}

func validateNodeUpdateSettings(update *dto.MinerDtoNode, gn *GlobalNode, opcode string) error {
	if update.StakePoolSettings.ServiceChargeRatio != nil {
		serviceChargeValue := *update.StakePoolSettings.ServiceChargeRatio
		if serviceChargeValue < 0 {
			return common.NewErrorf(opcode,
				"invalid negative service charge: %v", serviceChargeValue)
		}

		if serviceChargeValue > gn.MaxCharge {
			return common.NewErrorf(opcode,
				"max_charge is greater than allowed by SC: %v > %v",
				serviceChargeValue, gn.MaxCharge)
		}
	}

	if update.StakePoolSettings.MaxNumDelegates != nil {
		maxDelegateValue := *update.StakePoolSettings.MaxNumDelegates
		if maxDelegateValue <= 0 {
			return common.NewErrorf(opcode,
				"invalid non-positive number_of_delegates: %v", maxDelegateValue)
		}

		if maxDelegateValue > gn.MaxDelegates {
			return common.NewErrorf(opcode,
				"number_of_delegates greater than max_delegates of SC: %v > %v",
				maxDelegateValue, gn.MaxDelegates)
		}
	}

	return nil
}
