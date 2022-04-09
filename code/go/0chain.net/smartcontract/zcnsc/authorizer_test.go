package zcnsc_test

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/common"
	"0chain.net/core/logging"
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
			DelegateWallet:  "",
			MinStake:        12,
			MaxStake:        12,
			MaxNumDelegates: 12,
			ServiceCharge:   12,
		},
	}
	data, _ := params.Encode()

	_, err := contract.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)

	_, err = contract.AddAuthorizer(tr, data, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
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

	data, _ := json.Marshal(CreateAuthorizerParam())
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

	node, err := GetAuthorizerNode(authorizerID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	tr := CreateAddAuthorizerTransaction("another client", ctx)
	var data []byte

	sc := CreateZCNSmartContract()
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
