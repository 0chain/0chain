package zcnsc_test

import (
	"math/rand"
	"testing"
	"time"

	"0chain.net/core/logging"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	logging.Logger = zap.NewNop()
}

func Test_MintPayload_Encode_Decode(t *testing.T) {
	ctx := MakeMockStateContext()
	expected, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)
	actual := &MintPayload{}
	err = actual.Decode(expected.Encode())
	require.NoError(t, err)
	require.Equal(t, expected.Nonce, actual.Nonce)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.EthereumTxnID, actual.EthereumTxnID)
	require.Equal(t, expected.ReceivingClientID, actual.ReceivingClientID)
	require.Equal(t, len(expected.Signatures), len(actual.Signatures))
	for i := range actual.Signatures {
		require.Equal(t, expected.Signatures[i].ID, actual.Signatures[i].ID)
		require.Equal(t, expected.Signatures[i].Signature, actual.Signatures[i].Signature)
	}
}

func Test_FuzzyMintTest(t *testing.T) {
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	for _, client := range clients {
		transaction := CreateAddAuthorizerTransaction(client, ctx)

		response, err := contract.Mint(transaction, payload.Encode(), ctx)

		require.NoError(t, err, "Testing authorizer: '%s'", client)
		require.NotNil(t, response)
		require.NotEmpty(t, response)
	}
}

// TBD
func Test_MintPayloadNonceShouldBeHigherByOneThanUserNonce(t *testing.T) {
	ctx := MakeMockStateContext()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()

	payload.Nonce = 1
	node, err := GetUserNode(defaultClient, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.NoError(t, node.Save(ctx))

	resp, err := contract.Mint(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
