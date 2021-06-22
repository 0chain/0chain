package zcnsc

import (
	"0chain.net/chaincore/chain"
	"0chain.net/core/logging"
	"go.uber.org/zap"
	"math/rand"

	//cstate "0chain.net/chaincore/chain/state"
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

func Test_TransferStateAfterAddingAuthorizer(t *testing.T) {
	sc := CreateZCNSmartContract()
	ctx := CreateMockStateContext(clientId)
	addAuthorizer(t, sc, ctx, clientId, "pk")
}

func Test_AddingAuthorizer_Adds_Transfers_To_Context(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	ctx := CreateMockStateContext(clientId)
	tr := CreateDefaultTransaction()

	address, err := sc.addAuthorizer(tr, data, ctx)
	require.NoError(t, err, "must be able to add authorizer")
	require.NotEmpty(t, address)

	transfers := ctx.GetTransfers()
	require.Equal(t, len(transfers), 1)

	transfer := transfers[0]
	require.Equal(t, int64(transfer.Amount), tr.Value)
	require.Equal(t, transfer.ClientID, tr.ClientID)
	require.Equal(t, transfer.ToClientID, tr.ToClientID)
}

func Test_AuthorizersShouldNotBeInitializedWhenContextIsCreated(t *testing.T) {
	sc := CreateMockStateContext(clientId)
	ans, err := getAuthorizerNodes(sc)
	require.NoError(t, err)
	require.NotNil(t, ans)
	require.Equal(t, len(ans.NodeMap), 0)
}

func TestAuthorizerNodeShouldBeAbleToAddTransfer(t *testing.T) {
	sc := CreateMockStateContext(clientId)
	an := getNewAuthorizer("public key", "id")
	tr := CreateDefaultTransaction()

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
	an := getNewAuthorizer("public key", "id")
	tr := CreateDefaultTransaction()

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

func Test_ShouldAddAuthorizer(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	balances := CreateMockStateContext(clientId)
	tr := CreateDefaultTransaction()

	response, err := sc.addAuthorizer(tr, data, balances)

	require.NoError(t, err, "must be able to add authorizer")
	require.NotEmpty(t, response)

	ans, err := getAuthorizerNodes(balances)
	require.NoError(t, err)

	node := ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.Equal(t, len(ans.NodeMap), 1)
}

func Test_Should_AddOnlyOneAuthorizerWithSameID(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	balances := CreateMockStateContext(clientId)
	tr := CreateDefaultTransaction()

	address, err := sc.addAuthorizer(tr, data, balances)
	require.NoError(t, err, "must be able to add authorizer")
	require.NotEmpty(t, address)

	// Check nodes state
	ans, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	node := ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.Equal(t, len(ans.NodeMap), 1)

	// Try adding one more authorizer
	address, err = sc.addAuthorizer(tr, data, balances)
	require.Error(t, err, "must be able to add only one authorizer")
	require.Contains(t, err.Error(), "failed to add authorizer")
	require.Empty(t, address)

	// Check nodes state
	ans, err = getAuthorizerNodes(balances)
	require.NoError(t, err)
	node = ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.Equal(t, len(ans.NodeMap), 1)
}

func TestShould_Fail_If_TransactionValue_Less_Then_GlobalNode_MinStake(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	balances := CreateMockStateContext(clientId)
	tr := CreateDefaultTransaction()
	tr.Value = 99

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = 100
	err := node.save(balances)
	require.NoError(t, err)

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.Empty(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is lower than min amount")
}

func Test_Should_FailWithoutPublicKey(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	tr.PublicKey = ""
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.Empty(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "public key was not included with transaction")
}

func Test_Transaction_Or_InputData_MustBe_A_Key_InputData(t *testing.T) {
	pk := &PublicKey{Key: "public Key"}
	data, _ := json.Marshal(pk)
	tr := CreateDefaultTransaction()
	tr.PublicKey = ""
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func Test_Transaction_Or_InputData_MustBe_A_Key_Transaction(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	tr.PublicKey = "public Key"
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func Test_Cannot_Delete_AuthorizerFromAnotherClient(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	ans, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.Nil(t, ans.NodeMap[tr.ClientID])

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)

	ans, err = getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, ans.NodeMap[tr.ClientID])

	tr = CreateTransaction("another client", 10)

	authorizer, err := sc.deleteAuthorizer(tr, data, balances)
	require.Empty(t, authorizer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "doesn't exist")
}

func Test_LockingBasicLogicTest (t *testing.T) {
	tr := CreateDefaultTransaction()
	z := &tokenpool.ZcnLockingPool{
		ZcnPool:            tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "0",
				Balance: 0,
			},
		},
		TokenLockInterface: tokenLock{
			StartTime: common.Now(),
			Duration:  0,
		},
	}

	locked := z.IsLocked(tr)
	require.Equal(t, locked, true)
}

func Test_Can_DigPool(t *testing.T) {
	tr := CreateDefaultTransaction()
	an := getNewAuthorizer("key", tr.ClientID)

	_, _, err := an.Staking.DigPool(tr.Hash, tr)
	require.NoError(t, err)
}

func Test_Can_EmptyPool(t *testing.T) {
	balances := CreateMockStateContext(clientId)
	tr := CreateDefaultTransaction()
	gn := getGlobalNode(balances)

	an := getNewAuthorizer("key", tr.ClientID)

	_, _, _ = an.Staking.DigPool(tr.Hash, tr)
	_, _, err := an.Staking.EmptyPool(gn.ID, tr.ClientID, tr)

	require.NoError(t, err)
}

func TestAuthorizerNodeShouldBeDecodedWithStakingPool(t *testing.T) {
	tr := CreateDefaultTransaction()
	an := getNewAuthorizer(tr.PublicKey, tr.ClientID)
	require.NotNil(t, an.Staking.TokenLockInterface)

	ans := &authorizerNodes{}
	ans.NodeMap = make(map[string]*authorizerNode)
	ans.NodeMap[an.ID] = an

	ans2 := &authorizerNodes{}
	err := ans2.Decode(ans.Encode())
	require.NoError(t, err)

	node := ans2.NodeMap[an.ID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)
}

// With this test, the ability to save nodes to context is tested
func Test_GetAuthorizerNodes_ShouldBeAbleToReturnNodes(t *testing.T) {
	ans := &authorizerNodes{}
	balances := CreateMockStateContext(clientId)
	av, err := balances.GetTrieNode(allAuthorizerKey)
	if err != nil {
		ans.NodeMap = make(map[string]*authorizerNode)
	} else {
		// deep copy to the local context
		_ = ans.Decode(av.Encode())
	}

	require.NotNil(t, ans)

	tr := CreateDefaultTransaction()
	an := getNewAuthorizer(tr.PublicKey, tr.ClientID)
	err = ans.addAuthorizer(an)
	require.NoError(t, err)
	require.NotNil(t, an.Staking.TokenLockInterface)

	node := ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)

	// without saving it won't be possible to get nodes
	err = ans.save(balances)
	require.NoError(t, err)

	ans2, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	node = ans2.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_NewAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	// Init
	tr := CreateDefaultTransaction()
	node := getNewAuthorizer(tr.PublicKey, tr.ClientID)
	require.NotNil(t, node.Staking.TokenLockInterface)
	balances := CreateMockStateContext(clientId)

	// Add
	ans, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	_ = ans.addAuthorizer(node)

	// Get
	node = ans.NodeMap[tr.ClientID]
	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenLockInterface)

	// Save
	err = ans.save(balances)
	require.NoError(t, err)

	// Get nodes again from context
	ans2, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	node = ans2.NodeMap[tr.ClientID]
	require.NotNil(t, node)

	// Staking Pool must be initialized
	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_AddedAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	// Init
	var data []byte
	tr := CreateDefaultTransaction()
	sc := CreateZCNSmartContract()

	// Add
	balances := CreateMockStateContext(clientId)
	_, _ = sc.addAuthorizer(tr, data, balances)

	// Get
	ans, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	node := ans.NodeMap[tr.ClientID]

	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_Can_Delete_Authorizer(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)
	require.NoError(t, err)

	ans, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, ans.NodeMap[tr.ClientID])
	require.NotNil(t, ans.NodeMap[tr.ClientID].Staking)

	tr = CreateDefaultTransaction()

	//_, err = sc.deleteAuthorizer(tr, data, balances)
	//require.NotEmpty(t, authorizer)
	require.NoError(t, err)
}

