package zcnsc_test

import (
	"testing"

	"0chain.net/chaincore/state"

	"0chain.net/smartcontract/stakepool"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
)

func Test_StakingPoolSettingsCanBeCreatedOrUpdatedOnExistingAuthorizer(t *testing.T) {
	// Prerequisites: authorizer and stakepool exist
	// Test: update stake pool and verify

	const authorizerID = "auth0"

	// Default authorizer transaction
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(authorizerID, ctx)

	// Add authorizer
	resp, err := contract.AddAuthorizer(tr, CreateAuthorizerParamPayload(), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	// Check nodes state
	node, err := GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Check settings
	require.Equal(t, state.Balance(100), node.StakePoolSettings.MinStake)
	require.Equal(t, state.Balance(100), node.StakePoolSettings.MaxStake)
	require.Equal(t, 100, node.StakePoolSettings.MaxNumDelegates)
	require.Equal(t, float64(100), node.StakePoolSettings.ServiceCharge)
	require.Equal(t, "100", node.StakePoolSettings.DelegateWallet)

	// New update
	params := &AuthorizerParameter{
		PublicKey: tr.PublicKey,
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  "200",
			MinStake:        200,
			MaxStake:        200,
			MaxNumDelegates: 200,
			ServiceCharge:   200,
		},
	}

	data, _ := params.Encode()

	resp, err = contract.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Empty(t, resp)

	// Check nodes state
	node, err = GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Check settings
	require.Equal(t, state.Balance(200), node.StakePoolSettings.MinStake)
	require.Equal(t, state.Balance(200), node.StakePoolSettings.MaxStake)
	require.Equal(t, 200, node.StakePoolSettings.MaxNumDelegates)
	require.Equal(t, float64(200), node.StakePoolSettings.ServiceCharge)
	require.Equal(t, "200", node.StakePoolSettings.DelegateWallet)
}

func Test_WhenAuthorizerExists_StakePool_IsUpdated(t *testing.T) {
}

func Test_WhenAuthorizerExists_StakePool_IsCreated(t *testing.T) {
}

func Test_WhenAuthorizerDoesNotExists_StakePool_IsUpdatedOrCreated(t *testing.T) {
}
