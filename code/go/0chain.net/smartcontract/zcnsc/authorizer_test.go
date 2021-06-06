package zcnsc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TODO: Mock transaction.Transaction
// TODO: Prepare inputData []byte
// TODO: Mock c_state.StateContextI
// TODO: Create SC mock
// TODO: Mock Transaction.TransactionData with SmartContractTransactionData
// TODO: Mock SmartContractTransactionData

func TestShouldAddAuthorizer(t *testing.T) {
	//var sc = ZCNSmartContract{}

	tr := CreateTransaction()
	require.NotEmpty(t, tr.PublicKey)
	require.NotNil(t, tr.PublicKey)
	require.NotNil(t, tr.ClientID)
	require.NotNil(t, tr.ToClientID)
	require.NotZero(t, tr.Value)

	t.Logf("Public key: %s", tr.PublicKey)

	//data := []byte{}

	//address, _ := sc.addAuthorizer(tr, data, nil)

	//require.NotEmpty(t, address)
}

func TestShouldDeleteAuthorizer(t *testing.T) {
	var sc = ZCNSmartContract{}
	require.NotNil(t, sc)
}

func TestShouldFailIfAuthorizerExists(t *testing.T) {
	var sc = ZCNSmartContract{}
	require.NotNil(t, sc)
}