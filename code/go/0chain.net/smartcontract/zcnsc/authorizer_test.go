package zcnsc_test

import (
	"0chain.net/chaincore/chain"
	"0chain.net/core/logging"
	. "0chain.net/smartcontract/zcnsc"
	"go.uber.org/zap"
	"math/rand"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var (
	stringEmpty = ""
)

func init() {
	rand.Seed(time.Now().UnixNano())
	chain.ServerChain = new(chain.Chain)
	chain.ServerChain.Config = new(chain.Config)
	chain.ServerChain.ClientSignatureScheme = "bls0chain"

	logging.Logger = zap.NewNop()
}

func Test_AuthorizersShouldBeInitialized(t *testing.T) {
	ctx := MakeMockStateContext()
	nodes, err := ctx.GetTrieNode(AllAuthorizerKey)
	require.NoError(t, err)
	require.NotNil(t, nodes)
	an := nodes.(*AuthorizerNodes)
	require.Equal(t, 3, len(an.NodeMap))
}

func Test_BasicAuthorizersShouldBeInitialized(t *testing.T) {
	ctx := MakeMockStateContext()
	nodes, err := GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	require.NotNil(t, nodes)
	require.Equal(t, 3, len(nodes.NodeMap))
}

func Test_Basic_GetGlobalNode_InitsNode(t *testing.T) {
	ctx := MakeMockStateContext()

	node, err := GetGlobalSavedNode(ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, ADDRESS, node.ID)
}

func Test_Basic_GetAuthorizerNode_InitsNode(t *testing.T) {
	ctx := MakeMockStateContext()

	nodes, err := GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	require.NotNil(t, nodes)
}

func Test_Basic_GetUserNode_ReturnsUserNode(t *testing.T) {
	ctx := MakeMockStateContext()

	node, err := GetUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.Equal(t, clientId, node.ID)
	require.Equal(t, ADDRESS+clientId, node.GetKey(ADDRESS))
}

func Test_AddingDuplicateAuthorizerShouldFail(t *testing.T) {
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()
	tr := CreateTransactionToZcnsc(clientId, 10)

	publicKey := &PublicKey{Key: tr.PublicKey}
	data, _ := publicKey.Encode()

	_, err := contract.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

//func Test_AddingAuthorizer_Adds_Transfers_To_Context(t *testing.T) {
//	ctx := MakeMockStateContext()
//	tr := CreateDefaultTransactionToZcnsc()
//
//	transfers := ctx.GetTransfers()
//	require.Equal(t, len(transfers), 1)
//
//	transfer := transfers[0]
//	require.Equal(t, int64(transfer.Amount), tr.Value)
//	require.Equal(t, transfer.ClientID, tr.ClientID)
//	require.Equal(t, transfer.ToClientID, tr.ToClientID)
//}

func Test_AuthorizersShouldNotBeInitializedWhenContextIsCreated(t *testing.T) {
	sc := MakeMockStateContext()
	ans, err := GetAuthorizerNodes(sc)
	require.NoError(t, err)
	require.NotNil(t, ans)
	require.Equal(t, 3, len(ans.NodeMap))
}

func TestAuthorizerNodeShouldBeAbleToAddTransfer(t *testing.T) {
	sc := MakeMockStateContext()
	an := GetNewAuthorizer("public key", "id")
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
	an := GetNewAuthorizer("public key", "id")
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
	var data []byte
	sc := CreateZCNSmartContract()
	ctx := MakeMockStateContext()
	tr := CreateTransactionToZcnsc("client0", 10)

	ans, err := GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	require.NotNil(t, ans)
	require.Equal(t, 3, len(ans.NodeMap))

	_, err = sc.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)

	ans, err = GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	require.NotNil(t, ans)
	require.Equal(t, 4, len(ans.NodeMap))
}

func Test_Should_AddOnlyOneAuthorizerWithSameID(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	balances := MakeMockStateContext()
	tr := CreateTransactionToZcnsc("client0", 10)

	address, err := sc.AddAuthorizer(tr, data, balances)
	require.NoError(t, err, "must be able to add authorizer")
	require.NotEmpty(t, address)

	// Check nodes state
	ans, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	node := ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.Equal(t, 4, len(ans.NodeMap))

	// Try adding one more authorizer
	address, err = sc.AddAuthorizer(tr, data, balances)
	require.Error(t, err, "must be able to add only one authorizer")
	require.Contains(t, err.Error(), "failed to add authorizer")
	require.Empty(t, address)

	// Check nodes state
	ans, err = GetAuthorizerNodes(balances)
	require.NoError(t, err)
	node = ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.Equal(t, 4, len(ans.NodeMap))
}

func Test_Basic_ShouldSaveGlobalNode(t *testing.T){
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
	var data []byte
	sc := CreateZCNSmartContract()
	balances := MakeMockStateContext()
	tr := CreateTransactionToZcnsc("client0", 10)
	tr.Value = 99

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = 100
	err := node.Save(balances)
	require.NoError(t, err)

	resp, err := sc.AddAuthorizer(tr, data, balances)
	require.Error(t, err)
	require.Empty(t, resp)
	require.Contains(t, err.Error(), "is lower than min amount")
}

func Test_Should_FailWithoutPublicKey(t *testing.T) {
	var data []byte
	tr := CreateTransactionToZcnsc("client0", 10)
	tr.PublicKey = ""
	balances := MakeMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.AddAuthorizer(tr, data, balances)
	require.Empty(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "public key was not included with transaction")
}

func Test_Transaction_Or_InputData_MustBe_A_Key_InputData(t *testing.T) {
	pk := &PublicKey{Key: "public Key"}
	data, _ := json.Marshal(pk)
	tr := CreateTransactionToZcnsc("client0", 10)
	tr.PublicKey = ""
	balances := MakeMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.AddAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func Test_Transaction_Or_InputData_MustBe_A_Key_Transaction(t *testing.T) {
	var data []byte
	tr := CreateTransactionToZcnsc("client0", 10)
	tr.PublicKey = "public Key"
	balances := MakeMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.AddAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func Test_Cannot_Delete_AuthorizerFromAnotherClient(t *testing.T) {
	var data []byte
	ctx := MakeMockStateContext()
	tr := CreateTransactionToZcnsc("client0", 10)
	sc := CreateZCNSmartContract()

	ans, err := GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	require.Nil(t, ans.NodeMap["client0"])

	tr = CreateTransactionToZcnsc("another client", 10)

	authorizer, err := sc.DeleteAuthorizer(tr, data, ctx)
	require.Empty(t, authorizer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "doesn't exist")
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
		TokenLockInterface: TokenLock{
			StartTime: common.Now(),
			Duration:  0,
		},
	}

	locked := z.IsLocked(tr)
	require.Equal(t, locked, true)
}

func Test_Can_DigPool(t *testing.T) {
	tr := CreateDefaultTransactionToZcnsc()
	an := GetNewAuthorizerWithBalance("key", tr.ClientID, 100)

	_, _, err := an.Staking.DigPool(tr.Hash, tr)
	require.NoError(t, err)
}

func Test_Can_EmptyPool(t *testing.T) {
	balances := MakeMockStateContext()
	tr := CreateDefaultTransactionToZcnsc()
	gn := GetGlobalNode(balances)

	an := GetNewAuthorizer("key", tr.ClientID)

	_, _, _ = an.Staking.DigPool(tr.Hash, tr)
	_, _, err := an.Staking.EmptyPool(gn.ID, tr.ClientID, tr)

	require.NoError(t, err)
}

func TestAuthorizerNodeShouldBeDecodedWithStakingPool(t *testing.T) {
	tr := CreateDefaultTransactionToZcnsc()
	an := GetNewAuthorizer(tr.PublicKey, tr.ClientID)
	require.NotNil(t, an.Staking.TokenLockInterface)

	ans := &AuthorizerNodes{}
	ans.NodeMap = make(map[string]*AuthorizerNode)
	ans.NodeMap[an.ID] = an

	ans2 := &AuthorizerNodes{}
	err := ans2.Decode(ans.Encode())
	require.NoError(t, err)

	node := ans2.NodeMap[an.ID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)
}

// With this test, the ability to Save nodes to context is tested
func Test_GetAuthorizerNodes_ShouldBeAbleToReturnNodes(t *testing.T) {
	ans := &AuthorizerNodes{}
	balances := MakeMockStateContext()
	av, err := balances.GetTrieNode(AllAuthorizerKey)
	if err != nil {
		ans.NodeMap = make(map[string]*AuthorizerNode)
	} else {
		// deep copy to the local context
		_ = ans.Decode(av.Encode())
	}

	require.NotNil(t, ans)

	tr := CreateTransactionToZcnsc("client0", 10)
	an := GetNewAuthorizer(tr.PublicKey, tr.ClientID)
	err = ans.AddAuthorizer(an)
	require.NoError(t, err)
	require.NotNil(t, an.Staking.TokenLockInterface)

	node := ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)

	// without saving it won't be possible to get nodes
	err = ans.Save(balances)
	require.NoError(t, err)

	ans2, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	node = ans2.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_NewAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	// Init
	tr := CreateTransactionToZcnsc("client0", 10)
	node := GetNewAuthorizer(tr.PublicKey, tr.ClientID)
	require.NotNil(t, node.Staking.TokenLockInterface)
	balances := MakeMockStateContext()

	// Add
	ans, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	_ = ans.AddAuthorizer(node)

	// Get
	node = ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)

	// Save
	err = ans.Save(balances)
	require.NoError(t, err)

	// Get nodes again from context
	ans2, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	node = ans2.NodeMap[tr.ClientID]
	require.NotNil(t, node)

	// Staking Pool must be initialized
	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_AddedAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	// Init
	var data []byte
	tr := CreateDefaultTransactionToZcnsc()
	sc := CreateZCNSmartContract()

	// Add
	balances := MakeMockStateContext()
	_, _ = sc.AddAuthorizer(tr, data, balances)

	// Get
	ans, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	node := ans.NodeMap[tr.ClientID]

	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_Can_Delete_Authorizer(t *testing.T) {
	var data []byte
	ctx := MakeMockStateContext()
	sc := CreateZCNSmartContract()

	ans, err := GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(ans.NodeMap))

	tr := CreateTransactionToZcnsc(authorizers[0], 10)
	_, err = sc.DeleteAuthorizer(tr, data, ctx)
	require.NoError(t, err)

	ans, err = GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(ans.NodeMap))
}

