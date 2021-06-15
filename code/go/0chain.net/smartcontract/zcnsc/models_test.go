package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
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
	balances := CreateMockStateContext()
	nodes := getAuthorizerNodes(balances)

	require.NotNil(t, nodes)
	require.Equal(t, len(nodes.NodeMap), 0)
}

func Test_Authorizers_Should_Add_And_Return_And_UpdateAuthorizers(t *testing.T) {
	authorizer := getNewAuthorizer("public key")
	balances := CreateMockStateContext()

	nodes := getAuthorizerNodes(balances)
	err := nodes.addAuthorizer(authorizer)
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

func createStateAndNodeAndAddNodeToState() (cstate.StateContextI, *globalNode, error) {
	node := CreateSmartContractGlobalNode()
	node.MinBurnAmount = 111
	balances := CreateMockStateContext()
	err := node.save(balances)
	return balances, node, err
}
