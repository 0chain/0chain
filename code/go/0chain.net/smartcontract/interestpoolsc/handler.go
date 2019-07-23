package interestpoolsc

import (
	"context"
	"time"
	// "encoding/json"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

func (ip *InterestPoolSmartContract) getPoolsStats(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un := ip.getUserNode(params.Get("client_id"), balances)
	if len(un.Pools) == 0 {
		return nil, common.NewError("failed to get stats", "no pools exist")
	}
	t := time.Now()
	stats := &poolStats{}
	for _, pool := range un.Pools {
		stat, err := ip.getPoolStats(pool, t)
		if err != nil {
			return nil, common.NewError("failed to get stats", "crap this shouldn't happen")
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
