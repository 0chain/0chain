package interestpoolsc

import (
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"context"
	"fmt"
	"net/url"
	"time"

	c_state "0chain.net/chaincore/chain/state"
)

func (ip *InterestPoolSmartContract) getConfig(_ context.Context, _ url.Values, _ c_state.StateContextI) (interface{}, error) {
	const pfx = "smart_contracts.interestpoolsc."
	return &inputMap{
		Fields: map[string]interface{}{
			Settings[MinLock]:       config.SmartContractConfig.GetInt64(pfx + Settings[MinLock]),
			Settings[MaxMint]:       config.SmartContractConfig.GetInt64(pfx + Settings[MaxMint]),
			Settings[MinLockPeriod]: config.SmartContractConfig.GetInt64(pfx + Settings[MinLockPeriod]),
			Settings[Apr]:           config.SmartContractConfig.GetInt64(pfx + Settings[Apr]),
			"apr":                   config.SmartContractConfig.GetInt64(pfx + "apr"),
		},
	}, nil
}

func (ip *InterestPoolSmartContract) getPoolsStats(_ context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un := ip.getUserNode(params.Get("client_id"), balances)
	if len(un.Pools) == 0 {
		return nil, common.NewErrNoResource("can't find user node")
	}
	t := time.Now()
	stats := &poolStats{}
	for _, pool := range un.Pools {
		stat, err := ip.getPoolStats(pool, t)
		if err != nil {
			return nil, common.NewErrInternal("can't get pool stats", err.Error())
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
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	stat.ID = pool.ID
	stat.Locked = pool.IsLocked(t)
	stat.Balance = pool.Balance
	stat.APR = pool.APR
	stat.TokensEarned = pool.TokensEarned
	return stat, nil
}

func (ip *InterestPoolSmartContract) getLockConfig(_ context.Context, _ url.Values, balances c_state.StateContextI) (interface{}, error) {
	return ip.getGlobalNode(balances, "updateVariables"), nil
}
