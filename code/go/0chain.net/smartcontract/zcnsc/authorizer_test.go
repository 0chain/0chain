package zcnsc_test

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/stakepool"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	stringEmpty = ""
)

func init() {
	rand.Seed(time.Now().UnixNano())
	chain.ServerChain = new(chain.Chain)

	chain.ServerChain.Config = chain.NewConfigImpl(&chain.ConfigData{ClientSignatureScheme: "bls0chain"})
	logging.Logger = zap.NewNop()
}

func Test_BasicAuthorizersShouldBeInitialized(t *testing.T) {
	ctx := MakeMockStateContext()
	for _, authorizerKey := range authorizersID {
		node := &AuthorizerNode{ID: authorizerKey}
		err := ctx.GetTrieNode(node.GetKey(), node)
		require.NoError(t, err)
	}
}

func Test_Basic_GetGlobalNode_InitsNode(t *testing.T) {
	ctx := MakeMockStateContext()

	node, err := GetGlobalSavedNode(ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, ADDRESS+":globalnode:"+node.ID, node.GetKey())
}

func Test_Basic_GetUserNode_ReturnsUserNode(t *testing.T) {
	ctx := MakeMockStateContext()

	clientID := clients[0]

	node, err := GetUserNode(clientID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, clientID, node.ID)
	key := node.GetKey()
	require.Equal(t, ADDRESS+":usernode:"+clientID, key)
}

func Test_AddingDuplicateAuthorizerShouldFail(t *testing.T) {
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()
	tr := CreateAddAuthorizerTransaction("auth0", ctx)

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

func TestAuthorizerNodeShouldBeAbleToAddTransfer(t *testing.T) {
	sc := MakeMockStateContext()
	an := NewAuthorizer("id", "public key", "https://localhost:9876", nil)
	tr := CreateDefaultTransactionToZcnsc()

	var transfer *state.Transfer
	transfer, resp, err := an.LockingPool.DigPool(tr.Hash, tr)

	require.NoError(t, err, "must be able to dig pool")
	require.NotNil(t, transfer)
	require.NotNil(t, resp)

	// LockingPool params
	require.Equal(t, an.LockingPool.ID, tr.Hash)
	require.Equal(t, an.LockingPool.Balance, state.Balance(tr.Value))

	// Transfer params
	transferDigPoolEqualityCheck(t, transfer, tr)
	// Response params
	responseDigPoolEqualityCheck(t, resp, tr, &an.LockingPool.ID, &stringEmpty)

	err = sc.AddTransfer(transfer)
	require.NoError(t, err, "must be able to add transfer")
}

func TestAuthorizerNodeShouldBeAbleToDigPool(t *testing.T) {
	an := NewAuthorizer("id", "public key", "https://localhost:9876", nil)
	tr := CreateDefaultTransactionToZcnsc()

	var transfer *state.Transfer
	transfer, resp, err := an.LockingPool.DigPool(tr.Hash, tr)

	require.NoError(t, err, "must be able to dig pool")
	require.NotNil(t, transfer)
	require.NotNil(t, resp)

	// Transfer params
	transferDigPoolEqualityCheck(t, transfer, tr)
	// Response params
	responseDigPoolEqualityCheck(t, resp, tr, &an.LockingPool.ID, &stringEmpty)
}

func Test_BasicShouldAddAuthorizer(t *testing.T) {
	ctx := MakeMockStateContext()

	param := CreateAuthorizerParam()
	data, _ := param.Encode()
	sc := CreateZCNSmartContract()
	authorizerID := authorizersID[0] + ":10"
	tr := CreateAddAuthorizerTransaction(authorizerID, ctx)

	resp, err := sc.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	authorizeNode, err := GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)

	err = ctx.GetTrieNode(authorizeNode.GetKey(), authorizeNode)
	require.NoError(t, err)
}

