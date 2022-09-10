package zcnsc_test

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/stakepool"
	. "0chain.net/smartcontract/zcnsc"
)

const (
	clientPrefixID     = "fred"
	authorizerPrefixID = "authorizer"
)

var (
	events map[string]*AuthorizerNode
	//authorizers       = make(map[string]*Authorizer, len(authorizersID))
	authorizersID     = []string{authorizerPrefixID + "_0", authorizerPrefixID + "_1", authorizerPrefixID + "_2"}
	clients           = []string{clientPrefixID + "_0", clientPrefixID + "_1", clientPrefixID + "_2"}
	defaultAuthorizer = authorizersID[0]
	defaultClient     = clients[0]
)

type Authorizer struct {
	Scheme encryption.SignatureScheme
	Node   *AuthorizerNode
}

func (n *Authorizer) Sign(payload string) (string, error) {
	return n.Scheme.Sign(payload)
}

func (n *Authorizer) Verify(sig, hash string) (bool, error) {
	return n.Scheme.Verify(sig, hash)
}

func CreateDefaultTransactionToZcnsc() *transaction.Transaction {
	return CreateAddAuthorizerTransaction(defaultClient, MakeMockStateContext())
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

func CreateDeleteAuthorizerTransaction(fromClient string, ctx state.StateContextI) (*transaction.Transaction, error) {
	scheme := ctx.GetSignatureScheme()
	_ = scheme.GenerateKeys()
	value, err := currency.ParseZCN(1)
	if err != nil {
		return nil, err
	}
	txn := &transaction.Transaction{
		HashIDField:       datastore.HashIDField{Hash: txHash + "_transaction"},
		ClientID:          fromClient,
		ToClientID:        ADDRESS,
		Value:             value,
		CreationDate:      startTime,
		PublicKey:         scheme.GetPublicKey(),
		TransactionData:   "",
		Signature:         "",
		Fee:               0,
		TransactionType:   transaction.TxnTypeSmartContract,
		TransactionOutput: "",
		OutputHash:        "",
	}
	addTransactionData(txn, DeleteAuthorizerFunc, nil)
	return txn, nil
}

func CreateAddAuthorizerTransaction(fromClient string, ctx state.StateContextI) *transaction.Transaction {
	scheme := ctx.GetSignatureScheme()
	_ = scheme.GenerateKeys()

	var txn = &transaction.Transaction{
		HashIDField:       datastore.HashIDField{Hash: txHash + "_transaction"},
		ClientID:          fromClient,
		ToClientID:        ADDRESS,
		Value:             1,
		CreationDate:      startTime,
		PublicKey:         scheme.GetPublicKey(),
		TransactionData:   "",
		Signature:         "",
		Fee:               0,
		TransactionType:   transaction.TxnTypeSmartContract,
		TransactionOutput: "",
		OutputHash:        "",
	}

	addTransactionData(txn, AddAuthorizerFunc, CreateAuthorizerParamPayload(fromClient))

	return txn
}

func CreateTransaction(fromClient, method string, payload []byte, ctx state.StateContextI) (*transaction.Transaction, error) {
	scheme := ctx.GetSignatureScheme()
	_ = scheme.GenerateKeys()
	value, err := currency.ParseZCN(1)
	if err != nil {
		return nil, err
	}

	var txn = &transaction.Transaction{
		HashIDField:       datastore.HashIDField{Hash: txHash + "_transaction"},
		ClientID:          fromClient,
		ToClientID:        ADDRESS,
		Value:             value,
		CreationDate:      startTime,
		PublicKey:         scheme.GetPublicKey(),
		TransactionData:   "",
		Signature:         "",
		Fee:               0,
		TransactionType:   transaction.TxnTypeSmartContract,
		TransactionOutput: "",
		OutputHash:        "",
	}

	addTransactionData(txn, method, payload)

	return txn, nil
}

func CreateAuthorizerParam(delegateWalletID string) *AddAuthorizerPayload {
	return &AddAuthorizerPayload{
		PublicKey: "public key",
		URL:       "http://localhost:2344",
		StakePoolSettings: stakepool.Settings{
			DelegateWallet:     delegateWalletID,
			MinStake:           12345678,
			MaxStake:           12345678,
			MaxNumDelegates:    12345678,
			ServiceChargeRatio: 12345678,
		},
	}
}

func CreateAuthorizerStakingPoolParam(delegateWalletID string) *UpdateAuthorizerStakePoolPayload {
	return &UpdateAuthorizerStakePoolPayload{
		StakePoolSettings: stakepool.Settings{
			DelegateWallet:     delegateWalletID,
			MinStake:           100,
			MaxStake:           100,
			MaxNumDelegates:    100,
			ServiceChargeRatio: 100,
		},
	}
}

func CreateAuthorizerParamPayload(delegateWalletID string) []byte {
	p := CreateAuthorizerParam(delegateWalletID)
	encode, _ := p.Encode()
	return encode
}

func CreateAuthorizerStakingPoolParamPayload(delegateWalletID string) []byte {
	p := CreateAuthorizerStakingPoolParam(delegateWalletID)
	encode := p.Encode()
	return encode
}

func CreateZCNSmartContract() *ZCNSmartContract {
	msc := new(ZCNSmartContract)
	msc.SmartContract = new(smartcontractinterface.SmartContract)
	msc.ID = ADDRESS
	msc.SmartContractExecutionStats = make(map[string]interface{})
	return msc
}

func CreateSmartContractGlobalNode() *GlobalNode {
	gn := &GlobalNode{
		ID: ADDRESS,
		ZCNSConfig: &ZCNSConfig{
			MinMintAmount:      111,
			MinBurnAmount:      100,
			MinStakeAmount:     200,
			MinLockAmount:      0,
			MinAuthorizers:     1,
			PercentAuthorizers: 70,
			MaxFee:             0,
			BurnAddress:        "0xBEEF",
			OwnerId:            "",
			Cost: map[string]int{
				AddAuthorizerFunc:    100,
				MintFunc:             100,
				BurnFunc:             100,
				DeleteAuthorizerFunc: 100,
			},
			MaxDelegates: 0,
		},
	}

	return gn
}

func createBurnPayload() *BurnPayload {
	return &BurnPayload{
		EthereumAddress: ADDRESS,
	}
}

func CreateMintPayload(ctx *mockStateContext, receiverId string) (payload *MintPayload, err error) {
	payload = &MintPayload{
		EthereumTxnID:     txHash,
		Amount:            200,
		Nonce:             1,
		ReceivingClientID: receiverId,
	}

	payload.Signatures, err = createTransactionSignatures(ctx, payload)

	return
}

func createTransactionSignatures(ctx *mockStateContext, m *MintPayload) ([]*AuthorizerSignature, error) {
	var sigs []*AuthorizerSignature

	for _, authorizer := range ctx.authorizers {
		signature, err := authorizer.Sign(m.GetStringToSign())
		if err != nil {
			return nil, err
		}

		sigs = append(sigs, &AuthorizerSignature{
			ID:        authorizer.Node.ID,
			Signature: signature,
		})
	}

	return sigs, nil
}

func createUserNode(id string) *UserNode {
	return &UserNode{
		ID: id,
	}
}

//
//func CreateMockAuthorizer(clientId string, ctx state.StateContextI) (*AuthorizerNode, error) {
//	tr := CreateAddAuthorizerTransaction(clientId, ctx, 100)
//	authorizerNode := NewAuthorizer(clientId, tr.PublicKey, "https://localhost:9876")
//	_, _, err := authorizerNode.LockingPool.DigPool(tr.Hash, tr)
//	return authorizerNode, err
//}
