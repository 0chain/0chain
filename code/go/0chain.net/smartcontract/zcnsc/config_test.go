package zcnsc_test

import (
	"fmt"
	"strings"
	"testing"

	"0chain.net/smartcontract/zcnsc"

	"github.com/stretchr/testify/require"
)

func TestConfigMap_Get(t *testing.T) {
	cfg := &zcnsc.GlobalNode{
		ID: "",
		ZCNSConfig: &zcnsc.ZCNSConfig{
			BurnAddress:        "0xBEEF",
			MinMintAmount:      100,
			PercentAuthorizers: 101,
			MinAuthorizers:     102,
			MinBurnAmount:      103,
			MinStakeAmount:     104,
			MaxFee:             105,
			OwnerId:            "106",
			Cost: map[string]int{
				"mint":             100,
				"burn":             100,
				"DeleteAuthorizer": 100,
				"AddAuthorizer":    100,
			},
		},
	}

	stringMap := cfg.ToStringMap()

	require.Equal(t, 15, len(stringMap.Fields))
	require.Contains(t, stringMap.Fields, zcnsc.OwnerID)
	require.Contains(t, stringMap.Fields, zcnsc.MinBurnAmount)
	require.Contains(t, stringMap.Fields, zcnsc.MinMintAmount)
	require.Contains(t, stringMap.Fields, zcnsc.MinLockAmount)
	require.Contains(t, stringMap.Fields, zcnsc.MinAuthorizers)
	require.Contains(t, stringMap.Fields, zcnsc.MinStakeAmount)
	require.Contains(t, stringMap.Fields, zcnsc.MaxFee)
	require.Contains(t, stringMap.Fields, zcnsc.BurnAddress)
	require.Contains(t, stringMap.Fields, zcnsc.PercentAuthorizers)
	require.Contains(t, stringMap.Fields, zcnsc.Cost)
	require.Contains(t, stringMap.Fields, zcnsc.MaxDelegates)

	for _, costFunction := range zcnsc.CostFunctions {
		require.Contains(t, stringMap.Fields, fmt.Sprintf("%s.%s", zcnsc.Cost, costFunction))
	}

	require.Equal(t, fmt.Sprintf("%v", cfg.OwnerId), stringMap.Fields[zcnsc.OwnerID])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinBurnAmount), stringMap.Fields[zcnsc.MinBurnAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinMintAmount), stringMap.Fields[zcnsc.MinMintAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinLockAmount), stringMap.Fields[zcnsc.MinLockAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinAuthorizers), stringMap.Fields[zcnsc.MinAuthorizers])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinStakeAmount), stringMap.Fields[zcnsc.MinStakeAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MaxFee), stringMap.Fields[zcnsc.MaxFee])
	require.Equal(t, fmt.Sprintf("%v", cfg.BurnAddress), stringMap.Fields[zcnsc.BurnAddress])
	require.Equal(t, fmt.Sprintf("%v", cfg.PercentAuthorizers), stringMap.Fields[zcnsc.PercentAuthorizers])
	require.Equal(t, fmt.Sprintf("%v", cfg.Cost), stringMap.Fields[zcnsc.Cost])
	require.Equal(t, fmt.Sprintf("%v", cfg.MaxDelegates), stringMap.Fields[zcnsc.MaxDelegates])

	for _, costFunction := range zcnsc.CostFunctions {
		t.Log("expected key,  value:", costFunction, fmt.Sprintf("%d", cfg.Cost[strings.ToLower(costFunction)]))
		t.Log("actual key,  value:", costFunction, stringMap.Fields[fmt.Sprintf("%s.%s", zcnsc.Cost, costFunction)])
		require.Equal(t, fmt.Sprintf("%d", cfg.Cost[strings.ToLower(costFunction)]), stringMap.Fields[fmt.Sprintf("%s.%s", zcnsc.Cost, costFunction)])
	}
}
