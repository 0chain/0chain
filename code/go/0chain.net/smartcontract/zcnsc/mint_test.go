package zcnsc_test

import (
	"0chain.net/chaincore/chain"
	"0chain.net/core/logging"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	chain.ServerChain = new(chain.Chain)
	chain.ServerChain.Config = new(chain.Config)
	chain.ServerChain.ClientSignatureScheme = "bls0chain"

	logging.Logger = zap.NewNop()
}

func Test_MintPayload_Encode_Decode(t *testing.T) {
	expected, _, err := CreateMintPayload("client0", []string{"1", "2", "3"})
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

	payload, _, err := CreateMintPayload("client0", authorizers)
	require.NoError(t, err)

	for _, authorizer := range authorizers {
		transaction := CreateTransactionToZcnsc(authorizer, tokens)

		response, err := contract.Mint(transaction, payload.Encode(), ctx)

		require.NoError(t, err, "Testing authorizer: '%s'", authorizer)
		require.NotNil(t, response)
		require.NotEmpty(t, response)
	}
}

// TBD
func Test_MintPayloadNonceShouldBeHigherByOneThanUserNonce(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext(clientId)

	payload.Nonce = 1
	node, err := GetUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce - 1
	require.NoError(t, node.Save(ctx))

	resp, err := contract.Mint(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func Test_Chain_Prerequisite_test(t *testing.T) {
	ch := chain.GetServerChain()
	require.NotNil(t, ch)
	require.NotNil(t, ch.ClientSignatureScheme)
	require.NotEmpty(t, ch.ClientSignatureScheme)
	require.NotNil(t, ch.GetSignatureScheme())
}
