package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) addToDelegatePool(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
		resp string, err error) {

	var dp delegatePool
	if err = dp.Decode(inputData); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"decoding request: %v", err)
	}

	var un *UserNode
	if un, err = msc.getUserNode(t.ClientID, balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"getting user node: %v", err)
	}

	var (
		pool = sci.NewDelegatePool()

		mn       *MinerNode
		transfer *state.Transfer
	)
	mn, err = msc.getMinerNode(dp.MinerID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("delegate_pool_add",
			"unexpected DB error: %v", err)
	}

	if err == util.ErrValueNotPresent {
		return "", common.NewErrorf("delegate_pool_add",
			"miner not found or genesis miner used")
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

	pool.TokenLockInterface = &ViewChangeLock{
		Owner:               t.ClientID,
		DeleteViewChangeSet: false,
	}
	pool.DelegateID = t.ClientID
	pool.Status = PENDING

	Logger.Info("add delegate pool", zap.Any("pool", pool))

	if transfer, _, err = pool.DigPool(t.Hash, t); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"digging delegate pool: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"adding transfer: %v", err)
	}

	// user node pool information
	un.Pools[mn.ID] = append(un.Pools[mn.ID], t.Hash)

	// add to pending making it active next VC
	mn.Pending[t.Hash] = pool

	// save user node and the miner/sharder
	if err = un.save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"saving user node: %v", err)
	}
	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"saving miner node: %v", err)
	}

	resp = string(mn.Encode()) + string(transfer.Encode()) + string(un.Encode())
	return
}

func (msc *MinerSmartContract) deleteFromDelegatePool(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	var dp delegatePool
	if err = dp.Decode(inputData); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error decoding request: %v", err)
	}

	var mn *MinerNode
	if mn, err = msc.getMinerNode(dp.MinerID, balances); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error getting miner node: %v", err)
	}

	var un *UserNode
	if un, err = msc.getUserNode(t.ClientID, balances); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error getting user node: %v", err)
	}

	// just delete it if it's still pending
	if pool, ok := mn.Pending[dp.PoolID]; ok {
		if pool.DelegateID != t.ClientID {
			return "", common.NewErrorf("delegate_pool_del",
				"you (%v) do not own the pool, it belongs to %v",
				t.ClientID, pool.DelegateID)
		}
		var transfer *state.Transfer
		transfer, resp, err = pool.EmptyPool(msc.ID, t.ClientID, nil)
		if err != nil {
			return "", common.NewErrorf("delegate_pool_del",
				"error emptying delegate pool: %v", err)
		}

		if err = balances.AddTransfer(transfer); err != nil {
			return "", common.NewErrorf("delegate_pool_del",
				"adding transfer: %v", err)
		}

		delete(un.Pools, dp.PoolID)
		delete(mn.Pending, dp.PoolID)

		if err = un.save(balances); err != nil {
			return "", common.NewError("delegate_pool_del", err.Error())
		}

		if err = mn.save(balances); err != nil {
			return "", common.NewError("delegate_pool_del", err.Error())
		}

		return resp, nil
	}

	// move to deleting if it's active

	var pool, ok = mn.Active[dp.PoolID]
	if !ok {
		return "", common.NewError("delegate_pool_del",
			"pool does not exist for deletion")
	}

	if pool.Status == DELETING {
		return "", common.NewError("delegate_pool_del",
			"pool already deleted")
	}

	if pool.DelegateID != t.ClientID {
		return "", common.NewErrorf("delegate_pool_del",
			"you (%v) do not own the pool, it belongs to %v",
			t.ClientID, pool.DelegateID)
	}

	pool.Status = DELETING // mark as deleting
	pool.TokenLockInterface = &ViewChangeLock{
		Owner:               t.ClientID,
		DeleteViewChangeSet: true,
		DeleteVC:            gn.ViewChange,
	}
	mn.Deleting[dp.PoolID] = pool // add to deleting

	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"saving miner node: %v", err)
	}

	return `{"action": "pool will be released next VC"}`, nil
}
