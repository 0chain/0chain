package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"
)

func (msc *MinerSmartContract) addToDelegatePool(t *transaction.Transaction,
	input []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	r := stakepool.Restrictions{
		MinStake:     gn.MinStake,
		MaxStake:     gn.MaxStake,
		MaxDelegates: gn.MaxDelegates,
	}
	return stakepool.StakePoolLock(t, input, balances, r, msc.getStakePoolAdapter)
}

// getStakePool of given blobber
func (msc *MinerSmartContract) getStakePoolAdapter(pType spenum.Provider, providerID string,
	balances cstate.CommonStateContextI) (sp stakepool.AbstractStakePool, err error) {
	var mn *MinerNode
	switch pType {
	case spenum.Miner:
		mn, err = getMinerNode(providerID, balances)
	case spenum.Sharder:
		mn, err = getSharderNode(providerID, balances)
	}
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		return mn, common.NewErrorf("delegate_pool_add",
			"miner not found or genesis miner used")
	default:
		return mn, common.NewErrorf("delegate_pool_add",
			"unexpected DB error: %v", err)
	}
	return mn, nil
}

func (msc *MinerSmartContract) deleteFromDelegatePool(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	return stakepool.StakePoolUnlock(t, inputData, balances, msc.getStakePoolAdapter)
}
