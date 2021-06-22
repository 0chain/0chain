package zcnsc

import (
	"github.com/stretchr/testify/require"
	"testing"
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
