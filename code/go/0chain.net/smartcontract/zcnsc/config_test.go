package zcnsc_test

import (
	"fmt"
	"strings"
	"testing"

	. "0chain.net/smartcontract/zcnsc"

	"github.com/stretchr/testify/require"
)

func TestConfigMap_Get(t *testing.T) {
	cfg := &GlobalNode{
		ID: "",
		ZCNSConfig: &ZCNSConfig{
			BurnAddress:        "0xBEEF",
			MinMintAmount:      100,
			PercentAuthorizers: 101,
			MinAuthorizers:     102,
			MinBurnAmount:      103,
			MinStakeAmount:     104,
			MaxFee:             105,
			OwnerId:            "106",
			Cost: map[string]int{
				MintFunc:             100,
				BurnFunc:             100,
				DeleteAuthorizerFunc: 100,
				AddAuthorizerFunc:    100,
			},
		},
	}

	stringMap := cfg.ToStringMap()

	require.Equal(t, 14, len(stringMap.Fields))
	require.Contains(t, stringMap.Fields, OwnerID)
	require.Contains(t, stringMap.Fields, MinBurnAmount)
	require.Contains(t, stringMap.Fields, MinMintAmount)
	require.Contains(t, stringMap.Fields, MinLockAmount)
	require.Contains(t, stringMap.Fields, MinAuthorizers)
	require.Contains(t, stringMap.Fields, MinStakeAmount)
	require.Contains(t, stringMap.Fields, MaxFee)
	require.Contains(t, stringMap.Fields, BurnAddress)
	require.Contains(t, stringMap.Fields, PercentAuthorizers)
	require.Contains(t, stringMap.Fields, MaxDelegates)

	for _, costFunction := range CostFunctions {
		require.Contains(t, stringMap.Fields, fmt.Sprintf("%s.%s", Cost, costFunction))
	}

	require.Equal(t, fmt.Sprintf("%v", cfg.OwnerId), stringMap.Fields[OwnerID])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinBurnAmount), stringMap.Fields[MinBurnAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinMintAmount), stringMap.Fields[MinMintAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinLockAmount), stringMap.Fields[MinLockAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinAuthorizers), stringMap.Fields[MinAuthorizers])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinStakeAmount), stringMap.Fields[MinStakeAmount])
	require.Equal(t, fmt.Sprintf("%v", cfg.MaxFee), stringMap.Fields[MaxFee])
	require.Equal(t, fmt.Sprintf("%v", cfg.BurnAddress), stringMap.Fields[BurnAddress])
	require.Equal(t, fmt.Sprintf("%v", cfg.PercentAuthorizers), stringMap.Fields[PercentAuthorizers])
	require.Equal(t, fmt.Sprintf("%v", cfg.MaxDelegates), stringMap.Fields[MaxDelegates])

	for _, costFunction := range CostFunctions {
		t.Log("expected key,  value:", costFunction, fmt.Sprintf("%d", cfg.Cost[strings.ToLower(costFunction)]))
		t.Log("actual key,  value:", costFunction, stringMap.Fields[fmt.Sprintf("%s.%s", Cost, costFunction)])
		require.Equal(t, fmt.Sprintf("%d", cfg.Cost[strings.ToLower(costFunction)]), stringMap.Fields[fmt.Sprintf("%s.%s", Cost, costFunction)])
	}
}
