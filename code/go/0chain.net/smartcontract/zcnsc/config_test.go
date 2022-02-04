package zcnsc_test

import (
	"fmt"
	"testing"

	"0chain.net/smartcontract/zcnsc"

	"github.com/stretchr/testify/require"
)

func TestConfigMap_Get(t *testing.T) {
	cfg := &zcnsc.ZCNSConfig{
		BurnAddress:        "0xBEEF",
		MinMintAmount:      100,
		PercentAuthorizers: 101,
		MinAuthorizers:     102,
		MinBurnAmount:      103,
		MinStakeAmount:     104,
		MaxFee:             105,
	}

	stringMap, err := cfg.ToStringMap()
	require.NoError(t, err)

	require.Contains(t, stringMap.Fields, "burn_address")
	require.Contains(t, stringMap.Fields, "min_mint_amount")
	require.Contains(t, stringMap.Fields, "percent_authorizers")
	require.Contains(t, stringMap.Fields, "min_authorizers")
	require.Contains(t, stringMap.Fields, "min_burn_amount")
	require.Contains(t, stringMap.Fields, "min_stake_amount")
	require.Contains(t, stringMap.Fields, "max_fee")

	require.Equal(t, fmt.Sprintf("%v", cfg.BurnAddress), stringMap.Fields["burn_address"])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinMintAmount), stringMap.Fields["min_mint_amount"])
	require.Equal(t, fmt.Sprintf("%v", cfg.PercentAuthorizers), stringMap.Fields["percent_authorizers"])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinAuthorizers), stringMap.Fields["min_authorizers"])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinBurnAmount), stringMap.Fields["min_burn_amount"])
	require.Equal(t, fmt.Sprintf("%v", cfg.MinStakeAmount), stringMap.Fields["min_stake_amount"])
	require.Equal(t, fmt.Sprintf("%v", cfg.MaxFee), stringMap.Fields["max_fee"])
}
