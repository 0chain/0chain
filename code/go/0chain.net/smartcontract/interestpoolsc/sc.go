package interestpoolsc

import (
	"fmt"
	"time"

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
	ADDRESS   = "cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4"
	name      = "interest"
	YEAR      = time.Duration(time.Hour * 8784)
)

type InterestPoolSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ipsc *InterestPoolSmartContract) InitSC() {}

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
	ipsc.SmartContract.RestHandlers["/getLockConfig"] = ipsc.getLockConfig
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
	if t.Value < int64(gn.MinLock) {
		return "", common.NewError("failed locking tokens", "insufficent amount to dig an interest pool")
	}
	balance, err := balances.GetClientBalance(t.ClientID)
	if err == util.ErrValueNotPresent {
		return "", common.NewError("failed locking tokens", "you have no tokens to your name")
	}
	if state.Balance(t.Value) > balance {
		return "", common.NewError("failed locking tokens", "lock amount is greater than balance")
	}
	if npr.Duration > YEAR {
		return "", common.NewError("failed locking tokens", fmt.Sprintf("duration (%v) is longer than max lock period (%v)", npr.Duration.String(), YEAR.String()))
	}
	if npr.Duration < gn.MinLockPeriod {
		return "", common.NewError("failed locking tokens", fmt.Sprintf("duration (%v) is shorter than min lock period (%v)", npr.Duration.String(), gn.MinLockPeriod.String()))
	}
	if !gn.canMint() {
		return "", common.NewError("failed locking tokens", "can't mint anymore")
	}
	pool := newInterestPool()
	pool.TokenLockInterface = &tokenLock{StartTime: t.CreationDate, Duration: npr.Duration, Owner: un.ClientID}
	transfer, resp, err := pool.DigPool(t.Hash, t)
	if err == nil {
		balances.AddTransfer(transfer)
		pool.APR = gn.APR
		pool.TokensEarned = state.Balance(
			float64(transfer.Amount) * gn.APR * float64(npr.Duration) / float64(YEAR),
		)
		balances.AddMint(&state.Mint{
			Minter:   ip.ID,
			Receiver: transfer.Sender,
			Amount:   pool.TokensEarned,
		})
		// add to total minted
		gn.TotalMinted += pool.TokensEarned
		balances.InsertTrieNode(gn.getKey(), gn)
		// add to user pools
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
	const pfx = "smart_contracts.interestpoolsc."
	var conf = config.SmartContractConfig
	if newGn.APR > 0.0 {
		gn.APR = newGn.APR
		conf.Set(pfx+"interest_rate", gn.APR)
	}
	if newGn.MinLockPeriod > 0 {
		gn.MinLockPeriod = newGn.MinLockPeriod
		conf.Set(pfx+"min_lock_period", gn.MinLockPeriod)
	}
	if newGn.MinLock > 0 {
		gn.MinLock = newGn.MinLock
		conf.Set(pfx+"min_lock", gn.MinLock)
	}
	if newGn.MaxMint > 0 {
		gn.MaxMint = newGn.MaxMint
		conf.Set(pfx+"max_mint", gn.MaxMint)
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
	const pfx = "smart_contracts.interestpoolsc."
	var conf = config.SmartContractConfig
	gn.MinLockPeriod = conf.GetDuration(pfx + "min_lock_period")
	gn.APR = conf.GetFloat64(pfx + "apr")
	gn.MinLock = state.Balance(conf.GetInt64(pfx + "min_lock"))
	gn.MaxMint = state.Balance(conf.GetFloat64(pfx+"max_mint") * 1e10)
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
