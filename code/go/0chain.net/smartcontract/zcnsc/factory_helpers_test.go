package zcnsc_test

import (
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/smartcontract/zcnsc"
	"encoding/json"
	"fmt"
)

const (
	AddAuthorizer = "AddAuthorizerFunc"
)

var (
	zcnAddressId                  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
	tokens                float64 = 10
	clientSignatureScheme         = "bls0chain"
	authorizers                   = []string{clientId, clientId + "1", clientId + "2"}
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
		HashIDField:           datastore.HashIDField{Hash: txHash + "_transaction"},
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

	publicKey := &AuthorizerParameter{PublicKey: txn.PublicKey, URL: "https://localhost:9876"}
	pkBytes, err := json.Marshal(publicKey)
	if err != nil {
		panic(err.Error())
	}

	addTransactionData(txn, AddAuthorizer, pkBytes)

	return txn
}

func CreateAuthorizerParam() *AuthorizerParameter {
	return &AuthorizerParameter{
		PublicKey: "public key",
		URL:       "http://localhost:2344",
	}
}

func CreateZCNSmartContract() *ZCNSmartContract {
	msc := new(ZCNSmartContract)
	msc.SmartContract = new(smartcontractinterface.SmartContract)
	msc.ID = ADDRESS
	msc.SmartContractExecutionStats = make(map[string]interface{})
	return msc
}

func CreateSmartContractGlobalNode() *GlobalNode {
	return &GlobalNode{
		ID:                 ADDRESS,
		MinMintAmount:      111,
		PercentAuthorizers: 70,
		MinBurnAmount:      100,
		MinStakeAmount:     200,
		BurnAddress:        "0",
		MinAuthorizers:     1,
	}
}

func createBurnPayload() *BurnPayload {
	return &BurnPayload{
		Nonce:           1,
		EthereumAddress: ADDRESS,
	}
}

func CreateMintPayload(receiverId string, authorizers []string) (*MintPayload, string, error) {
	m := &MintPayload{
		EthereumTxnID:     txHash,
		Amount:            200,
		Nonce:             1,
		ReceivingClientID: receiverId,
	}

	signatures, pk, err := createTransactionSignatures(m, authorizers)
	if err != nil {
		return nil, pk, err
	}

	m.Signatures = signatures

	return m, pk, nil
}

func createTransactionSignatures(m *MintPayload, authorizers []string) ([]*AuthorizerSignature, string, error) {
	var sigs []*AuthorizerSignature

	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	err := signatureScheme.GenerateKeys()
	if err != nil {
		return nil, "", err
	}

	signature, err := signatureScheme.Sign(m.GetStringToSign())
	if err != nil {
		return nil, "", err
	}

	for _, id := range authorizers {
		sigs = append(
			sigs,
			&AuthorizerSignature{
				ID:        id,
				Signature: signature,
			})
	}

	return sigs, signatureScheme.GetPublicKey(), nil
}

func createUserNode(id string, nonce int64) *UserNode {
	return &UserNode{
		ID:    id,
		Nonce: nonce,
	}
}

func CreateMockAuthorizer(clientId string) *AuthorizerNode {
	tr := CreateTransactionToZcnsc(clientId, 100)
	authorizerNode := GetNewAuthorizer(tr.PublicKey, clientId, "https://localhost:9876")
	_, _, _ = authorizerNode.Staking.DigPool(tr.Hash, tr)
	return authorizerNode
}
