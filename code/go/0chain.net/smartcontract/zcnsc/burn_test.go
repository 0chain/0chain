package zcnsc_test

import (
	"math/rand"
	"testing"
	"time"

	"0chain.net/core/logging"
	. "0chain.net/smartcontract/zcnsc"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	logging.Logger = zap.NewNop()
}

func TestBurnPayload_Encode_Decode(t *testing.T) {
	actual := BurnPayload{}
	expected := createBurnPayload()
	err := actual.Decode(expected.Encode())
	require.NoError(t, err)
	require.Equal(t, expected.EthereumAddress, actual.EthereumAddress)
}

func Test_FuzzyBurnTest(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()

	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
	require.NotEmpty(t, burn)
}

func Test_BurnPayloadNonceShouldBeHigherByOneThanUserNonce(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()

	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.NoError(t, node.Save(ctx))

	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
}

func Test_BurnNonceShouldIncrementBurnNonceBy1(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()

	// Save initial user node
	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.NoError(t, node.Save(ctx))
	require.Equal(t, int64(0), node.BurnNonce, "Initial nonce value should be 0")

	// Burn increments user node nonce
	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.Contains(t, burn, "\"nonce\":1")
	require.NoError(t, err)
	require.NotEmpty(t, burn)

	node, err = GetUserNode(defaultClient, ctx)
	require.Equal(t, int64(1), node.BurnNonce, "Nonce should be incremented to 1")
	require.NoError(t, err)
	require.NotNil(t, node)
	require.NoError(t, node.Save(ctx))

	burn, err = contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, burn)
	require.Contains(t, burn, "\"nonce\":2")
	node, err = GetUserNode(defaultClient, ctx)
	require.Equal(t, int64(2), node.BurnNonce, "Nonce should be incremented to 2")
}

func Test_EthereumAddressShouldBeFilled(t *testing.T) {
	// Without address

	payload := createBurnPayload()
	payload.EthereumAddress = stringEmpty

	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()

	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ethereum address is required")
	require.Empty(t, burn)

	// Fill address

	payload = createBurnPayload()
	payload.EthereumAddress = "EthereumAddress"

	tr = CreateDefaultTransactionToZcnsc()
	contract = CreateZCNSmartContract()
	ctx = MakeMockStateContext()

	resp, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
}

func Test_BurnNonceShouldIncrementDuringBurn(t *testing.T) {
	ctx := MakeMockStateContext()

	payload := createBurnPayload()
	contract := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(defaultClient, ctx)

	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	nonce := node.BurnNonce

	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
	require.NotEmpty(t, burn)

	node, err = GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	require.Equal(t, node.BurnNonce, nonce+1)
}

func Test_UserNodeSaveTest(t *testing.T) {
	ctx := MakeMockStateContext()
	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	node.BurnNonce += 2
	err = node.Save(ctx)
	require.NoError(t, err)

	node2, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	require.Equal(t, node.BurnNonce, node2.BurnNonce)
}

func Test_UserNodeEncode_Decode(t *testing.T) {
	ctx := MakeMockStateContext()
	node, err := GetUserNode(defaultClient, ctx)
	actual := UserNode{}
	err = actual.Decode(node.Encode())
	require.NoError(t, err)
	require.Equal(t, node.ID, actual.ID)
	require.Equal(t, node.BurnNonce, actual.BurnNonce)
}

func Test_Burn_should_return_encoded_payload(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()

	resp, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)

	actual := &BurnPayload{}
	err = actual.Decode([]byte(resp))
	require.NoError(t, err)
}

func Test_Should_Have_Added_TransferAfter_Burn(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()

	resp, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)

	transfers := ctx.GetTransfers()
	require.Equal(t, len(transfers), 1)

	gn, _ := GetGlobalNode(ctx)
	require.NotNil(t, gn)

	transfer := transfers[0]
	require.Equal(t, transfer.Amount, tr.Value)
	require.Equal(t, transfer.ClientID, tr.ClientID)
	require.Equal(t, transfer.ToClientID, gn.BurnAddress)
}
