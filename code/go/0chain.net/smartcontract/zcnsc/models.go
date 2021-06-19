package zcnsc

import (
	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"encoding/json"
	"fmt"
	"time"

	// "0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	allAuthorizerKey = ADDRESS + encryption.Hash("all_authorizers")
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
	return ADDRESS + gn.ID
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

func (gn *globalNode) save(balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(gn.GetKey(), gn)
	return
}

func getGlobalSavedNode(balances cstate.StateContextI) (*globalNode, error) {
	gn := &globalNode{ID: ADDRESS}
	gv, err := balances.GetTrieNode(gn.GetKey())
	if err != nil {
		return gn, err
	}
	_ = gn.Decode(gv.Encode())
	return gn, err
}

func getGlobalNode(balances cstate.StateContextI) *globalNode {
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
		_ = signatureScheme.SetPublicKey(ans.NodeMap[v.ID].PublicKey)
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

func (pk *PublicKey) Encode() (data []byte, err error) {
	data, err = json.Marshal(pk)
	return
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

func (an *authorizerNode) Encode() []byte {
	bytes, _ := json.Marshal(an)
	return bytes
}

func (an *authorizerNode) Decode(input []byte, tokenlock tokenpool.TokenLockInterface) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}

	id, ok := objMap["id"]
	if ok {
		var idStr *string
		err = json.Unmarshal(*id, &idStr)
		if err != nil {
			return err
		}
		an.ID = *idStr
	}

	pk, ok := objMap["public_key"]
	if ok {
		var pkStr *string
		err = json.Unmarshal(*pk, &pkStr)
		if err != nil {
			return err
		}
		an.PublicKey = *pkStr
	}

	if an.Staking == nil {
		an.Staking = &tokenpool.ZcnLockingPool{
			ZcnPool:            tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{},
			},
		}
	}

	staking, ok := objMap["staking"]
	if ok {
		err = an.Staking.Decode(*staking, tokenlock)
		if err != nil {
			return err
		}
	}
	return nil
}

// To review
func getNewAuthorizer(pk string, id string) *authorizerNode {
	return &authorizerNode{
		PublicKey: pk,
		Staking: &tokenpool.ZcnLockingPool{
			ZcnPool: tokenpool.ZcnPool{
				TokenPool: tokenpool.TokenPool{
					ID:      id,
					Balance: 0,
				},
			},
			TokenLockInterface: tokenLock{
				StartTime: 0,
				Duration:  0,
				Owner:     id,
			},
		},
		ID: id,
	}
}

type authorizerNodes struct {
	NodeMap map[string]*authorizerNode `json:"node_map"`
}

func (an *authorizerNodes) Decode(input []byte) error {
	if an.NodeMap == nil {
		an.NodeMap = make(map[string]*authorizerNode)
	}

	var objMap map[string]json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}

	nodeMap, ok := objMap["node_map"]
	if ok {
		var authorizerNodes map[string]json.RawMessage
		err := json.Unmarshal(nodeMap, &authorizerNodes)
		if err != nil {
			return err
		}

		for _, raw := range authorizerNodes {
			target := &authorizerNode{}
			err := target.Decode(raw, &tokenLock{})
			if err != nil {
				return err
			}

			err = an.addAuthorizer(target)
			if err != nil {
				return err
			}
		}
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

func (an *authorizerNodes) save(balances cstate.StateContextI) (err error) {
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
	if node == nil {
		err = common.NewError("failed to add authorizer", "authorizerNode is not initialized")
		return
	}

	if an.NodeMap == nil {
		err = common.NewError("failed to add authorizer", "receiver NodeMap is nil")
		return
	}

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

func getAuthorizerNodes(balances cstate.StateContextI) (*authorizerNodes, error) {
	an := &authorizerNodes{}
	av, err := balances.GetTrieNode(allAuthorizerKey)
	if err != nil {
		an.NodeMap = make(map[string]*authorizerNode)
		return an, nil
	}
	err = an.Decode(av.Encode())
	return an, err
}

type userNode struct {
	ID    string `json:"id"`
	Nonce int64  `json:"nonce"`
}

func (un *userNode) GetKey(globalKey string) datastore.Key {
	return globalKey + un.ID
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

func (un *userNode) save(balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(un.GetKey(ADDRESS), un)
	return
}

func getUserNode(id string, balances cstate.StateContextI) (*userNode, error) {
	un := &userNode{ID: id}
	uv, err := balances.GetTrieNode(un.GetKey(ADDRESS))
	if err != nil {
		return un, err
	}
	_ = un.Decode(uv.Encode())
	return un, err
}

type tokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
	Owner     datastore.Key    `json:"owner"`
}

func (tl tokenLock) IsLocked(entity interface{}) bool {
	tm, ok := entity.(time.Time)
	if ok {
		return tm.Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	tm, ok := entity.(time.Time)
	if ok {
		p := &poolStat{
			StartTime: tl.StartTime,
			Duration:  tl.Duration,
			TimeLeft:  tl.Duration - tm.Sub(common.ToTime(tl.StartTime)), Locked: tl.IsLocked(tm)}
		return p.encode()
	}
	return nil
}

type poolStat struct {
	ID           datastore.Key    `json:"pool_id"`
	StartTime    common.Timestamp `json:"start_time"`
	Duration     time.Duration    `json:"duration"`
	TimeLeft     time.Duration    `json:"time_left"`
	Locked       bool             `json:"locked"`
	APR          float64          `json:"apr"`
	TokensEarned state.Balance    `json:"tokens_earned"`
	Balance      state.Balance    `json:"balance"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStat) decode(input []byte) error {
	err := json.Unmarshal(input, ps)
	return err
}
