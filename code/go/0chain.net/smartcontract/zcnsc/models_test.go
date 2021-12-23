package zcnsc_test

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/logging"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	chain.ServerChain = new(chain.Chain)
	chain.ServerChain.Config = chain.NewConfigImpl(&chain.ConfigData{ClientSignatureScheme: "bls0chain"})

	logging.Logger = zap.NewNop()
}

func Test_ShouldSign(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	_, err = signatureScheme.Sign(hex.EncodeToString(bytes))
	require.NoError(t, err)
}

func Test_ShouldSignAndVerify(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	hash := hex.EncodeToString(bytes)
	sig, err := signatureScheme.Sign(hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	ok, err := signatureScheme.Verify(sig, hash)
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

func Test_ShouldSignAndVerifyUsingPublicKey(t *testing.T) {
	bytes, err := json.Marshal("sample string")
	require.NoError(t, err)

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	err = signatureScheme.GenerateKeys()
	require.NoError(t, err)

	hash := hex.EncodeToString(bytes)
	sig, err := signatureScheme.Sign(hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	pk := signatureScheme.GetPublicKey()
	signatureScheme = chain.GetServerChain().GetSignatureScheme()
	err = signatureScheme.SetPublicKey(pk)
	require.NoError(t, err)

	ok, err := signatureScheme.Verify(sig, hash)
	require.NoError(t, err)
	require.Equal(t, ok, true)
}

func Test_ShouldVerifySignature(t *testing.T) {
	ctx := MakeMockStateContext()
	mp, err := CreateMintPayload("client0", []string{"p1", "p2"}, ctx)
	require.NoError(t, err)

	signatureScheme := ctx.GetSignatureScheme()
	require.NoError(t, err)

	toSign := mp.GetStringToSign()
	for _, v := range mp.Signatures {
		ok, err := signatureScheme.Verify(v.Signature, toSign)
		require.NoError(t, err)
		require.Equal(t, true, ok)
	}
}

func Test_ShouldSaveGlobalNode(t *testing.T) {
	_, _, err := createStateAndNodeAndAddNodeToState()
	require.NoError(t, err, "must Save the global node in state")
}

func Test_ShouldGetGlobalNode(t *testing.T) {
	balances, node, err := createStateAndNodeAndAddNodeToState()
	require.NoError(t, err, "must Save the global node in state")

	expected, _ := GetGlobalNode(balances)

	require.Equal(t, node.ID, expected.ID)
	require.Equal(t, node.MinBurnAmount, expected.MinBurnAmount)
}

func Test_GlobalNodeEncodeAndDecode(t *testing.T) {
	node := CreateSmartContractGlobalNode()
	node.BurnAddress = "11"
	node.MinMintAmount = 12
	node.MinBurnAmount = 13

	expected := CreateSmartContractGlobalNode()

	bytes := node.Encode()
	err := expected.Decode(bytes)

	require.NoError(t, err, "must Save the global node in state")

	expected.BurnAddress = "11"
	expected.MinMintAmount = 12
	expected.MinBurnAmount = 13
}

func Test_EmptyAuthorizersShouldNotHaveAnyNode(t *testing.T) {
	balances := MakeMockStateContext()
	nodes, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, nodes)
	require.Equal(t, 3, len(nodes.NodeMap))
}

func Test_Authorizers_Should_Add_And_Return_And_UpdateAuthorizers(t *testing.T) {
	authorizer := GetNewAuthorizer("public key", "id", "https://localhost:9876")
	balances := MakeMockStateContext()

	nodes, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	err = nodes.AddAuthorizer(authorizer)
	require.NoError(t, err, "must add authorizer")

	err = nodes.DeleteAuthorizer(authorizer.ID)
	require.NoError(t, err, "must delete authorizer")
}

func Test_PublicKey(t *testing.T) {
	pk := AuthorizerParameter{}

	err := pk.Decode(nil)
	require.Error(t, err)

	var data []byte
	err = pk.Decode(data)
	require.Error(t, err)

	data = []byte("")
	err = pk.Decode(data)
	require.Error(t, err)

	pk.PublicKey = "public key"

	bytes, err := json.Marshal(pk)
	require.NoError(t, err)

	expected := AuthorizerParameter{}
	err = expected.Decode(bytes)
	require.NoError(t, err)
	require.Equal(t, expected.PublicKey, pk.PublicKey)
}

func Test_ZcnLockingPool_ShouldBeSerializable(t *testing.T) {
	pool := &tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "id",
				Balance: 100,
			},
		},
		TokenLockInterface: TokenLock{
			StartTime: 0,
			Duration:  0,
			Owner:     "id",
		},
	}

	target := &tokenpool.ZcnLockingPool{}

	err := target.Decode(pool.Encode(), &TokenLock{})
	require.NoError(t, err)
	require.Equal(t, int(target.Balance), 100)
}

func Test_AuthorizerNode_ShouldBeSerializableWithTokenLock(t *testing.T) {
	// Create authorizer node
	tr := CreateDefaultTransactionToZcnsc()
	node := GetNewAuthorizer(tr.PublicKey, tr.ClientID, "https://localhost:9876")
	_, _, _ = node.Staking.DigPool(tr.Hash, tr)
	node.Staking.ID = "11"

	// Deserialize it into new instance
	target := &AuthorizerNode{}

	err := target.Decode(node.Encode())
	require.NoError(t, err)
	require.Equal(t, target.Staking.ID, "11")
	require.Equal(t, int64(target.Staking.Balance), tr.Value)
}

// This will test authorizer node serialization
func Test_AuthorizersTreeShouldBeSerialized(t *testing.T) {
	// Create authorizer node
	tr := CreateDefaultTransactionToZcnsc()
	node := GetNewAuthorizer(tr.PublicKey, tr.ClientID, "https://localhost:9876")
	node.Staking.ID = "11"
	node.Staking.Balance = 100

	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenPool)

	// Create authorizers nodes tree
	balances := MakeMockStateContext()
	tree, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.NodeMap)

	// Save authorizer node in the dictionary (nodes tree)
	tree.NodeMap[node.ID] = node

	// Serialize and deserialize nodes tree
	target := &AuthorizerNodes{}
	err = target.Decode(tree.Encode())
	require.NoError(t, err)
	require.NotNil(t, target)

	targetNode := target.NodeMap[node.ID]
	require.NotNil(t, targetNode)
	require.Equal(t, targetNode.ID, node.ID)
	require.Equal(t, targetNode.URL, node.URL)
	require.Equal(t, targetNode.PublicKey, node.PublicKey)
	require.Equal(t, targetNode.Staking.ID, node.Staking.ID)
	require.Equal(t, targetNode.Staking.Balance, node.Staking.Balance)
}

func Test_Authorizers_NodeMap_ShouldBeInitializedAfterDeserializing(t *testing.T) {
	// Create authorizers nodes tree
	balances := MakeMockStateContext()
	tree, err := GetAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.NodeMap)

	// Serialize and deserialize nodes tree
	target := &AuthorizerNodes{}
	err = target.Decode(tree.Encode())
	require.NoError(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.NodeMap)
}

func createStateAndNodeAndAddNodeToState() (cstate.StateContextI, *GlobalNode, error) {
	node := CreateSmartContractGlobalNode()
	node.MinBurnAmount = 111
	balances := MakeMockStateContext()
	err := node.Save(balances)
	return balances, node, err
}
