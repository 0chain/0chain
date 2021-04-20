package interestpoolsc

import (
	"0chain.net/smartcontract"
	"context"
	"fmt"
	"net/url"
	"time"

	c_state "0chain.net/chaincore/chain/state"
)

func (ip *InterestPoolSmartContract) getPoolsStats(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un := ip.getUserNode(params.Get("client_id"), balances)
	if len(un.Pools) == 0 {
		err := fmt.Errorf("%w: %s", smartcontract.FailRetrievingStatsErr, "no pools exist")
		return nil, smartcontract.WrapErrNoResource(err)
	}
	t := time.Now()
	stats := &poolStats{}
	for _, pool := range un.Pools {
		stat, err := ip.getPoolStats(pool, t)
		if err != nil {
			err = smartcontract.NewError(smartcontract.FailRetrievingStatsErr, err)
			return nil, smartcontract.WrapErrInternal(err)
		}
		stats.addStat(stat)
	}
	return stats, nil
}

func (ip *InterestPoolSmartContract) getPoolStats(pool *interestPool, t time.Time) (*poolStat, error) {
	stat := &poolStat{}
	statBytes := pool.LockStats(t)
	err := stat.decode(statBytes)
	if err != nil {
		return nil, err
	}
	stat.ID = pool.ID
	stat.Locked = pool.IsLocked(t)
	stat.Balance = pool.Balance
	stat.APR = pool.APR
	stat.TokensEarned = pool.TokensEarned
	return stat, nil
}

func (ip *InterestPoolSmartContract) getLockConfig(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	return ip.getGlobalNode(balances, "updateVariables"), nil
}
