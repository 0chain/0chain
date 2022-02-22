package zcnsc_test

import (
	"math/rand"

	cstate "0chain.net/chaincore/chain/state"

	"0chain.net/chaincore/chain"
	"0chain.net/core/logging"
	. "0chain.net/smartcontract/zcnsc"
	"go.uber.org/zap"

	"encoding/json"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/stretchr/testify/require"
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
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx, 10)

	publicKey := &AuthorizerParameter{PublicKey: tr.PublicKey}
	data, _ := publicKey.Encode()

	_, err := contract.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestAuthorizerNodeShouldBeAbleToAddTransfer(t *testing.T) {
	sc := MakeMockStateContext()
	an := CreateAuthorizer("id", "public key", "https://localhost:9876")
	tr := CreateDefaultTransactionToZcnsc()

	var transfer *state.Transfer
	transfer, resp, err := an.Staking.DigPool(tr.Hash, tr)

	require.NoError(t, err, "must be able to dig pool")
	require.NotNil(t, transfer)
	require.NotNil(t, resp)

	// Staking params
	require.Equal(t, an.Staking.ID, tr.Hash)
	require.Equal(t, an.Staking.Balance, state.Balance(tr.Value))

	// Transfer params
	transferDigPoolEqualityCheck(t, transfer, tr)
	// Response params
	responseDigPoolEqualityCheck(t, resp, tr, &an.Staking.ID, &stringEmpty)

	err = sc.AddTransfer(transfer)
	require.NoError(t, err, "must be able to add transfer")
}

func TestAuthorizerNodeShouldBeAbleToDigPool(t *testing.T) {
	an := CreateAuthorizer("id", "public key", "https://localhost:9876")
	tr := CreateDefaultTransactionToZcnsc()

	var transfer *state.Transfer
	transfer, resp, err := an.Staking.DigPool(tr.Hash, tr)

	require.NoError(t, err, "must be able to dig pool")
	require.NotNil(t, transfer)
	require.NotNil(t, resp)

	// Transfer params
	transferDigPoolEqualityCheck(t, transfer, tr)
	// Response params
	responseDigPoolEqualityCheck(t, resp, tr, &an.Staking.ID, &stringEmpty)
}

func Test_BasicShouldAddAuthorizer(t *testing.T) {
	ctx := MakeMockStateContext()

	param := CreateAuthorizerParam()
	data, _ := param.Encode()
	sc := CreateZCNSmartContract()
	authorizerID := authorizersID[0] + ":10"
	tr := CreateAddAuthorizerTransaction(authorizerID, ctx, 10)

	resp, err := sc.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	authorizeNode, _ := GetAuthorizerNode(authorizerID, ctx)

	err = ctx.GetTrieNode(authorizeNode.GetKey(), authorizeNode)
	require.NoError(t, err)
}

func Test_Should_AddOnlyOneAuthorizerWithSameID(t *testing.T) {
	authorizerID := authorizersID[0] + time.Now().String()
	param := CreateAuthorizerParam()
	data, _ := param.Encode()
	sc := CreateZCNSmartContract()
	ctx := MakeMockStateContext()
	tr := CreateAddAuthorizerTransaction(authorizerID, ctx, 10)

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
	require.Contains(t, err.Error(), "failed to add authorizer")
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
	require.Equal(t, int64(11), globalNode.MinStakeAmount)

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = int64(100)

	err = node.Save(ctx)
	require.NoError(t, err)

	globalNode, err = GetGlobalSavedNode(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(100), globalNode.MinStakeAmount)
}

func TestShould_Fail_If_TransactionValue_Less_Then_GlobalNode_MinStake(t *testing.T) {
	ctx := MakeMockStateContext()
	au := AuthorizerNode{ID: authorizersID[0]}
	authParam := AuthorizerParameter{
		PublicKey: ctx.authorizers[au.GetKey()].Node.PublicKey,
		URL:       "hhh",
	}
	data, _ := authParam.Encode()

	sc := CreateZCNSmartContract()

	client := defaultAuthorizer + time.Now().String()
	tr := CreateAddAuthorizerTransaction(client, MakeMockStateContext(), 10)
	tr.Value = 99

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = 100
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
	tr := CreateAddAuthorizerTransaction("client0", ctx, 10)
	tr.PublicKey = ""
	sc := CreateZCNSmartContract()

	resp, err := sc.AddAuthorizer(tr, data, ctx)
	require.Empty(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "input data is nil")
}

