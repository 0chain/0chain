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

func (msc *MinerSmartContract) addToDelegatePool(tx *transaction.Transaction,
	inputData []byte, global *GlobalNode, balances cstate.StateContextI) (
		resp string, err error) {

	var dPool delegatePool
	if err = dPool.Decode(inputData); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"decoding request: %v", err)
	}

	var userNode *UserNode
	if userNode, err = msc.getUserNode(tx.ClientID, balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"getting user node: %v", err)
	}

	var (
		pool = sci.NewDelegatePool()

		node     *ConsensusNode
		transfer *state.Transfer
	)
	node, err = msc.getConsensusNode(dPool.ConsensusNodeID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("delegate_pool_add",
			"unexpected DB error: %v", err)
	}

	if err == util.ErrValueNotPresent {
		return "", common.NewErrorf("delegate_pool_add",
			"nodeconsensus node not found or genesis nodeconsensus node used")
	}

	var delegatesAmount = node.delegatesAmount();

	if nodeLimit := node.NumberOfDelegates; delegatesAmount >= nodeLimit {
		return "", common.NewErrorf("delegate_pool_add",
			"node's delegates limit already reached: %d (%d)", delegatesAmount, nodeLimit)
	}

	if scLimit := global.MaxDelegates; delegatesAmount >= scLimit {
		return "", common.NewErrorf("delegate_pool_add",
			"SC delegates limit already reached: %d (%d)", delegatesAmount, scLimit)
	}

	if tx.Value < int64(node.MinStake) {
		return "", common.NewErrorf("delegate_pool_add",
			"stake is less then min allowed: %d < %d", tx.Value, node.MinStake)
	}
	if tx.Value > int64(node.MaxStake) {
		return "", common.NewErrorf("delegate_pool_add",
			"stake is greater then max allowed: %d > %d", tx.Value, node.MaxStake)
	}

	pool.TokenLockInterface = &ViewChangeLock{
		Owner:               tx.ClientID,
		DeleteViewChangeSet: false,
	}
	pool.DelegateID = tx.ClientID
	pool.Status = PENDING

	Logger.Info("add delegate pool", zap.Any("pool", pool))

	if transfer, _, err = pool.DigPool(tx.Hash, tx); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"digging delegate pool: %v", err)
	}

	if err = balances.AddTransfer(transfer); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"adding transfer: %v", err)
	}

	// user node pool information
	userNode.Pools[node.ID] = append(userNode.Pools[node.ID], tx.Hash)

	// add to pending making it active next VC
	node.Pending[tx.Hash] = pool

	// save user node and the miner/sharder
	if err = userNode.save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"saving user node: %v", err)
	}
	if err = node.save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"saving nodeconsensus node: %v", err)
	}

	resp = string(node.Encode()) + string(transfer.Encode()) + string(userNode.Encode())
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

	var mn *ConsensusNode
	if mn, err = msc.getConsensusNode(dp.ConsensusNodeID, balances); err != nil {
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
