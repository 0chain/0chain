package zcnsc

import (
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

var (
	zcnscAddressId                = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
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
		ToClientID:            zcnscAddressId,
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
		MinMintAmount:      0,
		PercentAuthorizers: 0,
		MinBurnAmount:      0,
		MinStakeAmount:     0,
		BurnAddress:        "0",
		MinAuthorizers:     0,
	}
}
