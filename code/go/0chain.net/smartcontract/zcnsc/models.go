package zcnsc

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/chain"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	// "0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	allAuthorizerKey = datastore.Key(ADDRESS + encryption.Hash("all_authorizers"))
)

type globalNode struct {
	ID                 string        `json:"id"`
	MinMintAmount      state.Balance `json:"min_mint_amount"`
	PercentAuthorizers float64       `json:"percent_authorizers"`
	MinBurnAmount      int64         `json:"min_burn_amount"`
	MinStakeAmount     int64         `json:"min_stake_amount"`
	BurnAddress        string        `json:"burn_address"`
	MinAuthorizers     int64         `json:"min_authorizers"`
}

func (gn *globalNode) GetKey() datastore.Key {
	return datastore.Key(gn.ID + gn.ID)
}

func (gn *globalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *globalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

func (gn *globalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *globalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, gn)
	return err
}

func (gn *globalNode) save(balances c_state.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(gn.GetKey(), gn)
	return
}

func getGlobalSavedNode(balances c_state.StateContextI) (*globalNode, error) {
	gn := &globalNode{ID: ADDRESS}
	gv, err := balances.GetTrieNode(gn.GetKey())
	if err != nil {
		return gn, err
	}
	gn.Decode(gv.Encode())
	return gn, err
}

func getGlobalNode(balances c_state.StateContextI) *globalNode {
	gn, err := getGlobalSavedNode(balances)
	if err == nil {
		return gn
	}
	gn.MinMintAmount = state.Balance(config.SmartContractConfig.GetInt("smart_contracts.zcnsc.min_mint_amount"))
	gn.PercentAuthorizers = config.SmartContractConfig.GetFloat64("smart_contracts.zcnsc.percent_authorizers")
	gn.MinAuthorizers = config.SmartContractConfig.GetInt64("smart_contracts.zcnsc.min_authorizers")
	gn.MinBurnAmount = config.SmartContractConfig.GetInt64("smart_contracts.zcnsc.min_burn_amount")
	gn.MinStakeAmount = config.SmartContractConfig.GetInt64("smart_contracts.zcnsc.min_stake_amount")
	gn.BurnAddress = config.SmartContractConfig.GetString("smart_contracts.zcnsc.burn_address")
	return gn
}

type authorizerSignatures struct {
	ID        string `json:"authorizer_id"`
	Signature string `json:"signature"`
}

type mintPayload struct {
	EthereumTxnID     string                 `json:"ethereum_txn_id"`
	Amount            state.Balance          `json:"amount"`
	Nonce             int64                  `json:"nonce"`
	Signatures        []authorizerSignatures `json:"signatures"`
	ReceivingClientID string                 `json:"receiving_client_id"`
}

func (mp *mintPayload) Encode() []byte {
	buff, _ := json.Marshal(mp)
	return buff
}

func (mp *mintPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, mp)
	return err
}

func (mp *mintPayload) getStringToSign() string {
	return encryption.Hash(fmt.Sprintf("%v:%v:%v:%v", mp.EthereumTxnID, mp.Amount, mp.Nonce, mp.ReceivingClientID))
}

func (mp mintPayload) verifySignatures(ans *authorizerNodes) (ok bool) {
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	toSign := mp.getStringToSign()
	for _, v := range mp.Signatures {
		signatureScheme.SetPublicKey(ans.NodeMap[v.ID].PublicKey)
		ok, _ = signatureScheme.Verify(v.Signature, toSign)
		if !ok {
			return
		}
	}
	return
}

type burnPayload struct {
	TxnID           string `json:"0chain_txn_id"`
	Nonce           int64  `json:"nonce"`
	Amount          int64  `json:"amount"`
	EthereumAddress string `json:"ethereum_address"`
}

func (bp *burnPayload) Encode() []byte {
	buff, _ := json.Marshal(bp)
	return buff
}

func (bp *burnPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, bp)
	return err
}

type PublicKey struct {
	Key string `json:"public_key"`
}

func (pk *PublicKey) Decode(input []byte) error {
	err := json.Unmarshal(input, pk)
	return err
}

type authorizerNode struct {
	ID        string                    `json:"id"`
	PublicKey string                    `json:"public_key"`
	Staking   *tokenpool.ZcnLockingPool `json:"staking"`
}

func getNewAuthorizer(pk string) *authorizerNode {
	return &authorizerNode{PublicKey: pk, Staking: &tokenpool.ZcnLockingPool{}}
}

type authorizerNodes struct {
	NodeMap map[string]*authorizerNode `json:"node_map"`
}

func (an *authorizerNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, an)
	if err != nil {
		return err
	}
	return nil
}

func (an *authorizerNodes) Encode() []byte {
	buff, _ := json.Marshal(an)
	return buff
}

func (an *authorizerNodes) GetHash() string {
	return util.ToHex(an.GetHashBytes())
}

func (an *authorizerNodes) GetHashBytes() []byte {
	return encryption.RawHash(an.Encode())
}

func (an *authorizerNodes) save(balances c_state.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(allAuthorizerKey, an)
	return
}

func (an *authorizerNodes) deleteAuthorizer(id string) (err error) {
	if an.NodeMap[id] == nil {
		err = common.NewError("failed to delete authorizer", fmt.Sprintf("authorizer (%v) does not exist", id))
		return
	}
	delete(an.NodeMap, id)
	return
}

func (an *authorizerNodes) addAuthorizer(node *authorizerNode) (err error) {
	if an.NodeMap[node.ID] != nil {
		err = common.NewError("failed to add authorizer", fmt.Sprintf("authorizer (%v) already exists", node.ID))
		return
	}
	an.NodeMap[node.ID] = node
	return
}

func (an *authorizerNodes) updateAuthorizer(node *authorizerNode) (err error) {
	if an.NodeMap[node.ID] == nil {
		err = common.NewError("failed to update authorizer", fmt.Sprintf("authorizer (%v) does not exist", node.ID))
		return
	}
	an.NodeMap[node.ID] = node
	return
}

func getAuthorizerNodes(balances c_state.StateContextI) (an *authorizerNodes) {
	an = &authorizerNodes{}
	av, err := balances.GetTrieNode(allAuthorizerKey)
	if err != nil {
		an.NodeMap = make(map[string]*authorizerNode)
		return
	}
	an.Decode(av.Encode())
	return
}

type userNode struct {
	ID    string `json:"id"`
	Nonce int64  `json:"nonce"`
}

func (un *userNode) GetKey() datastore.Key {
	return datastore.Key(un.ID + un.ID)
}

func (un *userNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *userNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

func (un *userNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *userNode) Decode(input []byte) error {
	err := json.Unmarshal(input, un)
	return err
}

func (un *userNode) save(balances c_state.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(un.GetKey(), un)
	return
}

func getUserNode(id string, balances c_state.StateContextI) (*userNode, error) {
	un := &userNode{ID: id}
	uv, err := balances.GetTrieNode(un.GetKey())
	if err != nil {
		return un, err
	}
	un.Decode(uv.Encode())
	return un, err
}
