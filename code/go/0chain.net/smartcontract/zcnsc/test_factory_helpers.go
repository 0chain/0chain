package zcnsc

import (
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	ADD_AUTHORIZER = "addAuthorizer"
)

var (
	zcnAddressId                  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
	tokens                float64 = 10
	clientSignatureScheme         = "bls0chain"
)

func CreateDefaultTransactionToZcnsc() *transaction.Transaction {
	return CreateTransactionToZcnsc(clientId, tokens)
}

func addTransactionData(tr *transaction.Transaction, methodName string, input []byte) {
	sn := smartcontractinterface.SmartContractTransactionData{FunctionName: methodName, InputData: input}
	snBytes, err := json.Marshal(sn)
	if err != nil {
		panic(fmt.Sprintf("create smart contract failed due to invalid data. %s", err.Error()))
	}
	tr.TransactionType = transaction.TxnTypeSmartContract
	tr.TransactionData = string(snBytes)
}

func CreateTransactionToZcnsc(fromClient string, amount float64) *transaction.Transaction {
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
		TransactionType:       transaction.TxnTypeSmartContract,
		TransactionOutput:     "",
		OutputHash:            "",
		Status:                0,
	}

	publicKey := &PublicKey{Key: txn.PublicKey}
	pkBytes, err := json.Marshal(publicKey)
	if err != nil {
		panic(err.Error())
	}

	addTransactionData(txn, ADD_AUTHORIZER, pkBytes)

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

func createMintPayload(authorizers []string) (*mintPayload, string, error) {
	m := &mintPayload{
		EthereumTxnID:     txHash,
		Amount:            200,
		Nonce:             1,
		ReceivingClientID: "Client0",
	}

	signatures, pk, err := createTransactionSignatures(m, authorizers)
	if err != nil {
		return nil, pk, err
	}

	m.Signatures = signatures

	return m, pk, nil
}

func createTransactionSignatures(m *mintPayload, authorizers []string) ([]*authorizerSignature, string, error) {
	var sigs []*authorizerSignature

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	err := signatureScheme.GenerateKeys()
	if err != nil {
		return nil, "", err
	}

	signature, err := signatureScheme.Sign(m.getStringToSign())
	if err != nil {
		return nil, "", err
	}

	for _, id := range authorizers {
		sigs = append(
			sigs,
			&authorizerSignature{
				ID:        id,
				Signature: signature,
			})
	}

	return sigs, signatureScheme.GetPublicKey(), nil
}

func createUserNode(id string, nonce int64) *userNode {
	return &userNode{
		ID:    id,
		Nonce: nonce,
	}
}

func addAuthorizer(
	t *testing.T,
	contract *ZCNSmartContract,
	clientId string,
) {
	tr := CreateTransactionToZcnsc(clientId, 10)
	ctx := UpdateMockStateContext(tr)

	publicKey := &PublicKey{Key: tr.PublicKey}
	data, _ := publicKey.Encode()

	resp, err := contract.addAuthorizer(tr, data, ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)
}
