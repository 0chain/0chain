package zcnsc_test

import (
	"encoding/hex"
	// "errors"
	"testing"

	// "0chain.net/chaincore/chain/state"
	// "0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
	// "0chain.net/smartcontract/stakepool"
	// "0chain.net/smartcontract/stakepool/spenum"
	// "0chain.net/smartcontract/zcnsc"
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

func Test_WhenAuthorizerDoesNotExists_StakePool_IsNotUpdatedOrCreated(t *testing.T) {
	const authorizerID = "auth0"

	// Default authorizer transaction
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()

	// Add UpdateAuthorizerStakePool
	payload := CreateAuthorizerStakingPoolParamPayload(authorizerID)
	tr, err := CreateTransaction(authorizerID, UpdateAuthorizerStakePoolFunc, payload, ctx)
	require.NoError(t, err)
	resp, err := contract.UpdateAuthorizerStakePool(tr, payload, ctx)
	require.Error(t, err)
	require.EqualError(t, err, "authorizer(authorizerID: "+authorizerID+") not found")
	require.Empty(t, resp)
}
func Test_StakePoolLock_NoGetFuncProvided(t *testing.T) {
     // Mock necessary dependencies
	 const authorizerID = "auth0"
	 ctx:= MakeMockStateContext()
	//  tInput := &transaction.Transaction{} // Provide a valid transaction object
	//  input := []byte{}                    // Provide a valid input byte array
	//  bal := &mockStateContext{}      // Provide a mock implementation of StateContextI
	 
	 zcn := CreateZCNSmartContract() // Initialize ZCNSmartContract
     
	//  payload := CreateAuthorizerStakingPoolParamPayload(authorizerID)
    //  tr,err := CreateTransaction(authorizerID, UpdateAuthorizerStakePoolFunc, payload, ctx)
    //  require.NoError(t,err)
	//  resp,err := zcn.UpdateAuthorizerConfig(tr,payload,ctx)
	//  require.Error(t,err)
	//  require.EqualError(t, err, "authorizer(authorizerID: "+authorizerID+") not found")
	//  require.Empty(t,resp)
	 
    // Create a transaction for updating authorizer config
    payload := CreateAuthorizerStakingPoolParamPayload(authorizerID)
    tr, err := CreateTransaction(authorizerID, UpdateAuthorizerStakePoolFunc, payload, ctx)
    require.NoError(t, err)
    
    // Call the AddToDelegatePool function
    resp, err := zcn.UpdateAuthorizerConfig(tr, payload, ctx)
    
    // Check if an error occurred because no get function was provided
    require.Error(t, err)
    require.EqualError(t, err, "provide get func")
    require.Empty(t, resp)
}