package zcnsc_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"0chain.net/core/encryption"
	"0chain.net/smartcontract/stakepool"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
)

func Test_WhenAuthorizerExists_StakePool_IsCreated(t *testing.T) {
	ctx := MakeMockStateContext()

	publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
	id := encryption.Hash(publicKeyBytes)

	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(ownerId, ctx)

	resp, err := sc.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	// Check nodes state
	node, err := GetAuthorizerNode(id, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Add UpdateAuthorizerStakePool
	payload := CreateAuthorizerStakingPoolParamPayload(id)
	tr, err = CreateTransaction(id, UpdateAuthorizerStakePoolFunc, payload, ctx)
	require.NoError(t, err)
	resp, err = sc.UpdateAuthorizerStakePool(tr, payload, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
}

func Test_UpdateAuthorizerStakePool(t *testing.T) {

	t.Run("StakePool_Without_ClientID", func(t *testing.T) {
		ctx := MakeMockStateContext()
		publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
		id := encryption.Hash(publicKeyBytes)
		sc := CreateZCNSmartContract()
		tr := CreateAddAuthorizerTransaction(ownerId, ctx)
		resp, err := sc.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		// Check nodes state
		node, err := GetAuthorizerNode(id, ctx)
		require.NoError(t, err)
		require.NotNil(t, node)
		// Add UpdateAuthorizerStakePool
		payload := CreateAuthorizerStakingPoolParamPayload(id)
		tr, err = CreateTransaction(id, UpdateAuthorizerStakePoolFunc, payload, ctx)
		tr.ClientID = ""
		require.NoError(t, err)
		resp, err = sc.UpdateAuthorizerStakePool(tr, payload, ctx)
		require.Error(t, err)
		require.EqualError(t, err, "update_authorizer_staking_pool_failed: tran.ClientID is empty")
		require.Empty(t, resp)
	})
	t.Run("StakePool_With_Empty_Payload", func(t *testing.T) {
		ctx := MakeMockStateContext()
		publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
		id := encryption.Hash(publicKeyBytes)
		sc := CreateZCNSmartContract()
		tr := CreateAddAuthorizerTransaction(ownerId, ctx)
		resp, err := sc.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		node, err := GetAuthorizerNode(id, ctx)
		require.NoError(t, err)
		require.NotNil(t, node)
		var payload []byte
		tr, err = CreateTransaction(id, UpdateAuthorizerStakePoolFunc, payload, ctx)
		require.NoError(t, err)
		resp, err = sc.UpdateAuthorizerStakePool(tr, payload, ctx)
		require.Error(t, err)
		require.EqualError(t, err, "update_authorizer_staking_pool_failed: input data is nil")
		require.Empty(t, resp)
	})

	t.Run("WhenAuthorizerDoesNotExist_StakePool_NotUpdatedOrCreated", func(t *testing.T) {
		const authorizerID = "auth0"
		ctx := MakeMockStateContext()
		contract := CreateZCNSmartContract()
		payload := CreateAuthorizerStakingPoolParamPayload(authorizerID)
		tr, err := CreateTransaction(authorizerID, UpdateAuthorizerStakePoolFunc, payload, ctx)
		require.NoError(t, err)
		resp, err := contract.UpdateAuthorizerStakePool(tr, payload, ctx)
		require.Error(t, err)
		require.EqualError(t, err, "authorizer(authorizerID: "+authorizerID+") not found")
		require.Empty(t, resp)
	})
}

func Test_DeleteFromDelegatePool(t *testing.T) {
	t.Run("StakePool_Value_NotPresent", func(t *testing.T) {
		ctx := MakeMockStateContext()
		publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
		id := encryption.Hash(publicKeyBytes)
		sc := CreateZCNSmartContract()
		tr := CreateAddAuthorizerTransaction(ownerId, ctx)
		resp, err := sc.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		node, err2 := GetAuthorizerNode(id, ctx)
		require.NoError(t, err2)
		require.NotNil(t, node)

		payload := CreateAuthorizerStakingPoolParamPayload(id)
		tr, err = CreateTransaction(id, DeleteFromDelegatePoolFunc, payload, ctx)
		require.NoError(t, err)
		resp, err1 := sc.DeleteFromDelegatePool(tr, payload, ctx)
		require.Error(t, err1)
		require.EqualError(t, err1, "stake_pool_unlock_failed: can't get related stake pool: value not present")
		require.Empty(t, resp)
	})
	t.Run("NoDelegatePool_For_ClientID", func(t *testing.T) {
		ctx := MakeMockStateContext()
		contract := CreateZCNSmartContract()
		publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
		authorizerID := encryption.Hash(publicKeyBytes)
		auth := new(Authorizer)
		auth.Node = NewAuthorizerNode("provider")
		ctx.authorizers["authorizer:stakepool:provider"] = auth
		ctx.globalNode.MaxStakeAmount = 100000000000
		payload := stakepool.StakePoolRequest{
			ProviderID:   "provider",
			ProviderType: 50,
		}
		en := payload.Encode()
		tr := CreateAddAuthorizerTransaction(ownerId, ctx)
		resp, err := contract.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		tr, err = CreateTransaction(authorizerID, DeleteFromDelegatePoolFunc, en, ctx)
		require.NoError(t, err)
		resp, err = contract.DeleteFromDelegatePool(tr, en, ctx)
		require.Error(t, err)
		str := "stake_pool_unlock_failed: no such delegate pool: " + ownerId
		require.Equal(t, strings.TrimSpace(err.Error()), str)
		require.Empty(t, resp)
	})

}

func Test_CollectRewards_ZeroReward(t *testing.T) {
	ctx := MakeMockStateContext()
	publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
	id := encryption.Hash(publicKeyBytes)
	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(ownerId, ctx)
	resp, err := sc.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	node, err2 := GetAuthorizerNode(id, ctx)
	require.NoError(t, err2)
	require.NotNil(t, node)
	payload := stakepool.CollectRewardRequest{
		ProviderId:   "provider",
		ProviderType: 50,
	}
	encoded_payload := payload.Encode()
	auth := new(Authorizer)
	auth.Node = NewAuthorizerNode("provider")
	ctx.authorizers["authorizer:stakepool:provider"] = auth
	tr, err = CreateTransaction(id, CollectRewardsFunc, encoded_payload, ctx)
	require.NoError(t, err)
	resp, err1 := sc.CollectRewards(tr, encoded_payload, ctx)
	require.Error(t, err1)
	str := "pay_reward_failed: cannot find rewards for " + ownerId
	require.EqualError(t, err1, str)
	require.Empty(t, resp)
}

func Test_AddToDelegatePool(t *testing.T) {

	t.Run("StakePool_Value_NotPresent", func(t *testing.T) {
		ctx := MakeMockStateContext()
		publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
		id := encryption.Hash(publicKeyBytes)
		sc := CreateZCNSmartContract()
		tr := CreateAddAuthorizerTransaction(ownerId, ctx)
		resp, err := sc.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		// Check nodes state
		node, err2 := GetAuthorizerNode(id, ctx)
		require.NoError(t, err2)
		require.NotNil(t, node)

		payload := CreateAuthorizerStakingPoolParamPayload(id)
		tr, err = CreateTransaction(id, AddToDelegatePoolFunc, payload, ctx)
		require.NoError(t, err)

		resp, err1 := sc.AddToDelegatePool(tr, payload, ctx)
		require.Error(t, err1)
		require.EqualError(t, err1, "stake_pool_lock_failed: can't get stake pool: value not present")
		require.Empty(t, resp)
	})
	t.Run("No_more_StakePoolsAllowed", func(t *testing.T) {
		ctx := MakeMockStateContext()
		contract := CreateZCNSmartContract()
		publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
		authorizerID := encryption.Hash(publicKeyBytes)
		auth := new(Authorizer)
		auth.Node = NewAuthorizerNode("provider")
		ctx.authorizers["authorizer:stakepool:provider"] = auth
		ctx.globalNode.MaxStakeAmount = 100000000000

		payload := stakepool.StakePoolRequest{
			ProviderID:   "provider",
			ProviderType: 50,
		}
		en := payload.Encode()
		tr := CreateAddAuthorizerTransaction(ownerId, ctx)
		resp, err := contract.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
		require.NoError(t, err)
		require.NotEmpty(t, resp)

		tr, err = CreateTransaction(authorizerID, AddToDelegatePoolFunc, en, ctx)
		require.NoError(t, err)

		resp, err = contract.AddToDelegatePool(tr, en, ctx)
		require.Error(t, err)
		require.EqualError(t, err, "stake_pool_lock_failed: max_delegates reached: 0, no more stake pools allowed")
		require.Empty(t, resp)
	})

	t.Run("Insufficient_MaxStakeAmount", func(t *testing.T) {
		ctx := MakeMockStateContext()
		contract := CreateZCNSmartContract()
		publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
		authorizerID := encryption.Hash(publicKeyBytes)
		auth := new(Authorizer)
		auth.Node = NewAuthorizerNode("provider")
		ctx.authorizers["authorizer:stakepool:provider"] = auth
		ctx.globalNode.MaxStakeAmount = 1000000

		payload := stakepool.StakePoolRequest{
			ProviderID:   "provider",
			ProviderType: 50,
		}
		en := payload.Encode()
		tr := CreateAddAuthorizerTransaction(ownerId, ctx)
		resp, err := contract.AddAuthorizer(tr, CreateAuthorizerParamPayload("random_authorizer_delegate_wallet", AuthorizerPublicKey), ctx)
		require.NoError(t, err)
		require.NotEmpty(t, resp)

		tr, err = CreateTransaction(authorizerID, AddToDelegatePoolFunc, en, ctx)
		require.NoError(t, err)

		resp, err = contract.AddToDelegatePool(tr, en, ctx)
		require.Error(t, err)
		require.EqualError(t, err, "stake_pool_lock_failed: too large stake to lock: 10000000000 > 1000000")
		require.Empty(t, resp)
	})
}
