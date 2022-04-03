package zcnsc_test

import (
	"testing"

	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
)

func Test_WhenAuthorizerExists_StakePool_IsCreated(t *testing.T) {
	const authorizerID = "auth0"

	// Default authorizer transaction
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()

	// Add authorizer
	tr := CreateAddAuthorizerTransaction(authorizerID, ctx)
	resp, err := contract.AddAuthorizer(tr, CreateAuthorizerParamPayload(), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	// Check nodes state
	node, err := GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Add AddOrUpdateAuthorizerStakePool
	payloadFn := func() []byte {
		return CreateAuthorizerStakingPoolParamPayload(authorizerID)
	}
	tr = CreateTransaction(authorizerID, AddAuthorizerStakePool, payloadFn, ctx)
	resp, err = contract.AddOrUpdateAuthorizerStakePool(tr, payloadFn(), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
}

func Test_WhenAuthorizerDoesNotExists_StakePool_IsNotUpdatedOrCreated(t *testing.T) {
	const authorizerID = "auth0"

	// Default authorizer transaction
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()

	// Add AddOrUpdateAuthorizerStakePool
	payloadFn := func() []byte {
		return CreateAuthorizerStakingPoolParamPayload(authorizerID)
	}
	tr := CreateTransaction(authorizerID, AddAuthorizerStakePool, payloadFn, ctx)
	resp, err := contract.AddOrUpdateAuthorizerStakePool(tr, payloadFn(), ctx)
	require.Error(t, err)
	require.EqualError(t, err, "authorizer(authorizerID: "+authorizerID+") not found")
	require.Empty(t, resp)
}
