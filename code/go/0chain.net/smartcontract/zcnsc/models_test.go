package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/tokenpool"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestShouldSaveGlobalNode(t *testing.T) {
	_, _, err := createStateAndNodeAndAddNodeToState()
	require.NoError(t, err, "must save the global node in state")
}

func TestShouldGetGlobalNode(t *testing.T) {
	balances, node, err := createStateAndNodeAndAddNodeToState()
	require.NoError(t, err, "must save the global node in state")

	expected := getGlobalNode(balances)

	require.Equal(t, node.ID, expected.ID)
	require.Equal(t, node.MinBurnAmount, expected.MinBurnAmount)
}

func TestGlobalNodeEncodeAndDecode(t *testing.T) {
	node := CreateSmartContractGlobalNode()
	node.BurnAddress = "11"
	node.MinMintAmount = 12
	node.MinBurnAmount = 13

	expected := CreateSmartContractGlobalNode()

	bytes := node.Encode()
	err := expected.Decode(bytes)

	require.NoError(t, err, "must save the global node in state")

	expected.BurnAddress = "11"
	expected.MinMintAmount = 12
	expected.MinBurnAmount = 13
}

func TestEmptyAuthorizersShouldNotHaveAnyNode(t *testing.T) {
	balances := CreateMockStateContext(clientId)
	nodes, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, nodes)
	require.Equal(t, len(nodes.NodeMap), 0)
}

func Test_Authorizers_Should_Add_And_Return_And_UpdateAuthorizers(t *testing.T) {
	authorizer := getNewAuthorizer("public key", "id")
	balances := CreateMockStateContext(clientId)

	nodes, err := getAuthorizerNodes(balances)
	require.NoError(t, err, )
	err = nodes.addAuthorizer(authorizer)
	require.NoError(t, err, "must add authorizer")

	err = nodes.deleteAuthorizer(authorizer.ID)
	require.NoError(t, err, "must delete authorizer")
}

func Test_PublicKey(t *testing.T) {
	pk := PublicKey{}

	err := pk.Decode(nil)
	require.Error(t, err)

	var data []byte
	err = pk.Decode(data)
	require.Error(t, err)

	data = []byte("")
	err = pk.Decode(data)
	require.Error(t, err)

	pk.Key = "public key"

	bytes, err := json.Marshal(pk)
	require.NoError(t, err)

	expected := PublicKey{}
	err = expected.Decode(bytes)
	require.NoError(t, err)
	require.Equal(t, expected.Key, pk.Key)
}

func Test_ZcnLockingPool_ShouldBeSerializable(t *testing.T) {
	pool := &tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "id",
				Balance: 100,
			},
		},
		TokenLockInterface: tokenLock{
			StartTime: 0,
			Duration:  0,
			Owner:     "id",
		},
	}

	target := &tokenpool.ZcnLockingPool{}

	err := target.Decode(pool.Encode(), &tokenLock{})
	require.NoError(t, err)
	require.Equal(t, int(target.Balance), 100)
}

func TestAuthorizerNode_ShouldBeSerializableWithTokenLock(t *testing.T) {
	// Create authorizer node
	tr := CreateDefaultTransaction()
	node := getNewAuthorizer(tr.PublicKey, tr.ClientID)
	node.Staking.ID = "11"
	node.Staking.Balance = 100

	// Deserialize it into new instance
	target := &authorizerNode{}

	err := target.Decode(node.Encode(), &tokenLock{})
	require.NoError(t, err)
	require.Equal(t, target.Staking.ID, "11")
	require.Equal(t, int(target.Staking.Balance), 100)
}

// This will test authorizer node serialization
func Test_AuthorizersTreeShouldBeSerialized(t *testing.T) {
	// Create authorizer node
	tr := CreateDefaultTransaction()
	node := getNewAuthorizer(tr.PublicKey, tr.ClientID)
	node.Staking.ID = "11"
	node.Staking.Balance = 100

	require.NotNil(t, node)
	require.NotNil(t, node.Staking.TokenPool)

	// Create authorizers nodes tree
	balances := CreateMockStateContext(clientId)
	tree, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.NodeMap)

	// Save authorizer node in the dictionary (nodes tree)
	tree.NodeMap[node.ID] = node

	// Serialize and deserialize nodes tree
	target := &authorizerNodes{}
	err = target.Decode(tree.Encode())
	require.NoError(t, err)
	require.NotNil(t, target)

	targetNode := target.NodeMap[node.ID]
	require.NotNil(t, targetNode)
	require.Equal(t, targetNode.Staking.ID, "11")
	require.Equal(t, int(targetNode.Staking.Balance), 100)
}

func Test_Authorizers_NodeMap_ShouldBeInitializedAfterDeserializing (t *testing.T) {
	// Create authorizers nodes tree
	balances := CreateMockStateContext(clientId)
	tree, err := getAuthorizerNodes(balances)
	require.NoError(t, err)
	require.NotNil(t, tree)
	require.NotNil(t, tree.NodeMap)

	// Serialize and deserialize nodes tree
	target := &authorizerNodes{}
	err = target.Decode(tree.Encode())
	require.NoError(t, err)
	require.NotNil(t, target)
	require.NotNil(t, target.NodeMap)
}

func createStateAndNodeAndAddNodeToState() (cstate.StateContextI, *globalNode, error) {
	node := CreateSmartContractGlobalNode()
	node.MinBurnAmount = 111
	balances := CreateMockStateContext(clientId)
	err := node.save(balances)
	return balances, node, err
}
