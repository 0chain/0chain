package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract/stakepool/spenum"
)

func (msc *MinerSmartContract) addToDelegatePool(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var dp deletePool
	if err = dp.Decode(inputData); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"decoding request: %v", err)
	}

	var mn *MinerNode
	mn, err = getMinerNode(dp.MinerID, balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		return "", common.NewErrorf("delegate_pool_add",
			"miner not found or genesis miner used")
	default:
		return "", common.NewErrorf("delegate_pool_add",
			"unexpected DB error: %v", err)
	}

	if mn.Delete {
		return "", common.NewError("delegate_pool_add",
			"can't add delegate pool for miner being deleted")
	}

	numDelegates := mn.numDelegates()
	if numDelegates >= mn.Settings.MaxNumDelegates {
		return "", common.NewErrorf("delegate_pool_add",
			"max delegates already reached: %d (%d)", numDelegates, mn.Settings.MaxNumDelegates)
	}

	if numDelegates >= gn.MaxDelegates {
		return "", common.NewErrorf("delegate_pool_add",
			"SC max delegates already reached: %d (%d)", numDelegates, gn.MaxDelegates)
	}

	if t.ValueZCN < int64(mn.Settings.MinStake) {
		return "", common.NewErrorf("delegate_pool_add",
			"stake is less than min allowed: %d < %d", t.ValueZCN, mn.Settings.MinStake)
	}
	if t.ValueZCN > int64(mn.Settings.MaxStake) {
		return "", common.NewErrorf("delegate_pool_add",
			"stake is greater than max allowed: %d > %d", t.ValueZCN, mn.Settings.MaxStake)
	}

	if err := mn.LockPool(t, spenum.Miner, mn.ID, spenum.Pending, balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"digging delegate pool: %v", err)
	}

	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"saving miner node: %v", err)
	}

	resp = string(mn.Encode())
	return
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

	pool, ok := mn.Pools[dp.PoolID]
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
			_, err := mn.UnlockClientStakePool(t.ClientID, spenum.Miner, dp.MinerID, dp.PoolID, balances)
			if err != nil {
				return "", common.NewErrorf("delegate_pool_del",
					"stake_pool_unlock_failed: %v", err)
			}
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
