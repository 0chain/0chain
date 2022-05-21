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
	require.Equal(t, expected.Nonce, actual.Nonce)
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

	payload.Nonce = 1
	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce - 1
	require.NoError(t, node.Save(ctx))

	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
}

func Test_PayloadNonceLessOrEqualThanUserNonce_Fails(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := MakeMockStateContext()

	payload.Nonce = 1

	// case 1
	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce
	require.NoError(t, node.Save(ctx))

	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonce given (1) for burning client (fred_0) must be greater by 1 than the current node nonce (1) for Node.ID: 'fred_0'")
	require.Empty(t, burn)

	// case 2
	node, err = GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce + 1
	require.NoError(t, node.Save(ctx))

	burn, err = contract.Burn(tr, payload.Encode(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonce given (1) for burning client (fred_0) must be greater by 1 than the current node nonce (2) for Node.ID: 'fred_0'")
	require.Empty(t, burn)
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

func Test_UserNodeNonceShouldIncrement(t *testing.T) {
	ctx := MakeMockStateContext()

	payload := createBurnPayload()
	contract := CreateZCNSmartContract()
	tr := CreateAddAuthorizerTransaction(defaultClient, ctx)

	node, err := GetUserNode(tr.ClientID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	nonce := node.Nonce

	burn, err := contract.Burn(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, burn)
	require.NotEmpty(t, burn)

	node, err = GetUserNode(tr.ClientID, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	require.Equal(t, node.Nonce, nonce+1)
}

func Test_UpdateUserNode(t *testing.T) {
	ctx := MakeMockStateContext()
	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	node.Nonce += 2
	err = node.Save(ctx)
	require.NoError(t, err)

	node2, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)

	require.Equal(t, node.Nonce, node2.Nonce)
}

func Test_UserNodeEncode_Decode(t *testing.T) {
	node := createUserNode(defaultClient, 10)
	actual := UserNode{}
	err := actual.Decode(node.Encode())
	require.NoError(t, err)
	require.Equal(t, node.ID, actual.ID)
	require.Equal(t, node.Nonce, actual.Nonce)
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
	require.Equal(t, int64(transfer.Amount), tr.ValueZCN)
	require.Equal(t, transfer.ClientID, tr.ClientID)
	require.Equal(t, transfer.ToClientID, gn.BurnAddress)
}
