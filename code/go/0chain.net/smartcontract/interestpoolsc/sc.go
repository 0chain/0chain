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
)

const (
	Seperator = smartcontractinterface.Seperator
	owner     = "c8a5e74c2f4fae2c1bed79fb2b78d3b88f844bbb6bf1db5fc43240711f23321f"
	ADDRESS   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
)

type InterestPoolSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ipsc *InterestPoolSmartContract) SetSC(sc *smartcontractinterface.SmartContract) {
	ipsc.SmartContract = sc
}

func (ip *InterestPoolSmartContract) lockTokens(t *transaction.Transaction, un *userNode, gn *globalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	tp := newTypePool()
	err := tp.decode(inputData)
	if err != nil {
		return common.NewError("failed locking tokens", "request not formatted correctly").Error(), nil
	}
	if t.Value < gn.MinLock {
		return common.NewError("failed locking tokens", "insufficent amount to dig an interest pool").Error(), nil
	}
	pool := newTypePool()
	pool.Type = tp.Type
	pool.TokenLockInterface = &tokenLock{StartTime: t.CreationDate, Duration: gn.LockPeriod, Owner: un.ClientID}
	transfer, resp, err := pool.DigPool(t.Hash, t)
	if err == nil {
		un.addPool(pool)
		userBytes := un.encode()
		err := ip.DB.PutNode(un.getKey(), userBytes)
		if err == nil {
			balances.AddTransfer(transfer)
			if pool.Type == INTEREST {
				balances.AddMint(&state.Mint{Minter: ip.ID, ToClientID: transfer.ClientID, Amount: state.Balance(float64(transfer.Amount) * gn.InterestRate)})
			}
			return resp, nil
		}
		return err.Error(), nil
	}
	return err.Error(), nil
}

func (ip *InterestPoolSmartContract) unlockTokens(t *transaction.Transaction, un *userNode, gn *globalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	if len(un.Pools) == 0 {
		return fmt.Sprintf("no pools exist for user %v", un.ClientID), nil
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
		ip.DB.PutNode(un.getKey(), un.encode())
	}
	return string(responses.encode()), nil
}

func (ip *InterestPoolSmartContract) getPoolsStats(t *transaction.Transaction, un *userNode) (string, error) {
	if len(un.Pools) == 0 {
		return common.NewError("failed to get stats", "no pools exist").Error(), nil
	}
	stats := &poolStats{}
	for _, pool := range un.Pools {
		stat, err := ip.getPoolStats(pool, t)
		if err != nil {
			return "crap this shouldn't happen", nil
		}
		stats.addStat(stat)
	}
	return string(stats.encode()), nil
}

func (ip *InterestPoolSmartContract) getPoolStats(pool *typePool, t *transaction.Transaction) (*poolStat, error) {
	stat := &poolStat{}
	statBytes := pool.LockStats(t)
	err := stat.decode(statBytes)
	if err != nil {
		return nil, err
	}
	stat.ID = pool.ID
	stat.Locked = pool.IsLocked(t)
	stat.PoolType = pool.Type
	stat.Balance = pool.Balance
	return stat, nil
}

func (ip *InterestPoolSmartContract) updateVariables(t *transaction.Transaction, gn *globalNode, inputData []byte) (string, error) {
	if t.ClientID != owner {
		return common.NewError("unauthorized_access", "only the owner can update the variables").Error(), nil
	}
	newGn := &globalNode{}
	err := newGn.decode(inputData)
	if err != nil {
		return "request not formatted correctly", nil
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
	ip.DB.PutNode(gn.getKey(), gn.encode())
	return string(gn.encode()), nil
}

func (ip *InterestPoolSmartContract) getuserNode(id datastore.Key) *userNode {
	un := newUserNode(id)
	userBytes, err := ip.DB.GetNode(un.getKey())
	if err == nil {
		err = un.decode(userBytes)
		if err == nil {
			return un
		}
	}
	return un
}

func (ip *InterestPoolSmartContract) getGlobalNode() *globalNode {
	gn := newGlobalNode()
	globalBytes, err := ip.DB.GetNode(gn.getKey())
	if err == nil {
		err = gn.decode(globalBytes)
		if err == nil {
			return gn
		}
	}
	gn.LockPeriod = config.SmartContractConfig.GetDuration("smart_contracts.interestpoolsc.lock_period")
	gn.InterestRate = config.SmartContractConfig.GetFloat64("smart_contracts.interestpoolsc.interest_rate")
	gn.MinLock = config.SmartContractConfig.GetInt64("smart_contracts.interestpoolsc.min_lock")
	ip.DB.PutNode(gn.getKey(), gn.encode())
	return gn
}

func (ip *InterestPoolSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	un := ip.getuserNode(t.ClientID)
	gn := ip.getGlobalNode()
	switch funcName {
	case "lockTokens":
		return ip.lockTokens(t, un, gn, inputData, balances)
	case "unlockTokens":
		return ip.unlockTokens(t, un, gn, inputData, balances)
	case "getPoolsStats":
		return ip.getPoolsStats(t, un)
	case "updateVariables":
		return ip.updateVariables(t, gn, inputData)
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	}
}
