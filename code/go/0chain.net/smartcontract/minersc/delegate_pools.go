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
func (msc *MinerSmartContract) getStakePoolAdapter(_ spenum.Provider, providerID string,
	balances cstate.CommonStateContextI) (sp stakepool.AbstractStakePool, err error) {
	var mn *MinerNode
	mn, err = getMinerNode(providerID, balances)
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

	var dp deletePool
	if err = dp.Decode(inputData); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error decoding request: %v", err)
	}

	var mn *MinerNode
	if mn, err = getMinerNode(dp.MinerID, balances); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error getting miner node: %v", err)
	}

	pool, ok := mn.Pools[t.ClientID]
	if !ok {
		return "", common.NewError("delegate_pool_del",
			"pool does not exist for deletion")
	}

	if pool.DelegateID != t.ClientID {
		return "", common.NewErrorf("delegate_pool_del",
			"you (%v) do not own the pool, it belongs to %v",
			t.ClientID, pool.DelegateID)
	}

	switch pool.Status {
	case spenum.Pending:
		{
			if err = mn.save(balances); err != nil {
				return "", common.NewError("delegate_pool_del", err.Error())
			}
			return resp, nil
		}
	case spenum.Active:
		{
			pool.Status = spenum.Deleting
			if err = mn.save(balances); err != nil {
				return "", common.NewErrorf("delegate_pool_del",
					"saving miner node: %v", err)
			}
			return `{"action": "pool will be released next VC"}`, nil
		}
	case spenum.Deleting:
		return "", common.NewError("delegate_pool_del",
			"pool already deleted")
	case spenum.Deleted:
		return "", common.NewError("delegate_pool_del",
			"pool already deleted")
	default:
		return "", common.NewErrorf("delegate_pool_del",
			"unrecognised stakepool status: %v", pool.Status.String())
	}
}
