package zcnsc_test

import (
	"math/rand"
	"testing"
	"time"

	. "0chain.net/smartcontract/zcnsc"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
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

func Test_DifferentSenderAndReceiverMustFail(t *testing.T) {
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	transaction, err := CreateTransaction(defaultClient+"1", "mint", payload.Encode(), ctx)
	require.NoError(t, err)

	_, err = contract.Mint(transaction, payload.Encode(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction made from different account who made burn")
}

func Test_FuzzyMintTest(t *testing.T) {
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	for _, client := range clients {
		transaction, err := CreateTransaction(defaultClient, "mint", payload.Encode(), ctx)
		require.NoError(t, err)

		response, err := contract.Mint(transaction, payload.Encode(), ctx)

		require.NoError(t, err, "Testing authorizer: '%s'", client)
		require.NotNil(t, response)
		require.NotEmpty(t, response)
	}
}

func Test_MaxFeeMint(t *testing.T) {
	t.Skip("add back after issue #2011 is fixed")
	const maxFee = 10
	ctx := MakeMockStateContext()
	ctx.globalNode.ZCNSConfig.MaxFee = maxFee
	contract := CreateZCNSmartContract()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	transaction, err := CreateTransaction(defaultClient, "mint", payload.Encode(), ctx)
	require.NoError(t, err)

	response, err := contract.Mint(transaction, payload.Encode(), ctx)
	require.NoError(t, err, "Testing authorizer: '%s'", defaultClient)
	require.NotNil(t, response)
	require.NotEmpty(t, response)

	mm := ctx.GetMints()
	require.Equal(t, len(mm), len(authorizersID)+1)
	expectedShare, _, err := currency.DistributeCoin(maxFee, int64(len(authorizersID)))
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		require.Equal(t, mm[i].ToClientID, authorizersID[i])
		require.Equal(t, mm[i].Amount, expectedShare)
	}

	rp := &MintPayload{}
	require.NoError(t, rp.Decode([]byte(response)))
	require.Equal(t, payload.ReceivingClientID, rp.ReceivingClientID)
	sz, err := currency.Int64ToCoin(int64(len(authorizersID)))
	require.NoError(t, err)
	totalShare, err := currency.MultCoin(expectedShare, sz)
	require.NoError(t, err)
	expectedAmount, err := currency.MinusCoin(payload.Amount, totalShare)
	require.NoError(t, err)
	require.Equal(t, rp.Amount, expectedAmount)
}

func Test_EmptySignaturesShouldFail(t *testing.T) {
	ctx := MakeMockStateContext()
	contract := CreateZCNSmartContract()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	payload.Signatures = nil

	transaction, err := CreateTransaction(defaultClient, "mint", payload.Encode(), ctx)
	require.NoError(t, err)

	_, err = contract.Mint(transaction, payload.Encode(), ctx)
	require.Error(t, err)
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
