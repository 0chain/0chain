package zcnsc_test

import (
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
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
	type expect struct {
		sharedFee    currency.Coin
		remainAmount currency.Coin
	}

	tt := []struct {
		name   string
		maxFee currency.Coin
		expect expect
	}{
		{
			name:   "max fee not evenly distributed",
			maxFee: 10,
			expect: expect{
				sharedFee:    3,
				remainAmount: 191,
			},
		},
		{
			name:   "max fee evenly distributed",
			maxFee: 9,
			expect: expect{
				sharedFee:    3,
				remainAmount: 191,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := MakeMockStateContext()
			ctx.globalNode.ZCNSConfig.MaxFee = tc.maxFee
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

			auths := make([]string, 0, len(payload.Signatures))
			for _, sig := range payload.Signatures {
				auths = append(auths, sig.ID)
			}

			mintsMap := make(map[string]*state.Mint, len(mm))
			for i, m := range mm {
				mintsMap[m.ToClientID] = mm[i]
			}

			for _, id := range auths {
				require.Equal(t, tc.expect.sharedFee, mintsMap[id].Amount)
			}

			// assert transaction.ClientID has remaining amount
			tm, ok := mintsMap[transaction.ClientID]
			require.True(t, ok)
			require.Equal(t, tc.expect.remainAmount, tm.Amount)
		})
	}
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
	require.Contains(t, err.Error(), "signatures entry is missing in payload")
}

func Test_EmptyAuthorizersNonemptySignaturesShouldFail(t *testing.T) {
	ctx := MakeMockStateContextWithoutAutorizers()

	contract := CreateZCNSmartContract()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	// Add a few signatures.
	var signatures []*AuthorizerSignature
	for _, id := range []string{"sign1", "sign2", "sign3"} {
		signatures = append(signatures, &AuthorizerSignature{ID: id})
	}
	payload.Signatures = signatures

	transaction, err := CreateTransaction(defaultClient, "mint", payload.Encode(), ctx)
	require.NoError(t, err)

	_, err = contract.Mint(transaction, payload.Encode(), ctx)
	require.Equal(t, common.NewError("failed to mint", "no authorizers found"), err)
}

func Test_MintPayloadNonceShouldBeRecordedByUserNode(t *testing.T) {
	ctx := MakeMockStateContext()
	payload, err := CreateMintPayload(ctx, defaultClient)
	require.NoError(t, err)

	tr := CreateDefaultTransactionToZcnsc()
	contract := CreateZCNSmartContract()

	gn, err := GetGlobalNode(ctx)
	require.NoError(t, err)
	require.NotNil(t, gn)

	user, err := ctx.GetEventDB().GetUser(tr.ClientID)
	require.NoError(t, err)

	payload.Nonce = 1

	resp, err := contract.Mint(tr, payload.Encode(), ctx)
	require.NoError(t, err)
	require.NotZero(t, resp)

	require.Equal(t, user.MintNonce, payload.Nonce)

	resp, err = contract.Mint(tr, payload.Encode(), ctx)
	require.Error(t, err)
	require.Zero(t, resp)
}