func Test_Authorizer_With_EmptyPool_Cannot_Be_Deleted(t *testing.T) {
	var data []byte
	ctx := MakeMockStateContext()
	sc := CreateZCNSmartContract()
	tr := CreateTransactionToZcnsc(authorizers[0], 10)

	ans, err := GetAuthorizerNodes(ctx)
	require.NoError(t, err)
	_, _, err = ans.NodeMap[authorizers[0]].Staking.EmptyPool(ADDRESS, tr.ClientID, tr)
	require.NoError(t, err)

	resp, err := sc.DeleteAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
}

func Test_Authorizer_EmptyPool_SimpleTest_Transfer(t *testing.T) {
	ctx := MakeMockStateContext()
	tr := CreateTransactionToZcnsc(authorizers[0], 10)

	ans, err := GetAuthorizerNodes(ctx)
	require.NoError(t, err)

	gn := GetGlobalNode(ctx)
	transfer, resp, err := ans.NodeMap[tr.ClientID].Staking.EmptyPool(gn.ID, tr.ClientID, tr)
	require.NoError(t, err)

	transferEmptyPoolEqualityCheck(t, transfer, tr)
	responseEmptyPoolEqualityCheck(t, resp, tr, &stringEmpty, &tr.Hash)
}

func Test_Authorizers_NodeMap_ShouldBeInitializedAfterSaving(t *testing.T) {
	// Create authorizers nodes tree
	balances := MakeMockStateContext()
	tree, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.NodeMap)

	err = tree.Save(balances)
	require.NoError(t, err)

	tree, err = GetAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.NodeMap)
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
