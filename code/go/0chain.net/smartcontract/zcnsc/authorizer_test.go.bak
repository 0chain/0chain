package zcnsc_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool/spenum"
	"0chain.net/smartcontract/storagesc"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"

	cstate "0chain.net/chaincore/chain/state"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/0chain/common/core/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	stringEmpty = ""
	ownerId     = "8a15e216a3b4237330c1fff19c7b3916ece5b0f47341013ceb64d53595a4cebb"
)

func init() {
	common.SetupRootContext(context.Background())
	rand.Seed(time.Now().UnixNano())
	logging.Logger = zap.NewNop()
}

func Test_BasicAuthorizersShouldBeInitialized(t *testing.T) {
	ctx := MakeMockStateContext()
	for _, authorizerKey := range authorizersID {
		node := &AuthorizerNode{
			Provider: provider.Provider{
				ID:           authorizerKey,
				ProviderType: spenum.Authorizer,
			},
		}
		err := ctx.GetTrieNode(node.GetKey(), node)
		require.NoError(t, err)
	}
}

func Test_Basic_GetGlobalNode_InitsNode(t *testing.T) {
	ctx := MakeMockStateContext()

	node, err := GetGlobalNode(ctx)
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
	ctx := MakeMockStateContext()
	delegateWallet := authorizersID[0] + ":10"

	//node := CreateSmartContractGlobalNode()
	//node.MinStakeAmount = currency.Coin(100 * 1e10)
	//node.MaxDelegates = 1000000
	//
	//err := node.Save(ctx)
	//require.NoError(t, err)

	input := CreateAuthorizerParamPayload(delegateWallet, AuthorizerPublicKey)
	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(ownerId, ctx)

	_, err := sc.AddAuthorizer(tr, input, ctx)
	require.NoError(t, err)

	_, err = sc.AddAuthorizer(tr, input, ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func Test_BasicShouldAddAuthorizer(t *testing.T) {
	ctx := MakeMockStateContext()
	delegateWallet := authorizersID[0] + ":10"

	input := CreateAuthorizerParamPayload(delegateWallet, AuthorizerPublicKey)
	publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
	id := encryption.Hash(publicKeyBytes)

	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(ownerId, ctx)

	resp, err := sc.AddAuthorizer(tr, input, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	authorizeNode, err := GetAuthorizerNode(id, ctx)
	require.NoError(t, err)

	err = ctx.GetTrieNode(authorizeNode.GetKey(), authorizeNode)
	require.NoError(t, err)
}

func Test_Should_AddOnlyOneAuthorizerWithSameID(t *testing.T) {
	ctx := MakeMockStateContext()
	delegateWallet := authorizersID[0] + ":10"

	input := CreateAuthorizerParamPayload(delegateWallet, AuthorizerPublicKey)
	publicKeyBytes, _ := hex.DecodeString(AuthorizerPublicKey)
	id := encryption.Hash(publicKeyBytes)

	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(ownerId, ctx)

	address, err := sc.AddAuthorizer(tr, input, ctx)
	require.NoError(t, err, "must be able to add authorizer")
	require.NotEmpty(t, address)

	// Check nodes state
	node, err := GetAuthorizerNode(id, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Try adding one more authorizer
	address, err = sc.AddAuthorizer(tr, input, ctx)
	require.Error(t, err, "must be able to add only one authorizer")
	require.Contains(t, err.Error(), "already exists")
	require.Empty(t, address)

	// Check nodes state
	node, err = GetAuthorizerNode(id, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
}

func Test_Basic_ShouldSaveGlobalNode(t *testing.T) {
	ctx := MakeMockStateContext()

	globalNode, err := GetGlobalNode(ctx)
	require.NoError(t, err)
	require.Equal(t, currency.Coin(11), globalNode.MinStakeAmount)

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = currency.Coin(100 * 1e10)

	err = node.Save(ctx)
	require.NoError(t, err)

	globalNode, err = GetGlobalNode(ctx)
	require.NoError(t, err)
	require.Equal(t, currency.Coin(100*1e10), globalNode.MinStakeAmount)
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
	delegateWallet := authorizersID[0] + ":10"

	input := CreateAuthorizerParamPayload(delegateWallet, AuthorizerPublicKey)

	sc := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(ownerId, ctx)

	resp, err := sc.AddAuthorizer(tr, input, ctx)
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
		Fee: currency.Coin(111),
	}

	err := node.UpdateConfig(cfg)
	require.NoError(t, err)
	err = node.Save(ctx)
	require.NoError(t, err)

	// Get node and check its setting
	node = GetAuthorizerNodeFromCtx(t, ctx, defaultAuthorizer)
	require.NotNil(t, node.Config)
	require.Equal(t, currency.Coin(111), node.Config.Fee)
}

func Test_AuthorizerHealthCheck(t *testing.T) {
	ctx := MakeMockStateContext()

	tr := CreateDefaultTransactionToZcnsc()
	sc := CreateZCNSmartContract()

	globalNode, err := GetGlobalNode(ctx)
	require.NoError(t, err)

	tr.ClientID = globalNode.ZCNSConfig.OwnerId

	addAuthorizerPayload := CreateAuthorizerParam("random_authorizer_delegate_wallet", tr.PublicKey)
	data, err := json.Marshal(addAuthorizerPayload)
	require.NoError(t, err)

	_, err = sc.AddAuthorizer(tr, data, ctx)
	require.NoError(t, err)

	node1 := GetAuthorizerNodeFromCtx(t, ctx, defaultAuthorizer)
	require.NotNil(t, node1)
	require.Zero(t, node1.LastHealthCheck)

	tr.ClientID = defaultAuthorizer

	authorizerHealthCheckPayload := AuthorizerHealthCheckPayload{
		ID: defaultAuthorizer,
	}
	data, err = json.Marshal(authorizerHealthCheckPayload)
	require.NoError(t, err)

	_, err = sc.AuthorizerHealthCheck(tr, data, ctx)
	require.NoError(t, err)

	node2 := GetAuthorizerNodeFromCtx(t, ctx, defaultAuthorizer)
	require.NotNil(t, node2)
	require.NotEqual(t, node2.LastHealthCheck, node1.LastHealthCheck)
	require.Equal(t, tr.CreationDate, node2.LastHealthCheck)
}

func GetAuthorizerNodeFromCtx(t *testing.T, ctx cstate.StateContextI, key string) *AuthorizerNode {
	node, err := GetAuthorizerNode(key, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	return node
}

func getAuthorizerCount(ctx cstate.StateContextI) (int, error) {
	numAuth := &AuthCount{}
	err := ctx.GetTrieNode(storagesc.AUTHORIZERS_COUNT_KEY, numAuth)
	if err == util.ErrValueNotPresent {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return numAuth.Count, nil
}

func Test_Can_Delete_Authorizer(t *testing.T) {
	var (
		ctx  = MakeMockStateContext()
		data []byte
	)
	payload := DeleteAuthorizerPayload{
		ID: defaultAuthorizer,
	}
	data, _ = json.Marshal(payload)
	sc := CreateZCNSmartContract()
	tr, err := CreateDeleteAuthorizerTransaction(defaultAuthorizer, ctx, data)
	require.NoError(t, err)

	// auth num before deleting
	authNum, err := getAuthorizerCount(ctx)
	require.NoError(t, err)

	resp, err := sc.DeleteAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	authorizerNode, err := GetAuthorizerNode(defaultAuthorizer, ctx)
	require.Error(t, err)
	require.Nil(t, authorizerNode)

	// auth number after deleting
	authNumAfter, err := getAuthorizerCount(ctx)
	require.NoError(t, err)
	require.Equal(t, authNum-1, authNumAfter)
}
