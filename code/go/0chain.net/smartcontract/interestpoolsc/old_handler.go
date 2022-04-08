package interestpoolsc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"0chain.net/smartcontract"

	"0chain.net/core/common"

	c_state "0chain.net/chaincore/chain/state"
)

func (ip *InterestPoolSmartContract) getConfig(_ context.Context, _ url.Values, balances c_state.StateContextI) (interface{}, error) {
	gn, err := ip.getGlobalNode(balances, "funcName")
	if err != nil {
		return nil, err
	}

	fields := map[string]string{
		Settings[MinLock]:       fmt.Sprintf("%0v", gn.MinLock),
		Settings[MaxMint]:       fmt.Sprintf("%0v", gn.MaxMint),
		Settings[MinLockPeriod]: fmt.Sprintf("%0v", gn.MinLockPeriod),
		Settings[Apr]:           fmt.Sprintf("%0v", gn.APR),
		Settings[OwnerId]:       fmt.Sprintf("%v", gn.OwnerId),
	}

	for _, key := range costFunctions {
		fields[fmt.Sprintf("cost.%s", key)] = fmt.Sprintf("%0v", gn.Cost[strings.ToLower(key)])
	}

	return &smartcontract.StringMap{
		Fields: fields,
	}, nil
}

func (ip *InterestPoolSmartContract) getPoolsStats(_ context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un, err := ip.getUserNode(params.Get("client_id"), balances)
	if err != nil {
		return nil, common.NewErrInternal("can't user node", err.Error())
	}

	if len(un.Pools) == 0 {
		return nil, common.NewErrNoResource("can't find user node")
	}

	t := time.Now()
	stats := &poolStats{}
	for _, pool := range un.Pools {
		stat, err := getPoolStats(pool, t)
		if err != nil {
			return nil, common.NewErrInternal("can't get pool stats", err.Error())
		}
		stats.addStat(stat)
	}
	return stats, nil
}

func (ip *InterestPoolSmartContract) getLockConfig(_ context.Context, _ url.Values, balances c_state.StateContextI) (interface{}, error) {
	return ip.getGlobalNode(balances, "updateVariables")
}
