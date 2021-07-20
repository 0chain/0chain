package zcnsc

import (
	"0chain.net/chaincore/chain"
	"0chain.net/core/logging"
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
	expected := createMintPayload("1", "2", "3")
	actual := &mintPayload{}
	err := actual.Decode(expected.Encode())
	require.NoError(t, err)
	require.Equal(t, expected.Nonce, actual.Nonce)
	require.Equal(t, expected.Amount, actual.Amount)
	require.Equal(t, expected.EthereumTxnID, actual.EthereumTxnID)
	require.Equal(t, expected.ReceivingClientID, actual.ReceivingClientID)
	require.Equal(t, len(expected.Signatures), len(actual.Signatures))
	for i, _ := range actual.Signatures {
		require.Equal(t, expected.Signatures[i].ID, actual.Signatures[i].ID)
		require.Equal(t, expected.Signatures[i].Signature, actual.Signatures[i].Signature)
	}
}

// TBD
func Test_FuzzyMintTest(t *testing.T) {
	contract := CreateZCNSmartContract()
	tr := CreateDefaultTransactionToZcnsc()
	ctx := CreateMockStateContext(clientId)

	addAuthorizer(t, contract, ctx, clientId)
	addAuthorizer(t, contract, ctx, clientId + "1")
	addAuthorizer(t, contract, ctx, clientId + "2")

	payload := createMintPayload(clientId, clientId + "1", clientId + "2")

	response, err := contract.mint(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotEmpty(t, response)
}

// TBD
func MintPayloadNonceShouldBeHigherByOneThanUserNonce(t *testing.T) {
	payload := createBurnPayload()
	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()
	ctx := CreateMockStateContext(clientId)

	payload.Nonce = 1
	node, err := getUserNode(clientId, ctx)
	require.NoError(t, err)
	require.NotNil(t, node)
	node.Nonce = payload.Nonce - 1
	require.NoError(t, node.save(ctx))

	resp, err := contract.mint(tr, payload.Encode(), ctx)
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