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
)

type InterestPoolSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ipsc *InterestPoolSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return ipsc.RestHandlers
}

func (ipsc *InterestPoolSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ipsc.SmartContract = sc
	ipsc.SmartContract.RestHandlers["/getPoolsStats"] = ipsc.getPoolsStats
	ipsc.SmartContractExecutionStats["lockTokens"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "lockTokens"), nil)
	ipsc.SmartContractExecutionStats["unlockTokens"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "unlockTokens"), nil)
	ipsc.SmartContractExecutionStats["updateVariables"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "updateVariables"), nil)
}

func (ip *InterestPoolSmartContract) lockTokens(t *transaction.Transaction, un *UserNode, gn *GlobalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	tp := newTypePool()
	err := tp.decode(inputData)
	if err != nil {
		return "", common.NewError("failed locking tokens", "request not formatted correctly")
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
	pool := newTypePool()
	pool.Type = tp.Type
	pool.TokenLockInterface = &tokenLock{StartTime: t.CreationDate, Duration: gn.LockPeriod, Owner: un.ClientID}
	transfer, resp, err := pool.DigPool(t.Hash, t)
	if err == nil {
		un.addPool(pool)
		_, err := balances.InsertTrieNode(un.getKey(gn.ID), un)
		if err == nil {
			balances.AddTransfer(transfer)
			if pool.Type == INTEREST {
				balances.AddMint(&state.Mint{Minter: ip.ID, ToClientID: transfer.ClientID, Amount: state.Balance(float64(transfer.Amount) * gn.InterestRate)})
			}
			return resp, nil
		}
		return "", err
	}
	return "", err
}

func (ip *InterestPoolSmartContract) unlockTokens(t *transaction.Transaction, un *UserNode, gn *GlobalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	if len(un.Pools) == 0 {
		return "", common.NewError("failed to unlock", fmt.Sprintf("no pools exist for user %v", un.ClientID))
	}
	var responses transferResponses
	unlockCount := 0
	for _, pool := range un.Pools {
		transfer, resp, err := pool.EmptyPool(ip.ID, t.ClientID, t)
		if err == nil {
			err := un.deletePool(pool.ID)
			if err == nil {
				unlockCount++
				responses.addResponse(resp)
				balances.AddTransfer(transfer)
				if pool.Type == STAKE {
					balances.AddMint(&state.Mint{Minter: ip.ID, ToClientID: transfer.ToClientID, Amount: state.Balance(float64(transfer.Amount) * gn.InterestRate)})
				}
			}
		} else {
			responses.addResponse(err.Error())
		}
	}
	if unlockCount != 0 {
		balances.InsertTrieNode(un.getKey(gn.ID), un)
	}
	return string(responses.encode()), nil
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
	if newGn.LockPeriod > 0 {
		gn.LockPeriod = newGn.LockPeriod
		config.SmartContractConfig.Set("smart_contracts.interestpoolsc.lock_period", gn.LockPeriod)
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

func (ip *InterestPoolSmartContract) getGlobalNode(balances c_state.StateContextI) *GlobalNode {
	gn := newGlobalNode()
	globalBytes, err := balances.GetTrieNode(gn.getKey())
	if err == nil {
		err = gn.Decode(globalBytes.Encode())
		if err == nil {
			return gn
		}
	}
	gn.LockPeriod = config.SmartContractConfig.GetDuration("smart_contracts.interestpoolsc.lock_period")
	gn.InterestRate = config.SmartContractConfig.GetFloat64("smart_contracts.interestpoolsc.interest_rate")
	gn.MinLock = config.SmartContractConfig.GetInt64("smart_contracts.interestpoolsc.min_lock")
	return gn
}

func (ip *InterestPoolSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	un := ip.getUserNode(t.ClientID, balances)
	gn := ip.getGlobalNode(balances)
	switch funcName {
	case "lockTokens":
		return ip.lockTokens(t, un, gn, inputData, balances)
	case "unlockTokens":
		return ip.unlockTokens(t, un, gn, inputData, balances)
	case "updateVariables":
		return ip.updateVariables(t, gn, inputData, balances)
	default:
		return "", common.NewError("failed execution", "no function with that name")
	}
}
