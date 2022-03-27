package zcnsc_test

import (
	"testing"

	"0chain.net/smartcontract/stakepool"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
)

func Test_WhenAuthorizerExists_StakePool_IsUpdated(t *testing.T) {
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction("auth0", ctx, 10)

	params := &AuthorizerParameter{
		PublicKey: tr.PublicKey,
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  "100",
			MinStake:        100,
			MaxStake:        100,
			MaxNumDelegates: 100,
			ServiceCharge:   100,
		},
	}

	data, _ := params.Encode()

	_, err := contract.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)

	_, err = contract.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func Test_WhenAuthorizerExists_StakePool_IsCreated(t *testing.T) {
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction("auth0", ctx, 10)

	params := &AuthorizerParameter{
		PublicKey: tr.PublicKey,
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  "100",
			MinStake:        100,
			MaxStake:        100,
			MaxNumDelegates: 100,
			ServiceCharge:   100,
		},
	}
	data, _ := params.Encode()

	_, err := contract.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)

	_, err = contract.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func Test_WhenAuthorizerDoesNotExists_StakePool_IsUpdatedOrCreated(t *testing.T) {
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()
	tr := CreateAddAuthorizerTransaction("auth0", ctx, 10)

	params := &AuthorizerParameter{
		PublicKey: tr.PublicKey,
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  "100",
			MinStake:        100,
			MaxStake:        100,
			MaxNumDelegates: 100,
			ServiceCharge:   100,
		},
	}
	data, _ := params.Encode()

	_, err := contract.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)

	_, err = contract.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}