func Test_Should_AddOnlyOneAuthorizerWithSameID(t *testing.T) {
	authorizerID := authorizersID[0] + time.Now().String()
	param := CreateAuthorizerParam()
	data, _ := param.Encode()
	sc := CreateZCNSmartContract()
	ctx := MakeMockStateContext()
	tr := CreateAddAuthorizerTransaction(authorizerID, ctx)

	address, err := sc.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err, "must be able to add authorizer")
	require.NotEmpty(t, address)

	// Check nodes state
	node, err := GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Try adding one more authorizer
	address, err = sc.AddAuthorizer(tr, data, ctx)
	require.Error(t, err, "must be able to add only one authorizer")
	require.Contains(t, err.Error(), "already exists")
	require.Empty(t, address)

	// Check nodes state
	node, err = GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
}

func Test_Basic_ShouldSaveGlobalNode(t *testing.T) {
	ctx := MakeMockStateContext()

	globalNode, err := GetGlobalSavedNode(ctx)
	require.NoError(t, err)
	require.Equal(t, state.Balance(11), globalNode.MinStakeAmount)

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = state.Balance(100 * 1e10)

	err = node.Save(ctx)
	require.NoError(t, err)

	globalNode, err = GetGlobalSavedNode(ctx)
	require.NoError(t, err)
	require.Equal(t, state.Balance(100*1e10), globalNode.MinStakeAmount)
}

func TestShould_Fail_If_TransactionValue_Less_Then_GlobalNode_MinStake(t *testing.T) {
	ctx := MakeMockStateContext()
	au := AuthorizerNode{ID: authorizersID[0]}
	authParam := AuthorizerParameter{
		PublicKey: ctx.authorizers[au.GetKey()].Node.PublicKey,
		URL:       "hhh",
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  "100",
			MinStake:        100,
			MaxStake:        100,
			MaxNumDelegates: 100,
			ServiceCharge:   100,
		},
	}
	data, _ := authParam.Encode()

	sc := CreateZCNSmartContract()

	client := defaultAuthorizer + time.Now().String()
	tr := CreateAddAuthorizerTransaction(client, MakeMockStateContext())
	tr.Value = 99

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = state.Balance(100 * 1e10)
	err := node.Save(ctx)
	require.NoError(t, err)

	resp, err := sc.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Empty(t, resp)
	require.Contains(t, err.Error(), "min stake amount")
}

func Test_Should_FailWithoutInputData(t *testing.T) {
	ctx := MakeMockStateContext()

	var data []byte
	tr := CreateAddAuthorizerTransaction("client0", ctx)
	tr.PublicKey = ""
	sc := CreateZCNSmartContract()

	resp, err := sc.AddAuthorizer(tr, data, ctx)
	require.Empty(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "input data is nil")
}

func Test_Transaction_Or_InputData_MustBe_A_Key_InputData(t *testing.T) {
	ctx := MakeMockStateContext()

	pk := &AuthorizerParameter{
		PublicKey: "public Key",
		URL:       "https://localhost:9876",
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  "100",
			MinStake:        100,
			MaxStake:        100,
			MaxNumDelegates: 100,
			ServiceCharge:   100,
		},
	}
	data, _ := json.Marshal(pk)
	tr := CreateAddAuthorizerTransaction("client0", ctx)
	tr.PublicKey = ""
	sc := CreateZCNSmartContract()

	resp, err := sc.AddAuthorizer(tr, data, ctx)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func Test_Cannot_Delete_AuthorizerFromAnotherClient(t *testing.T) {
	ctx := MakeMockStateContext()
	authorizerID := authorizersID[0]

	var data []byte
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx)
	sc := CreateZCNSmartContract()

	node, err := GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	tr = CreateAddAuthorizerTransaction("another client", ctx)

	authorizer, err := sc.DeleteAuthorizer(tr, data, ctx)
	require.Empty(t, authorizer)
	require.Error(t, err)
}

func Test_LockingBasicLogicTest(t *testing.T) {
	tr := CreateDefaultTransactionToZcnsc()
	z := &tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "0",
				Balance: 0,
			},
		},
		TokenLockInterface: &TokenLock{
			StartTime: common.Now(),
			Duration:  0,
		},
	}

	locked := z.IsLocked(tr)
	require.Equal(t, locked, true)
}

func Test_Can_DigPool(t *testing.T) {
	tr := CreateDefaultTransactionToZcnsc()
	an := NewAuthorizer(tr.ClientID, "key", "https://localhost:9876", nil)
	_, _, err := an.LockingPool.DigPool(tr.Hash, tr)
	require.NoError(t, err)
}

