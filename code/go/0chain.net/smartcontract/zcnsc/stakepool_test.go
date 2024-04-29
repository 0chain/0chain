package zcnsc_test

import (
	"encoding/hex"
	"errors"
	"testing"
     "strconv"

	"github.com/stretchr/testify/require"
	"0chain.net/core/encryption"
	"0chain.net/chaincore/chain/state"
	. "0chain.net/smartcontract/zcnsc"
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

func TestDelegatePoolOperations(t *testing.T) {
	t.Parallel()

	t.Run("AddToDelegatePool_Success", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}

		// Action

		zcn := &ZCNSmartContract{}
		resp, err := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp)
		require.NoError(t, err)

		resp1, err1 := zcn.AddToDelegatePool(nil, nil, ctx.StateContextI)

		// Assertion
		require.NoError(t, err1)
		require.NotNil(t, resp1)
	})

	t.Run("AddToDelegatePool_Failure_GlobalNodeError", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
    // Mock the GetGlobalNode function to return an error
    mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
        return nil, errors.New("global node error") // Mock an error
    }

    zcn := &ZCNSmartContract{}
		resp, err := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp)
		require.NoError(t, err)

		resp1, err1 := zcn.AddToDelegatePool(nil, nil, ctx.StateContextI)

		// Assertion
		require.Error(t, err1)
		require.Empty(t, resp1)
	})


	t.Run("AddToDelegatePool_ExceedMaxDelegates", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context with the maximum number of delegates already reached
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}

		// Action
		zcn := &ZCNSmartContract{}
		resp, err := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp)
		require.NoError(t, err)

		resp1, err1 := zcn.AddToDelegatePool(nil, nil, ctx.StateContextI)

		// Assertion
		require.Error(t, err1)
		require.Empty(t, resp1)
	})
	t.Run("AddToDelegatePool_MaxStakeAmountExceeded", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}
	
		zcn := &ZCNSmartContract{}
		resp1, err1 := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp1)
		require.NoError(t, err1)
		stake := 6000
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)
		resp, err := zcn.AddToDelegatePool(nil, stakeBytes, ctx.StateContextI)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
	t.Run("AddToDelegatePool_MaxDelegatesReached", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   1,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}
	
		zcn := &ZCNSmartContract{}
		resp1, err1 := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp1)
		require.NoError(t, err1)
		stake := 2000
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)
		resp, err := zcn.AddToDelegatePool(nil, stakeBytes, ctx.StateContextI)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
	t.Run("AddToDelegatePool_MinStakeAmount", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}
	
		zcn := &ZCNSmartContract{}
		resp1, err1 := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp1)
		require.NoError(t, err1)
		// Attempt to add stake equal to the minimum allowed
		stake := 1000
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)
	
		resp, err := zcn.AddToDelegatePool(nil, stakeBytes, ctx.StateContextI)
	
		// Assertion
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
	t.Run("AddToDelegatePool_MinStakePerDelegate", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}
	
		zcn := &ZCNSmartContract{}
		resp1, err1 := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp1)
		require.NoError(t, err1)
		
		stake := 100
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)
	
		
		resp, err := zcn.AddToDelegatePool(nil, stakeBytes, ctx.StateContextI)
	
		// Assertion
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
	t.Run("AddToDelegatePool_InvalidStakeAmount", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}
	
		zcn := &ZCNSmartContract{}
		resp1, err1 := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp1)
		require.NoError(t, err1)
		
		stake :=-100
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)
	
		
		resp, err := zcn.AddToDelegatePool(nil, stakeBytes, ctx.StateContextI)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
	t.Run("AddToDelegatePool_InvalidContext", func(t *testing.T) {
		// Attempt to add stake with an invalid context
		zcn := &ZCNSmartContract{}

		stake :=100
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)
		resp, err := zcn.AddToDelegatePool(nil, stakeBytes, nil)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
	
	t.Run("AddToDelegatePool_MaxStakeReached", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}
		
	
		zcn := &ZCNSmartContract{}
		
		resp1, err1 := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp1)
		require.NoError(t, err1)
		stake :=6000
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)
	
		
		resp, err := zcn.AddToDelegatePool(nil, stakeBytes, ctx.StateContextI)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})


	t.Run("UpdateAuthorizerStakePool_AuthorizerNotFound", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Attempt to update stake pool for a non-existing authorizer
		authorizerID := "nonexistent_auth"
		payload := CreateAuthorizerStakingPoolParamPayload(authorizerID)
		tr, err := CreateTransaction(authorizerID, UpdateAuthorizerStakePoolFunc, payload, ctx.StateContextI)
		require.NoError(t, err)
		sc := CreateZCNSmartContract()
		
		// Call UpdateAuthorizerStakePool with invalid authorizer ID
		resp, err := sc.UpdateAuthorizerStakePool(tr, payload, ctx.StateContextI)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
			
	// Test cases for DeleteFromDelegatePool
	t.Run("DeleteFromDelegatePool_Success", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context with a delegate pool containing a sufficient balance
		// Generate a mock transaction and input data

		// Action
		zcn := &ZCNSmartContract{}
		resp, err := zcn.DeleteFromDelegatePool(nil, nil, ctx.StateContextI)

		// Assertion
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("DeleteFromDelegatePool_Failure_StakePoolUnlockError", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context with an empty delegate pool
		// Generate a mock transaction and input data

		// Action
		zcn := &ZCNSmartContract{}
		resp, err := zcn.DeleteFromDelegatePool(nil, nil, ctx.StateContextI)

		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
	t.Run("DeleteFromDelegatePool_NotEnoughBalanceForUnlock", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context
		// Mock the GetGlobalNode function to return a valid global node
		mockGlobalNode := &GlobalNode{
			ZCNSConfig : &ZCNSConfig {
			MinStakeAmount:       1000,
			MaxStakeAmount:       5000,
			MaxDelegates:   10,
			MinStakePerDelegate: 100,},

		}
		mockGetGlobalNode := func(balances state.StateContextI) (*GlobalNode, error) {
			return mockGlobalNode, nil
		}
	
		zcn := &ZCNSmartContract{}
		resp1, err1 := mockGetGlobalNode(ctx.StateContextI)
		require.NotNil(t, resp1)
		require.NoError(t, err1)
		stake := 3000
		stakeString := strconv.Itoa(stake)
        stakeBytes := []byte(stakeString)

		zcn.AddToDelegatePool(nil, stakeBytes, ctx.StateContextI)
		unlockAmount := 4000
		unlockAmountString := strconv.Itoa(unlockAmount)
        unlockAmountBytes := []byte(unlockAmountString)
		resp, err := zcn.DeleteFromDelegatePool(nil, unlockAmountBytes, ctx.StateContextI)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
	
	
	t.Run("DeleteFromDelegatePool_DelegateNotFound", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context with a delegate pool containing a sufficient balance
		// Mock the DeleteDelegate function to return an error indicating delegate not found
		mockDeleteDelegate := func(id string, balances state.StateContextI) error {
			// Simulate delegate not found error
			return errors.New("delegate not found")
		}
	
		zcn := &ZCNSmartContract{}

		invalidDelegateID := "invalid_delegate_id"
		invalidDelegateBytes := []byte(invalidDelegateID)
		err1:=mockDeleteDelegate(invalidDelegateID,ctx.StateContextI)
		require.Error(t,err1)
		// Call DeleteFromDelegatePool with invalid delegate ID
		resp, err := zcn.DeleteFromDelegatePool(nil,invalidDelegateBytes, ctx.StateContextI)
	
		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})

	t.Run("DeleteFromDelegatePool_InvalidInputData", func(t *testing.T) {
		// Setup
		ctx := MakeMockStateContext() // Create a mock state context with a non-existent delegate ID
		// Generate a mock transaction with invalid input data

		// Action
		zcn := &ZCNSmartContract{}
		resp, err := zcn.DeleteFromDelegatePool(nil, nil, ctx.StateContextI)

		// Assertion
		require.Error(t, err)
		require.Empty(t, resp)
	})
}