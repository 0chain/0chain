package interestpoolsc

import (
	"context"
	"net/url"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"github.com/0chain/gosdk/core/common/errors"
)

func (ip *InterestPoolSmartContract) getPoolsStats(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un := ip.getUserNode(params.Get("client_id"), balances)
	if len(un.Pools) == 0 {
		return nil, common.NewErrNoResource(nil, "can't find user node")
	}
	t := time.Now()
	stats := &poolStats{}
	for _, pool := range un.Pools {
		stat, err := ip.getPoolStats(pool, t)
		if err != nil {
			return nil, common.NewErrInternal(err, "can't get pool stats")
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
		return nil, errors.Wrap(err, common.ErrDecoding)
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