func Test_Can_EmptyPool(t *testing.T) {
	balances := MakeMockStateContext()
	tr := CreateDefaultTransactionToZcnsc()
	gn, err := GetGlobalNode(balances)

	an := NewAuthorizer(tr.ClientID, "key", "https://localhost:9876", nil)

	_, _, _ = an.LockingPool.DigPool(tr.Hash, tr)
	_, _, err = an.LockingPool.EmptyPool(gn.ID, tr.ClientID, tr)

	require.NoError(t, err)
}

func TestAuthorizerNodeShouldBeDecodedWithStakingPool(t *testing.T) {
	tr := CreateDefaultTransactionToZcnsc()
	node := NewAuthorizer(tr.ClientID, tr.PublicKey, "https://localhost:9876", nil)
	require.NotNil(t, node.LockingPool.TokenLockInterface)

	newNode := &AuthorizerNode{}
	err := newNode.Decode(node.Encode())
	require.NoError(t, err)
	require.NotNil(t, newNode.LockingPool.TokenLockInterface)
}

// With this test, the ability to Save nodes to context is tested
//func Test_GetAuthorizerNodes_ShouldBeAbleToReturnNodes(t *testing.T) {
//	ctx := MakeMockStateContext()
//
//	tr := CreateAddAuthorizerTransaction("client0", 10)
//	an := NewAuthorizer(tr.ClientID, tr.PublicKey, "https://localhost:9876")
//	err = ans.AddAuthorizer(an)
//	require.NoError(t, err)
//	require.NotNil(t, an.LockingPool.TokenLockInterface)
//
//	node := ans.NodeMap[tr.ClientID]
//	require.NotNil(t, node)
//	require.NotNil(t, node.LockingPool.TokenLockInterface)
//
//	// without saving, it won't be possible to get nodes
//	err = ans.Save(ctx)
//	require.NoError(t, err)
//
//	ans2, err := GetAuthorizerNodes(ctx)
//	require.NoError(t, err)
//	node = ans2.NodeMap[tr.ClientID]
//	require.NotNil(t, node)
//	require.NotNil(t, node.LockingPool.TokenLockInterface)
//}

func Test_NewAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	ctx := MakeMockStateContext()

	// Init
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx)
	node := NewAuthorizer(tr.ClientID, tr.PublicKey, "https://localhost:9876", nil)
	require.NotNil(t, node.LockingPool.TokenLockInterface)

	// Add
	err := node.Save(ctx)
	require.NoError(t, err)

	// FillFromContext
	newNode := &AuthorizerNode{}
	err = ctx.GetTrieNode(node.GetKey(), newNode)
	require.NoError(t, err)

	require.NotNil(t, newNode)
	require.NotNil(t, newNode.LockingPool.TokenLockInterface)
}

func Test_AddedAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	ctx := MakeMockStateContext()

	// Init
	var data []byte
	tr := CreateDefaultTransactionToZcnsc()
	sc := CreateZCNSmartContract()

	// Add
	_, _ = sc.AddAuthorizer(tr, data, ctx)

	// FillFromContext
	node := GetAuthorizerNodeFromCtx(t, ctx, defaultAuthorizer)
	require.NotNil(t, node.LockingPool.TokenLockInterface)
}

func Test_UpdateAuthorizerSettings(t *testing.T) {
	ctx := MakeMockStateContext()

	// Init
	var data []byte
	tr := CreateDefaultTransactionToZcnsc()
	sc := CreateZCNSmartContract()

	// Add
	_, _ = sc.AddAuthorizer(tr, data, ctx)

	// Get node and change its setting
	node := GetAuthorizerNodeFromCtx(t, ctx, defaultAuthorizer)
	require.NotNil(t, node)

	cfg := &AuthorizerConfig{
		Fee: state.Balance(111),
	}

	err := node.UpdateConfig(cfg)
	require.NoError(t, err)
	err = node.Save(ctx)
	require.NoError(t, err)

	// Get node and check its setting
	node = GetAuthorizerNodeFromCtx(t, ctx, defaultAuthorizer)
	require.NotNil(t, node.Config)
	require.Equal(t, state.Balance(111), node.Config.Fee)
}

