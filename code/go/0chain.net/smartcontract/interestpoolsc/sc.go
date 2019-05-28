package interestpoolsc

import (
	"fmt"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	Seperator = smartcontractinterface.Seperator
	owner     = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
	ADDRESS   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
	name      = "interest"
)

type InterestPoolSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ipsc *InterestPoolSmartContract) GetName() string {
	return name
}

func (ipsc *InterestPoolSmartContract) GetAddress() string {
	return ADDRESS
}

func (ipsc *InterestPoolSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return ipsc.RestHandlers
}

func (ipsc *InterestPoolSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ipsc.SmartContract = sc
	ipsc.SmartContract.RestHandlers["/getPoolsStats"] = ipsc.getPoolsStats
	ipsc.SmartContractExecutionStats["lock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "lock"), nil)
	ipsc.SmartContractExecutionStats["unlock"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "unlock"), nil)
	ipsc.SmartContractExecutionStats["updateVariables"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "updateVariables"), nil)
}

func (ip *InterestPoolSmartContract) lock(t *transaction.Transaction, un *UserNode, gn *GlobalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	npr := &newPoolRequest{}
	err := npr.decode(inputData)
	if err != nil {
		return "", common.NewError("failed locking tokens", fmt.Sprintf("request not formatted correctly (%v)", err.Error()))
	}
	if t.Value < gn.MinLock {
		return "", common.NewError("failed locking tokens", "insufficent amount to dig an interest pool")
	}
	balance, err := balances.GetClientBalance(t.ClientID)
	if err == util.ErrValueNotPresent {
		return "", common.NewError("failed locking tokens", "you have no tokens to your name")
	}
	if state.Balance(t.Value) > balance {
		return "", common.NewError("failed locking tokens", "lock amount is greater than balance")
	}
	if npr.Duration > gn.MaxLockPeriod {
		return "", common.NewError("failed locking tokens", fmt.Sprintf("duration (%v) is longer than max lock period (%v)", npr.Duration.String(), gn.MaxLockPeriod.String()))
	}
	if npr.Duration < gn.MinLockPeriod {
		return "", common.NewError("failed locking tokens", fmt.Sprintf("duration (%v) is shorter than min lock period (%v)", npr.Duration.String(), gn.MinLockPeriod.String()))
	}
	pool := newInterestPool()
	pool.TokenLockInterface = &tokenLock{StartTime: t.CreationDate, Duration: npr.Duration, Owner: un.ClientID}
	transfer, resp, err := pool.DigPool(t.Hash, t)
	if err == nil {
		balances.AddTransfer(transfer)
		pool.InterestRate = gn.InterestRate * float64(npr.Duration) / float64(gn.MaxLockPeriod)
		pool.InterestEarned = int64(float64(transfer.Amount) * pool.InterestRate)
		balances.AddMint(&state.Mint{Minter: ip.ID, ToClientID: transfer.ClientID, Amount: state.Balance(pool.InterestEarned)})
		un.addPool(pool)
		balances.InsertTrieNode(un.getKey(gn.ID), un)
		return resp, nil
	}
	return "", err
}

func (ip *InterestPoolSmartContract) unlock(t *transaction.Transaction, un *UserNode, gn *GlobalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	var response string
	var transfer *state.Transfer
	ps := &poolStat{}
	err := ps.decode(inputData)
	if err != nil {
		return "", common.NewError("failed to unlock tokens", fmt.Sprintf("input not formatted correctly: %v\n", err.Error()))
	}
	pool, ok := un.Pools[ps.ID]
	if ok {
		transfer, response, err = pool.EmptyPool(ip.ID, t.ClientID, common.ToTime(t.CreationDate))
		if err != nil {
			return "", common.NewError("failed to unlock tokens", fmt.Sprintf("error emptying pool %v", err.Error()))
		}
		err = un.deletePool(pool.ID)
		if err != nil {
			return "", common.NewError("failed to unlock tokens", fmt.Sprintf("error deleting pool from user node: %v", err.Error()))
		}
		balances.AddTransfer(transfer)
		balances.InsertTrieNode(un.getKey(gn.ID), un)
	} else {
		return "", common.NewError("failed to unlock tokens", fmt.Sprintf("pool (%v) doesn't exist", ps.ID))
	}
	return response, nil
}

func (ip *InterestPoolSmartContract) updateVariables(t *transaction.Transaction, gn *GlobalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	if t.ClientID != owner {
		return "", common.NewError("failed to update variables", "unauthorized access - only the owner can update the variables")
	}
	newGn := &GlobalNode{}
	err := newGn.Decode(inputData)
	if err != nil {
		return "", common.NewError("failed to update variables", "request not formatted correctly")
	}
	if newGn.InterestRate > 0.0 {
		gn.InterestRate = newGn.InterestRate
		config.SmartContractConfig.Set("smart_contracts.interestpoolsc.interest_rate", gn.InterestRate)
	}
	if newGn.MinLockPeriod > 0 {
		gn.MinLockPeriod = newGn.MinLockPeriod
		config.SmartContractConfig.Set("smart_contracts.interestpoolsc.min_lock_period", gn.MinLockPeriod)
	}
	if newGn.MaxLockPeriod > newGn.MinLockPeriod {
		gn.MaxLockPeriod = newGn.MaxLockPeriod
		config.SmartContractConfig.Set("smart_contracts.interestpoolsc.max_lock_period", gn.MaxLockPeriod)
	}
	if newGn.MinLock > 0 {
		gn.MinLock = newGn.MinLock
		config.SmartContractConfig.Set("smart_contracts.interestpoolsc.min_lock", gn.MinLock)
	}
	balances.InsertTrieNode(gn.getKey(), gn)
	return string(gn.Encode()), nil
}

func (ip *InterestPoolSmartContract) getUserNode(id datastore.Key, balances c_state.StateContextI) *UserNode {
	un := newUserNode(id)
	userBytes, err := balances.GetTrieNode(un.getKey(ip.ID))
	if err == nil {
		err = un.Decode(userBytes.Encode())
		if err == nil {
			return un
		}
	}
	return un
}

func (ip *InterestPoolSmartContract) getGlobalNode(balances c_state.StateContextI, funcName string) *GlobalNode {
	gn := newGlobalNode()
	globalBytes, err := balances.GetTrieNode(gn.getKey())
	if err == nil {
		err := gn.Decode(globalBytes.Encode())
		if err == nil {
			return gn
		}
	}
	gn.MinLockPeriod = config.SmartContractConfig.GetDuration("smart_contracts.interestpoolsc.min_lock_period")
	gn.MaxLockPeriod = config.SmartContractConfig.GetDuration("smart_contracts.interestpoolsc.max_lock_period")
	gn.InterestRate = config.SmartContractConfig.GetFloat64("smart_contracts.interestpoolsc.interest_rate")
	gn.MinLock = config.SmartContractConfig.GetInt64("smart_contracts.interestpoolsc.min_lock")
	if err == util.ErrValueNotPresent && funcName != "updateVariables" {
		balances.InsertTrieNode(gn.getKey(), gn)
	}
	return gn
}

func (ip *InterestPoolSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	un := ip.getUserNode(t.ClientID, balances)
	gn := ip.getGlobalNode(balances, funcName)
	switch funcName {
	case "lock":
		return ip.lock(t, un, gn, inputData, balances)
	case "unlock":
		return ip.unlock(t, un, gn, inputData, balances)
	case "updateVariables":
		return ip.updateVariables(t, gn, inputData, balances)
	default:
		return "", common.NewError("failed execution", "no function with that name")
	}
}
