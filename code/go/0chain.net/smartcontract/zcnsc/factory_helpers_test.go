package zcnsc_test

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/chaincore/chain/state"

	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/smartcontract/zcnsc"
)

const (
	AddAuthorizer      = "AddAuthorizerFunc"
	clientPrefixID     = "fred"
	authorizerPrefixID = "authorizer"
	zcnAddressId       = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
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

func CreateAddAuthorizerTransaction(fromClient string, ctx state.StateContextI) *transaction.Transaction {
	scheme := ctx.GetSignatureScheme()
	_ = scheme.GenerateKeys()

	var txn = &transaction.Transaction{
		HashIDField:       datastore.HashIDField{Hash: txHash + "_transaction"},
		ClientID:          fromClient,
		ToClientID:        zcnAddressId,
		Value:             int64(zcnToBalance(1)),
		CreationDate:      startTime,
		PublicKey:         scheme.GetPublicKey(),
		TransactionData:   "",
		Signature:         "",
		Fee:               0,
		TransactionType:   transaction.TxnTypeSmartContract,
		TransactionOutput: "",
		OutputHash:        "",
	}

	payload := &AuthorizerParameter{}
	bytes, _ := payload.Encode()
	addTransactionData(txn, AddAuthorizer, bytes)

	return txn
}

func CreateAuthorizerParam() *AuthorizerParameter {
	return &AuthorizerParameter{
		PublicKey: "public key",
		URL:       "http://localhost:2344",
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  "100",
			MinStake:        100,
			MaxStake:        100,
			MaxNumDelegates: 100,
			ServiceCharge:   100,
		},
	}
}

func CreateAuthorizerParamPayload() []byte {
	p := CreateAuthorizerParam()
	encode, _ := p.Encode()
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
	return &GlobalNode{
		ID:                 ADDRESS,
		MinMintAmount:      111,
		PercentAuthorizers: 70,
		MinAuthorizers:     1,
		MinBurnAmount:      100,
		MinStakeAmount:     200,
		BurnAddress:        "0",
		MaxFee:             0,
	}
}

func createBurnPayload() *BurnPayload {
	return &BurnPayload{
		Nonce:           1,
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

func createUserNode(id string, nonce int64) *UserNode {
	return &UserNode{
		ID:    id,
		Nonce: nonce,
	}
}

//
//func CreateMockAuthorizer(clientId string, ctx state.StateContextI) (*AuthorizerNode, error) {
//	tr := CreateAddAuthorizerTransaction(clientId, ctx, 100)
//	authorizerNode := NewAuthorizer(clientId, tr.PublicKey, "https://localhost:9876")
//	_, _, err := authorizerNode.LockingPool.DigPool(tr.Hash, tr)
//	return authorizerNode, err
//}