func GetAuthorizerNodeFromCtx(t *testing.T, ctx cstate.StateContextI, key string) *AuthorizerNode {
	node, err := GetAuthorizerNode(key, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	return node
}

func Test_Can_Delete_Authorizer(t *testing.T) {
	var (
		ctx  = MakeMockStateContext()
		data []byte
	)

	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx)
	resp, err := sc.DeleteAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	authorizerNode, err := GetAuthorizerNode(defaultAuthorizer, ctx)
	require.Error(t, err)
	require.Nil(t, authorizerNode)
}

func Test_Authorizer_With_EmptyPool_Cannot_Be_Deleted(t *testing.T) {
	var (
		ctx          = MakeMockStateContext()
		data         []byte
		authorizerID = authorizersID[0]
	)

	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx)

	node := GetAuthorizerNodeFromCtx(t, ctx, authorizerID)
	_, _, err := node.LockingPool.EmptyPool(ADDRESS, tr.ClientID, tr)
	require.NoError(t, err)

	resp, err := sc.DeleteAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
}

func Test_Authorizer_EmptyPool_SimpleTest_Transfer(t *testing.T) {
	var (
		ctx          = MakeMockStateContext()
		authorizerID = authorizersID[0]
	)

	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx)

	node := GetAuthorizerNodeFromCtx(t, ctx, authorizerID)

	gn, err := GetGlobalNode(ctx)
	transfer, resp, err := node.LockingPool.EmptyPool(gn.ID, tr.ClientID, tr)
	require.NoError(t, err)

	transferEmptyPoolEqualityCheck(t, transfer, tr)
	responseEmptyPoolEqualityCheck(t, resp, tr, &stringEmpty, &tr.Hash)
}

func getResponse(t *testing.T, resp string) *tokenpool.TokenPoolTransferResponse {
	response := &tokenpool.TokenPoolTransferResponse{}
	err := response.Decode([]byte(resp))
	require.NoError(t, err, "failed to decode response")
	return response
}

func responseEmptyPoolEqualityCheck(t *testing.T, resp string, tr *transaction.Transaction, toPoolID, fromPoolID *string) {
	require.NotEmpty(t, resp)
	response := getResponse(t, resp)
	require.Equal(t, tr.Value, int64(response.Value))
	require.Equal(t, tr.ClientID, response.ToClient)
	require.Equal(t, tr.ToClientID, response.FromClient)
	if toPoolID != nil {
		require.Equal(t, *toPoolID, response.ToPool)
	}
	if fromPoolID != nil {
		require.Equal(t, *fromPoolID, response.FromPool)
	}
	require.Equal(t, stringEmpty, response.TxnHash)
}

func responseDigPoolEqualityCheck(t *testing.T, resp string, tr *transaction.Transaction, toPoolID, fromPoolID *string) {
	require.NotEmpty(t, resp)
	response := getResponse(t, resp)
	require.Equal(t, tr.Value, int64(response.Value))
	require.Equal(t, tr.ClientID, response.FromClient)
	require.Equal(t, tr.ToClientID, response.ToClient)
	if toPoolID != nil {
		require.Equal(t, *toPoolID, response.ToPool)
	}
	if fromPoolID != nil {
		require.Equal(t, *fromPoolID, response.FromPool)
	}
	require.Equal(t, tr.Hash, response.TxnHash)
}

func transferEmptyPoolEqualityCheck(t *testing.T, transfer *state.Transfer, tr *transaction.Transaction) {
	require.Equal(t, tr.ClientID, transfer.ToClientID)
	require.Equal(t, tr.ToClientID, transfer.ClientID)
	require.Equal(t, state.Balance(tr.Value), transfer.Amount)
}

func transferDigPoolEqualityCheck(t *testing.T, transfer *state.Transfer, tr *transaction.Transaction) {
	require.Equal(t, tr.ClientID, transfer.ClientID)
	require.Equal(t, tr.ToClientID, transfer.ToClientID)
	require.Equal(t, state.Balance(tr.Value), transfer.Amount)
}
