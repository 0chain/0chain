package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"0chain.net/smartcontract/stakepool"
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

	if fnd, lnd := mn.numDelegates(), mn.NumberOfDelegates; fnd >= lnd {
		return "", common.NewErrorf("delegate_pool_add",
			"max delegates already reached: %d (%d)", fnd, lnd)
	}

	if fnd, scn := mn.numDelegates(), gn.MaxDelegates; fnd >= scn {
		return "", common.NewErrorf("delegate_pool_add",
			"SC max delegates already reached: %d (%d)", fnd, scn)
	}

	if t.Value < int64(mn.MinStake) {
		return "", common.NewErrorf("delegate_pool_add",
			"stake is less than min allowed: %d < %d", t.Value, mn.MinStake)
	}
	if t.Value > int64(mn.MaxStake) {
		return "", common.NewErrorf("delegate_pool_add",
			"stake is greater than max allowed: %d > %d", t.Value, mn.MaxStake)
	}

	if err := mn.LockPool(t, stakepool.Miner, mn.ID, stakepool.Pending, balances); err != nil {
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
	case stakepool.Pending:
		{
			_, err := mn.UnlockPool(t.ClientID, stakepool.Blobber, dp.MinerID, dp.PoolID, balances)
			if err != nil {
				return "", common.NewErrorf("delegate_pool_del",
					"stake_pool_unlock_failed: %v", err)
			}
			if err = mn.save(balances); err != nil {
				return "", common.NewError("delegate_pool_del", err.Error())
			}
			return resp, nil
		}
	case stakepool.Active:
		{
			pool.Status = stakepool.Deleting
			//pool.TokenLockInterface = &ViewChangeLock{
			//	Owner:               t.ClientID,
			//	DeleteViewChangeSet: true,
			//	DeleteVC:            gn.ViewChange,
			//}
			if err = mn.save(balances); err != nil {
				return "", common.NewErrorf("delegate_pool_del",
					"saving miner node: %v", err)
			}
			return `{"action": "pool will be released next VC"}`, nil
		}
	case stakepool.Deleting:
		return "", common.NewError("delegate_pool_del",
			"pool already deleted")
	case stakepool.Deleted:
		return "", common.NewError("delegate_pool_del",
			"pool already deleted")
	default:
		return "", common.NewErrorf("delegate_pool_del",
			"unrecognised stakepool status: %v", pool.Status.String())
	}
}
