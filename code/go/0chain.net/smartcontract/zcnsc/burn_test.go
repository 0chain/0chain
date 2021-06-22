package zcnsc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBurnPayload_Encode_Decode(t *testing.T) {
	actual := burnPayload{}
	expected := createBurnPayload()
	err := actual.Decode(expected.Encode())
	require.NoError(t, err)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.Nonce, actual.Nonce)
	require.Equal(t, expected.TxnID, actual.TxnID)
	require.Equal(t, expected.EthereumAddress, actual.EthereumAddress)
}

func Test_FuzzyTest(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransaction()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext()

	burn, err := contract.burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
	require.NotEmpty(t, burn)
}

func Test_PayloadNonceShouldBeHigherByOneThanUserNonce(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransaction()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext()

	payload.Nonce = 1
	node, err := getUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce - 1
	require.NoError(t, node.save(ctx))

	burn, err := contract.burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
}

func Test_PayloadNonceLessOrEqualThanUserNonce_Fails(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransaction()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext()

	payload.Nonce = 1

	// case 1
	node, err := getUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce
	require.NoError(t, node.save(ctx))

	burn, err := contract.burn(tr, payload.Encode(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "should be 1 higher than the current nonce")
	require.Empty(t, burn)

	// case 2
	node, err = getUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce + 1
	require.NoError(t, node.save(ctx))

	burn, err = contract.burn(tr, payload.Encode(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "should be 1 higher than the current nonce")
	require.Empty(t, burn)
}

func Test_EthereumAddressShouldBeFilled(t *testing.T) {

	// Without address

	payload := createBurnPayload()
	payload.EthereumAddress = ""

	tr := CreateDefaultTransaction()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext()

	burn, err := contract.burn(tr, payload.Encode(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ethereum address is required")
	require.Empty(t, burn)

	// Fill address

	payload = createBurnPayload()
	payload.EthereumAddress = "EthereumAddress"

	tr = CreateDefaultTransaction()
	contract = CreateZCNSmartContract()
	ctx = CreateMockStateContext()

	resp, err := contract.burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
}

func Test_userNodeNonceShouldIncrement(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransaction()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext()

	node, err := getUserNode(tr.ClientID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	nonce := node.Nonce

	burn, err := contract.burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
	require.NotEmpty(t, burn)

	node, err = getUserNode(tr.ClientID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	require.Equal(t, node.Nonce, nonce + 1)
}

func Test_getUserNode(t *testing.T) {
	ctx := CreateMockStateContext()
	node, err := getUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
}

func Test_updateUserNode(t *testing.T) {
	ctx := CreateMockStateContext()
	node, err := getUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	node.Nonce += 2
	err = node.save(ctx)
	require.NoError(t, err)

	node2, err := getUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	require.Equal(t, node.Nonce, node2.Nonce)
}

func Test_UserNodeEncode_Decode(t *testing.T) {
	node := createUserNode(clientId, 10)
	actual := userNode{}
	err := actual.Decode(node.Encode())
	require.NoError(t, err)
	require.Equal(t, node.ID, actual.ID)
	require.Equal(t, node.Nonce, actual.Nonce)
}

func Test_burn_should_return_encoded_payload(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransaction()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext()

	resp, err := contract.burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)

	actual := &burnPayload{}
	err = actual.Decode([]byte(resp))
	require.NoError(t, err)
	require.Equal(t, tr.Value, actual.Amount)
	require.Equal(t, tr.Hash, actual.TxnID)
}

func Test_Should_Have_Added_TransferAfter_Burn(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransaction()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext()

	resp, err := contract.burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)

	transfers := ctx.GetTransfers()
	require.Equal(t, len(transfers), 1)

	gn := getGlobalNode(ctx)
	require.NotNil(t, gn)

	transfer := transfers[0]
	require.Equal(t, int64(transfer.Amount), tr.Value)
	require.Equal(t, transfer.ClientID, tr.ClientID)
	require.Equal(t, transfer.ToClientID, gn.BurnAddress)
}