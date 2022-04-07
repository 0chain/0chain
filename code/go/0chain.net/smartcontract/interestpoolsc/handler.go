package interestpoolsc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"0chain.net/smartcontract"

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

	for _, key := range CostFunctions {
		fields[fmt.Sprintf("cost.%s", key)] = fmt.Sprintf("%0v", gn.Cost[strings.ToLower(key)])
	}

	return &smartcontract.StringMap{
		Fields: fields,
	}, nil
}
