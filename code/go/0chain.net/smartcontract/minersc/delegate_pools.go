package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) addToDelegatePool(t *transaction.Transaction,
	input []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	beforeFunc := func() {
		resp, err = stakepool.StakePoolLock(t, input, balances,
			stakepool.ValidationSettings{MaxStake: gn.MaxStake, MinStake: gn.MinStake, MaxNumDelegates: gn.MaxDelegates}, msc.getStakePoolAdapter)
	}

	afterFunc := func() {
		resp, err = stakepool.StakePoolLock(t, input, balances,
			stakepool.ValidationSettings{MaxStake: gn.MaxStake, MinStake: gn.MinStake, MaxNumDelegates: gn.MaxDelegates}, msc.getStakePoolAdapter, msc.refreshProvider)
	}

	cstate.WithActivation(balances, "hard_fork_1", beforeFunc, afterFunc)

	return resp, err
}

// getStakePool of given blobber
func (_ *MinerSmartContract) getStakePoolAdapter(pType spenum.Provider, providerID string,
	balances cstate.StateContextI) (sp stakepool.AbstractStakePool, err error) {
	var mn *MinerNode
	switch pType {
	case spenum.Miner:
		mn, err = getMinerNode(providerID, balances)
		if mn != nil && mn.NodeType != NodeTypeMiner {
			return nil, common.NewErrorf("get_stake_pool",
				"wrong provider type")
		}
	case spenum.Sharder:
		mn, err = getSharderNode(providerID, balances)
		if mn != nil && mn.NodeType != NodeTypeSharder {
			return nil, common.NewErrorf("get_stake_pool",
				"wrong provider type")
		}
	default:
		return mn, common.NewErrorf("get_stake_pool",
			"unknown provider type")
	}
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		return mn, common.NewErrorf("get_stake_pool",
			"miner not found or genesis miner used")
	default:
		return mn, common.NewErrorf("get_stake_pool",
			"unexpected DB error: %v", err)
	}

	return mn, nil
}

func (msc *MinerSmartContract) deleteFromDelegatePool(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	beforeFunc := func() {
		resp, err = stakepool.StakePoolUnlock(t, inputData, balances, msc.getStakePoolAdapter)
	}

	afterFunc := func() {
		resp, err = stakepool.StakePoolUnlock(t, inputData, balances, msc.getStakePoolAdapter, msc.refreshProvider)
	}

	cstate.WithActivation(balances, "hard_fork_1", beforeFunc, afterFunc)

	return resp, err
}

// getStakePool of given blobber
func (msc *MinerSmartContract) refreshProvider(
	providerType spenum.Provider, providerID string, balances cstate.StateContextI,
) (s stakepool.AbstractStakePool, err error) {

	logging.Logger.Info("refresh_provider", zap.String("provider_type", providerType.String()), zap.String("provider_id", providerID))

	sp, err := getStakePool(providerType, providerID, balances)
	if err != nil {
		return nil, err
	}

	logging.Logger.Info("refresh_provider", zap.Any("stake_pool", sp))

	totalStakePoolBalance, err := sp.TotalStake()
	if err != nil {
		return nil, err
	}

	if providerType == spenum.Miner {
		mn, err := getMinerNode(providerID, balances)
		if err != nil {
			return nil, err
		}

		logging.Logger.Info("refresh_provider", zap.Any("miner_node", mn))

		mn.TotalStaked = totalStakePoolBalance

		if err := mn.save(balances); err != nil {
			return nil, common.NewErrorf("refresh_provider",
				"failed to save miner node: %v", err)
		}

		return nil, nil
	} else if providerType == spenum.Sharder {
		sn, err := getSharderNode(providerID, balances)
		if err != nil {
			return nil, err
		}

		sn.TotalStaked = totalStakePoolBalance

		if err := sn.save(balances); err != nil {
			return nil, common.NewErrorf("refresh_provider",
				"failed to save sharder node: %v", err)
		}

		return nil, nil
	}
	return nil, nil
}

func getStakePool(providerType spenum.Provider, providerID datastore.Key, balances cstate.CommonStateContextI) (
	sp *stakepool.StakePool, err error) {
	err = balances.GetTrieNode(stakepool.StakePoolKey(providerType, providerID), sp)
	if err != nil {
		return nil, err
	}
	return sp, nil
}