func Test_Authorizer_With_EmptyPool_Cannot_Be_Deleted(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)

	// This method is translated below
	resp, err = sc.deleteAuthorizer(tr, data, balances)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	//require.NotEmpty(t, authorizer)
	require.NoError(t, err)
}

func Test_Authorizer_EmptyPool_SimpleTest_Transfer(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NoError(t, err)
	responseDigPoolEqualityCheck(t, resp, tr, &tr.Hash, &stringEmpty)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)

	ans, err := getAuthorizerNodes(balances)
	require.NoError(t, err)

	gn := getGlobalNode(balances)
	transfer, resp, err := ans.NodeMap[tr.ClientID].Staking.EmptyPool(gn.ID, tr.ClientID, tr)
	require.NoError(t, err)

	transferEmptyPoolEqualityCheck(t, transfer, tr)
	responseEmptyPoolEqualityCheck(t, resp, tr, &stringEmpty, &tr.Hash)
}

func Test_AddAuthorizerNode_IsPersisted (t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext(clientId)
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)

	nodes, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, nodes.NodeMap)
}

func Test_Authorizers_NodeMap_ShouldBeInitializedAfterSaving (t *testing.T) {
	// Create authorizers nodes tree
	balances := CreateMockStateContext(clientId)
	tree, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.NodeMap)

	err = tree.save(balances)
	require.NoError(t, err)

	tree, err = getAuthorizerNodes(balances)
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