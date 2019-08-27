package minersc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"

	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) addToDelegatePool(t *transaction.Transaction, inputData []byte, gn *globalNode, balances c_state.StateContextI) (string, error) {
	mn := NewMinerNode()
	dp := &deletePool{}
	err := dp.Decode(inputData)
	if err != nil {
		return "", common.NewError("failed to add to delegate pool", fmt.Sprintf("error decoding request: %v", err.Error()))
	}
	var transfer *state.Transfer
	un, err := msc.getUserNode(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("failed to add to delegate pool", fmt.Sprintf("error getting user node: %v", err.Error()))
	}
	pool := sci.NewDelegatePool()
	mn, err = msc.getMinerNode(dp.MinerID, balances)
	if err == util.ErrValueNotPresent {

		mn.DelegateID = t.ClientID
		mn.TotalStaked += t.Value
		pool.TokenLockInterface = &ViewChangeLock{Owner: t.ClientID, DeleteViewChangeSet: false}
		pool.DelegateID = t.ClientID
		pool.InterestRate = gn.InterestRate
		pool.Status = ACTIVE
		Logger.Info("add pool", zap.Any("pool", pool))
		transfer, _, err = pool.DigPool(t.Hash, t)
		if err != nil {
			return "", common.NewError("failed to add to delegate pool", fmt.Sprintf("error digging delegate pool: %v", err.Error()))
		}

	} else {
		pool.TokenLockInterface = &ViewChangeLock{Owner: t.ClientID, DeleteViewChangeSet: false}
		pool.DelegateID = t.ClientID
		mn.TotalStaked += t.Value
		pool.InterestRate = gn.InterestRate
		pool.Status = ACTIVE
		Logger.Info("add pool", zap.Any("pool", pool))
		transfer, _, err = pool.DigPool(t.Hash, t)
		if err != nil {
			return "", common.NewError("failed to add to delegate pool", fmt.Sprintf("error digging delegate pool: %v", err.Error()))
		}
	}
	balances.AddTransfer(transfer)
	un.Pools[t.Hash] = &poolInfo{MinerID: mn.ID, Balance: int64(transfer.Amount)}

	mn.Active[t.Hash] = pool // needs to be Pending pool; doing this just for testing
	// mn.Pending[t.Hash] = pool
	balances.InsertTrieNode(un.GetKey(), un)
	balances.InsertTrieNode(mn.getKey(), mn)
	return string(mn.Encode()) + string(transfer.Encode()) + string(un.Encode()), nil
}

func (msc *MinerSmartContract) deleteFromDelegatePool(t *transaction.Transaction, inputData []byte, gn *globalNode, balances c_state.StateContextI) (string, error) {
	dp := &deletePool{}
	err := dp.Decode(inputData)
	if err != nil {
		return "", common.NewError("failed to delete from delegate pool", fmt.Sprintf("error decoding request: %v", err.Error()))
	}
	mn, err := msc.getMinerNode(dp.MinerID, balances)
	if err != nil {
		return "", common.NewError("failed to delete from delegate pool", fmt.Sprintf("error getting miner node: %v", err.Error()))
	}
	un, err := msc.getUserNode(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("failed to delete from delegate pool", fmt.Sprintf("error getting user node: %v", err.Error()))
	}
	if pool, ok := mn.Pending[dp.PoolID]; ok {
		if pool.DelegateID != t.ClientID {
			return "", common.NewError("failed to delete from delegate pool", fmt.Sprintf("you (%v) do not own the pool, it belongs to %v", t.ClientID, pool.DelegateID))
		}
		transfer, response, err := pool.EmptyPool(msc.ID, t.ClientID, nil)
		if err != nil {
			return "", common.NewError("failed to delete from delegate pool", fmt.Sprintf("error emptying delegate pool: %v", err.Error()))
		}
		balances.AddTransfer(transfer)
		delete(un.Pools, dp.PoolID)
		delete(mn.Pending, dp.PoolID)
		if len(un.Pools) > 0 {
			balances.InsertTrieNode(un.GetKey(), un)
		} else {
			balances.DeleteTrieNode(un.GetKey())
		}
		balances.InsertTrieNode(mn.getKey(), mn)
		return response, nil
	}
	if pool, ok := mn.Active[dp.PoolID]; ok {
		if pool.DelegateID != t.ClientID {
			return "", common.NewError("failed to delete from delegate pool", fmt.Sprintf("you (%v) do not own the pool, it belongs to %v", t.ClientID, pool.DelegateID))
		}
		switch pool.Status {
		case ACTIVE:
			pool.Status = DELETING
			mn.Active[dp.PoolID] = pool
			balances.InsertTrieNode(mn.getKey(), mn)
			return `{"action": "pool has been marked as deleting. Delete again to move to Deleting Pool"}`, nil
		case DELETING:
			// THIS WILL BE GONE ONCE VIEW CHANGE IS ADDED. VIEW CHAGNE WILL TAKE CARE OF THIS
			pool.Status = CANDELETE
			pool.TokenLockInterface = &ViewChangeLock{Owner: t.ClientID, DeleteViewChangeSet: true, DeleteVC: balances.GetBlock().Round}
			mn.Deleting[dp.PoolID] = pool
			delete(mn.Active, dp.PoolID)
			balances.InsertTrieNode(mn.getKey(), mn)
			return `{"action": "pool has been moved from active to deleting. Tokens are ready for release"}`, nil
		}

	}
	return "", common.NewError("failed to delete from delegate pool", "pool does not exist for deletion")
}

func (msc *MinerSmartContract) releaseFromDelegatePool(t *transaction.Transaction, inputData []byte, gn *globalNode, balances c_state.StateContextI) (string, error) {
	dp := &deletePool{}
	err := dp.Decode(inputData)
	if err != nil {
		return "", common.NewError("failed to release from delegate pool", fmt.Sprintf("error decoding request: %v", err.Error()))
	}
	mn, err := msc.getMinerNode(dp.MinerID, balances)
	if err != nil {
		return "", common.NewError("failed to release from delegate pool", fmt.Sprintf("error getting miner node: %v", err.Error()))
	}
	un, err := msc.getUserNode(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("failed to release from delegate pool", fmt.Sprintf("error getting user node: %v", err.Error()))
	}
	if pool, ok := mn.Deleting[dp.PoolID]; ok {
		if pool.DelegateID != t.ClientID {
			return "", common.NewError("failed to release from delegate pool", fmt.Sprintf("you (%v) do not own the pool, it belongs to %v", t.ClientID, pool.DelegateID))
		}
		interestEarned := state.Balance(pool.InterestRate * float64(pool.Balance))
		transfer, response, err := pool.EmptyPool(msc.ID, t.ClientID, balances.GetBlock().Round)
		if err != nil {
			return "", common.NewError("failed to release from delegate pool", fmt.Sprintf("error emptying delegate pool: %v", err.Error()))
		}
		balances.AddMint(state.NewMint(ADDRESS, t.ClientID, interestEarned))
		balances.AddTransfer(transfer)
		delete(un.Pools, dp.PoolID)
		delete(mn.Deleting, dp.PoolID)
		if len(un.Pools) > 0 {
			balances.InsertTrieNode(un.GetKey(), un)
		} else {
			balances.DeleteTrieNode(un.GetKey())
		}
		balances.InsertTrieNode(mn.getKey(), mn)
		return response, nil
	}
	return "", common.NewError("failed to delete from release pool", "pool does not exist")
}
