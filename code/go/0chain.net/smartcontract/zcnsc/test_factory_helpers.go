package zcnsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	zcnscAddressId         = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
	tokens         float64 = 10
	clientSignatureScheme = "bls0chain"
)

func CreateTransaction() *transaction.Transaction {

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
		ClientID:              clientId,
		ToClientID:            zcnscAddressId,
		Value:                 int64(zcnToBalance(tokens)),
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

	//txn = &transaction.Transaction{
	//	ClientID:   clientID0,
	//	ToClientID: zrc20scAddress,
	//	Value:      4,
	//}

	return txn
}

func CreateStateContext() cstate.StateContextI {
	return &mockStateContext{
		ctx:                cstate.StateContext{},
		clientStartBalance: 0,
		store:              make(map[datastore.Key]util.Serializable),
	}
}
