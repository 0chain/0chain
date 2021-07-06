package zcnsc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/stretchr/testify/require"
	"testing"
)

var (
	zcnAddressId                  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
	tokens                float64 = 10
	clientSignatureScheme         = "bls0chain"
)

func CreateDefaultTransaction() *transaction.Transaction {
	return CreateTransaction(clientId, tokens)
}

func CreateTransaction(fromClient string, amount float64) *transaction.Transaction {
	sigScheme := encryption.GetSignatureScheme(clientSignatureScheme)
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}

	pk := sigScheme.GetPublicKey()

	var txn = &transaction.Transaction{
		HashIDField:           datastore.HashIDField{Hash: txHash},
		CollectionMemberField: datastore.CollectionMemberField{},
		VersionField:          datastore.VersionField{},
		ClientID:              fromClient,
		ToClientID:            zcnAddressId,
		Value:                 int64(zcnToBalance(amount)),
		CreationDate:          startTime,
		PublicKey:             pk,
		ChainID:               "",
		TransactionData:       "",
		Signature:             "",
		Fee:                   0,
		TransactionType:       0,
		TransactionOutput:     "",
		OutputHash:            "",
		Status:                0,
	}

	return txn
}

func CreateZCNSmartContract() *ZCNSmartContract {
	msc := new(ZCNSmartContract)
	msc.SmartContract = new(smartcontractinterface.SmartContract)
	msc.ID = ADDRESS
	msc.SmartContractExecutionStats = make(map[string]interface{})
	return msc
}

func CreateSmartContractGlobalNode() *globalNode {
	return &globalNode{
		ID:                 ADDRESS,
		MinMintAmount:      111,
		PercentAuthorizers: 70,
		MinBurnAmount:      100,
		MinStakeAmount:     200,
		BurnAddress:        "0",
		MinAuthorizers:     1,
	}
}

func createBurnPayload() *burnPayload {
	return &burnPayload{
		TxnID:           txHash,
		Nonce:           1,
		Amount:          100,
		EthereumAddress: ADDRESS,
	}
}

func createMintPayload() *mintPayload {
	return &mintPayload{
		EthereumTxnID:     txHash,
		Amount:            200,
		Nonce:             1,
		Signatures:        createTransactionSignatures(),
		ReceivingClientID: "Client0",
	}
}

func createTransactionSignatures() []*authorizerSignature {
	var sigs []*authorizerSignature
	sigs = append(
		sigs,
		&authorizerSignature{
			ID:        "1",
			Signature: "sig1",
		},
		&authorizerSignature{
			ID:        "2",
			Signature: "sig2",
		},
		&authorizerSignature{
			ID:        "3",
			Signature: "sig3",
		})

	return sigs
}

func createUserNode(id string, nonce int64) *userNode {
	return &userNode{
		ID:    id,
		Nonce: nonce,
	}
}

func addAuthorizer(t *testing.T, contract *ZCNSmartContract, ctx state.StateContextI, clientId, pk string) {
	publicKey := &PublicKey{Key: pk}
	tr := CreateTransaction(clientId, 10)
	tr.PublicKey = publicKey.Key
	tr.ClientID = clientId
	data, _ := publicKey.Encode()
	resp, err := contract.addAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)
}