func Test_Transaction_Or_InputData_MustBe_A_Key_InputData(t *testing.T) {
	ctx := MakeMockStateContext()

	pk := &AuthorizerParameter{PublicKey: "public Key", URL: "https://localhost:9876"}
	data, _ := json.Marshal(pk)
	tr := CreateAddAuthorizerTransaction("client0", ctx, 10)
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
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx, 10)
	sc := CreateZCNSmartContract()

	node, err := GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	tr = CreateAddAuthorizerTransaction("another client", ctx, 10)

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
	an := CreateAuthorizer(tr.ClientID, "key", "https://localhost:9876")
	_, _, err := an.Staking.DigPool(tr.Hash, tr)
	require.NoError(t, err)
}

func Test_Can_EmptyPool(t *testing.T) {
	balances := MakeMockStateContext()
	tr := CreateDefaultTransactionToZcnsc()
	gn, err := GetGlobalNode(balances)

	an := CreateAuthorizer(tr.ClientID, "key", "https://localhost:9876")

	_, _, _ = an.Staking.DigPool(tr.Hash, tr)
	_, _, err = an.Staking.EmptyPool(gn.ID, tr.ClientID, tr)

	require.NoError(t, err)
}

func TestAuthorizerNodeShouldBeDecodedWithStakingPool(t *testing.T) {
	tr := CreateDefaultTransactionToZcnsc()
	node := CreateAuthorizer(tr.ClientID, tr.PublicKey, "https://localhost:9876")
	require.NotNil(t, node.Staking.TokenLockInterface)

	newNode := &AuthorizerNode{}
	err := newNode.Decode(node.Encode())
	require.NoError(t, err)
	require.NotNil(t, newNode.Staking.TokenLockInterface)
}

// With this test, the ability to Save nodes to context is tested
//func Test_GetAuthorizerNodes_ShouldBeAbleToReturnNodes(t *testing.T) {
//	ctx := MakeMockStateContext()
//
//	tr := CreateAddAuthorizerTransaction("client0", 10)
//	an := CreateAuthorizer(tr.ClientID, tr.PublicKey, "https://localhost:9876")
//	err = ans.AddAuthorizer(an)
//	require.NoError(t, err)
//	require.NotNil(t, an.Staking.TokenLockInterface)
//
//	node := ans.NodeMap[tr.ClientID]
//	require.NotNil(t, node)
//	require.NotNil(t, node.Staking.TokenLockInterface)
//
//	// without saving, it won't be possible to get nodes
//	err = ans.Save(ctx)
//	require.NoError(t, err)
//
//	ans2, err := GetAuthorizerNodes(ctx)
//	require.NoError(t, err)
//	node = ans2.NodeMap[tr.ClientID]
//	require.NotNil(t, node)
//	require.NotNil(t, node.Staking.TokenLockInterface)
//}

func Test_NewAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	ctx := MakeMockStateContext()

	// Init
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx, 10)
	node := CreateAuthorizer(tr.ClientID, tr.PublicKey, "https://localhost:9876")
	require.NotNil(t, node.Staking.TokenLockInterface)

	// Add
	err := node.Save(ctx)
	require.NoError(t, err)

	// FillFromContext
	newNode := &AuthorizerNode{}
	err = ctx.GetTrieNode(node.GetKey(), newNode)
	require.NoError(t, err)

	//require.NotNil(t, blob)
	//newNode := &AuthorizerNode{}
	//err = newNode.Decode(blob.Encode())
	//require.NoError(t, err)
	require.NotNil(t, newNode)
	require.NotNil(t, newNode.Staking.TokenLockInterface)
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
	require.NotNil(t, node.Staking.TokenLockInterface)
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
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx, 10)
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
	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx, 10)

	node := GetAuthorizerNodeFromCtx(t, ctx, authorizerID)
	_, _, err := node.Staking.EmptyPool(ADDRESS, tr.ClientID, tr)
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

	tr := CreateAddAuthorizerTransaction(defaultAuthorizer, ctx, 100)

	node := GetAuthorizerNodeFromCtx(t, ctx, authorizerID)

	gn, err := GetGlobalNode(ctx)
	transfer, resp, err := node.Staking.EmptyPool(gn.ID, tr.ClientID, tr)
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
