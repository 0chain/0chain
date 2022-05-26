package interestpoolsc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/smartcontract"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"github.com/rcrowley/go-metrics"
)

const (
	Seperator = smartcontractinterface.Seperator
	ADDRESS   = "cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4"
	name      = "interest"
	YEAR      = time.Duration(time.Hour * 8784)
)

type InterestPoolSmartContract struct {
	*smartcontractinterface.SmartContract
}

func NewInterestPoolSmartContract() smartcontractinterface.SmartContractInterface {
	var ipscCopy = &InterestPoolSmartContract{
		SmartContract: smartcontractinterface.NewSC(ADDRESS),
	}
	ipscCopy.setSC(ipscCopy.SmartContract, &smartcontract.BCContext{})
	return ipscCopy
}

func (ipsc *InterestPoolSmartContract) GetName() string {
	return name
}

func (ipsc *InterestPoolSmartContract) GetAddress() string {
	return ADDRESS
}

func (ipsc *InterestPoolSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return ipsc.SmartContract.HandlerStats(ctx, params)
}

func (ipsc *InterestPoolSmartContract) GetExecutionStats() map[string]interface{} {
	return ipsc.SmartContractExecutionStats
}

func (ipsc *InterestPoolSmartContract) GetCost(t *transaction.Transaction, funcName string, balances c_state.StateContextI) (int, error) {
	n, err := ipsc.getGlobalNode(balances, funcName)
	if err != nil {
		return math.MaxInt32, fmt.Errorf("can't get global node, %v", err)
	}
	if n == nil {
		return math.MaxInt32, errors.New("can't get global node")
	}
	if n.Cost == nil {
		return math.MaxInt32, errors.New("can't get cost")
	}
	cost, ok := n.Cost[funcName]
	if !ok {
		return math.MaxInt32, errors.New("no cost given for " + funcName)
	}
	return cost, nil
}

func (ipsc *InterestPoolSmartContract) setSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ipsc.SmartContract = sc
	//ipsc.SmartContract.RestHandlers["/getPoolsStats"] = ipsc.getPoolsStats
	//ipsc.SmartContract.RestHandlers["/getLockConfig"] = ipsc.getLockConfig
	//ipsc.SmartContract.RestHandlers["/getConfig"] = ipsc.getConfig
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
	if currency.Coin(t.Value) > balance {
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
	pool.TokenLockInterface = &TokenLock{StartTime: t.CreationDate, Duration: npr.Duration, Owner: un.ClientID}
	transfer, resp, err := pool.DigPool(t.Hash, t)
	if err == nil {
		if err := balances.AddTransfer(transfer); err != nil {
			return "", common.NewErrorf("failed locking tokens", "could not add transfer: %v", err)
		}
		pool.APR = gn.APR
		pool.TokensEarned = currency.Coin(
			float64(transfer.Amount) * gn.APR * float64(npr.Duration) / float64(YEAR),
		)
		if err := balances.AddMint(&state.Mint{
			Minter:     ip.ID,
			ToClientID: transfer.ClientID,
			Amount:     pool.TokensEarned,
		}); err != nil {
			return "", err
		}
		// add to total minted
		gn.TotalMinted += pool.TokensEarned
		if _, err := balances.InsertTrieNode(gn.getKey(), gn); err != nil {
			return "", common.NewError("failed locking tokens", err.Error())
		}
		// add to user pools
		if err := un.addPool(pool); err != nil {
			return "", err
		}
		if _, err := balances.InsertTrieNode(un.getKey(gn.ID), un); err != nil {
			return "", common.NewError("failed locking tokens", err.Error())
		}
		return resp, nil
	}
	return "", err
}

func (ip *InterestPoolSmartContract) unlock(t *transaction.Transaction, un *UserNode, gn *GlobalNode, inputData []byte, balances c_state.StateContextI) (string, error) {
	ps := &poolStat{}
	err := ps.decode(inputData)
	if err != nil {
		return "", common.NewError("failed to unlock tokens",
			fmt.Sprintf("input not formatted correctly: %v\n", err.Error()))
	}
	pool, ok := un.Pools[ps.ID]
	if ok {
		transfer, response, err := pool.EmptyPool(ip.ID, t.ClientID, common.ToTime(t.CreationDate))
		if err != nil {
			return "", common.NewError("failed to unlock tokens", fmt.Sprintf("error emptying pool %v", err.Error()))
		}
		err = un.deletePool(pool.ID)
		if err != nil {
			return "", common.NewError("failed to unlock tokens", fmt.Sprintf("error deleting pool from user node: %v", err.Error()))
		}
		if err := balances.AddTransfer(transfer); err != nil {
			return "", common.NewErrorf("failed to unlock token", "add transfer err: %v", err)
		}
		if _, err := balances.InsertTrieNode(un.getKey(gn.ID), un); err != nil {
			return "", common.NewError("failed to unlock tokens", err.Error())
		}
		return response, nil
	}
	return "", common.NewError("failed to unlock tokens", fmt.Sprintf("pool (%v) doesn't exist", ps.ID))
}

func (ip *InterestPoolSmartContract) getUserNode(id datastore.Key, balances c_state.StateContextI) (*UserNode, error) {
	un := newUserNode(id)
	err := balances.GetTrieNode(un.getKey(ip.ID), un)
	switch err {
	case nil, util.ErrValueNotPresent:
		return un, nil
	default:
		return nil, err
	}
}

func (ip *InterestPoolSmartContract) getGlobalNode(balances c_state.StateContextI, funcName string) (*GlobalNode, error) {
	gn := newGlobalNode()
	err := balances.GetTrieNode(gn.getKey(), gn)
	switch err {
	case nil:
		return gn, nil
	case util.ErrValueNotPresent:
		const pfx = "smart_contracts.interestpoolsc."
		var conf = config.SmartContractConfig
		gn.MinLockPeriod = conf.GetDuration(pfx + "min_lock_period")
		gn.APR = conf.GetFloat64(pfx + "apr")
		gn.MinLock = currency.Coin(conf.GetInt64(pfx + "min_lock"))
		var err2 error
		gn.MaxMint, err2 = currency.ParseZCN(conf.GetFloat64(pfx + "max_mint"))
		if err2 != nil {
			return nil, err2
		}
		gn.OwnerId = conf.GetString(pfx + "owner_id")
		gn.Cost = conf.GetStringMapInt(pfx + "cost")
		if funcName != "updateVariables" {
			if _, err := balances.InsertTrieNode(gn.getKey(), gn); err != nil {
				return nil, err
			}
		}
		return gn, nil
	default:
		return nil, err
	}
}

func (ip *InterestPoolSmartContract) Execute(t *transaction.Transaction, funcName string, inputData []byte, balances c_state.StateContextI) (string, error) {
	un, err := ip.getUserNode(t.ClientID, balances)
	if err != nil {
		return "", common.NewError("failed execution - get user node", err.Error())
	}

	gn, err := ip.getGlobalNode(balances, funcName)
	if err != nil {
		return "", common.NewError("failed execution - get global node", err.Error())
	}

	switch funcName {
	case "lock":
		return ip.lock(t, un, gn, inputData, balances)
	case "unlock":
		return ip.unlock(t, un, gn, inputData, balances)
	case "updateVariables":
		return ip.updateVariables(t, gn, inputData, balances)
	default:
		return "", common.NewErrorf("failed execution", "no interest pool smart contract method with name %s", funcName)
	}
}
