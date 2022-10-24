package minersc

import (
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/partitions"
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

	part, err := getNodePartition(balances, dp.MinerID)
	if err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"error getting node partition: %v", err)
	}

	if err := part.Update(balances, GetNodeKey(dp.MinerID), func(data []byte) ([]byte, error) {
		mn := NewMinerNode()
		_, err := mn.UnmarshalMsg(data)
		if err != nil {
			return nil, fmt.Errorf("could not decode node: %v", err)
		}

		if mn.Delete {
			return nil, errors.New("can't add delegate pool for miner being deleted")
		}

		numDelegates := mn.numDelegates()
		if numDelegates >= mn.Settings.MaxNumDelegates {
			return nil, fmt.Errorf("max delegates already reached: %d (%d)", numDelegates, mn.Settings.MaxNumDelegates)
		}

		if numDelegates >= gn.MaxDelegates {
			return nil, fmt.Errorf("SC max delegates already reached: %d (%d)", numDelegates, gn.MaxDelegates)
		}

		if t.Value < mn.Settings.MinStake {
			return nil, fmt.Errorf("stake is less than min allowed: %d < %d", t.Value, mn.Settings.MinStake)
		}
		if t.Value > mn.Settings.MaxStake {
			return nil, fmt.Errorf("stake is greater than max allowed: %d > %d", t.Value, mn.Settings.MaxStake)
		}

		if err := mn.LockPool(t, spenum.Provider(mn.NodeType), mn.ID, spenum.Pending, balances); err != nil {
			return nil, fmt.Errorf("digging delegate pool: %v", err)
		}

		resp = string(mn.Encode())
		return mn.MarshalMsg(nil)
	}); err != nil {
		return "", common.NewErrorf("delegate_pool_add", err.Error())
	}

	if err := part.Save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_add",
			"saving node partition: %v", err)
	}

	return resp, nil
}

func getNodePartition(balances cstate.StateContextI, id string) (*partitions.Partitions, error) {
	part, err := minersPartitions.getPart(balances)
	if err != nil {
		return nil, fmt.Errorf("error getting miners partition: %v", err)
	}

	ok, err := part.Exist(balances, GetNodeKey(id))
	if err != nil {
		return nil, fmt.Errorf("error checking provider existence: %v", err)
	}

	if ok {
		return part, nil
	}

	part, err = shardersPartitions.getPart(balances)
	if err != nil {
		return nil, fmt.Errorf("error getting sharders partition: %v", err)
	}

	ok, err = part.Exist(balances, GetNodeKey(id))
	if err != nil {
		return nil, fmt.Errorf("error checking provider existence: %v", err)
	}

	if ok {
		return part, nil
	}

	return nil, fmt.Errorf("provider %v not found", id)
}

func (msc *MinerSmartContract) deleteFromDelegatePool(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	var dp deletePool
	if err = dp.Decode(inputData); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error decoding request: %v", err)
	}

	part, err := getNodePartition(balances, dp.MinerID)
	if err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error getting node partition: %v", err)
	}

	if err := part.Update(balances, GetNodeKey(dp.MinerID), func(data []byte) ([]byte, error) {
		mn := NewMinerNode()
		_, err := mn.UnmarshalMsg(data)
		if err != nil {
			return nil, fmt.Errorf("error decoding node: %v", err)
		}

		pool, ok := mn.Pools[t.ClientID]
		if !ok {
			return nil, errors.New("pool does not exist for deletion")
		}

		if pool.DelegateID != t.ClientID {
			return nil, fmt.Errorf("you (%v) do not own the pool, it belongs to %v",
				t.ClientID, pool.DelegateID)
		}

		switch pool.Status {
		case spenum.Pending:
			{
				_, err := mn.UnlockClientStakePool(t.ClientID, spenum.Miner, dp.MinerID, balances)
				if err != nil {
					return nil, fmt.Errorf("stake_pool_unlock_failed: %v", err)
				}

				return mn.MarshalMsg(nil)
			}
		case spenum.Active:
			{
				pool.Status = spenum.Deleting
				resp = `{"action": "pool will be released next VC"}`
				return mn.MarshalMsg(nil)
			}
		case spenum.Deleting:
			return nil, errors.New("pool already deleted")
		case spenum.Deleted:
			return nil, errors.New("pool already deleted")
		default:
			return nil, fmt.Errorf("unrecognised stakepool status: %v", pool.Status.String())
		}
	}); err != nil {
		return "", common.NewErrorf("delegate_pool_del", err.Error())
	}

	if err := part.Save(balances); err != nil {
		return "", common.NewErrorf("delegate_pool_del",
			"error saving node partition: %v", err)
	}

	return resp, nil
}
