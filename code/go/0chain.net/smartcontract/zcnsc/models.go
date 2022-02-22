package zcnsc

import (
	"encoding/json"
	"fmt"
	"time"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"github.com/pkg/errors"
)

//msgp:ignore MintPayload BurnPayloadResponse BurnPayload AuthorizerParameter poolStat
//go:generate msgp -io=false -tests=false -unexported=true -v

const (
	AuthorizerNodeType    = "authnode"
	AuthorizerNewNodeType = "create"
	GlobalNodeType        = "globalnode"
	UserNodeType          = "usernode"
)

// ------------- GlobalNode ------------------------

type GlobalNode struct {
	ID                 string        `json:"id"`
	MinMintAmount      state.Balance `json:"min_mint_amount"`
	PercentAuthorizers float64       `json:"percent_authorizers"`
	MinBurnAmount      int64         `json:"min_burn_amount"`
	MinStakeAmount     int64         `json:"min_stake_amount"`
	BurnAddress        string        `json:"burn_address"`
	MinAuthorizers     int64         `json:"min_authorizers"`
	MaxFee             int64         `json:"max_fee"`
}

func (gn *GlobalNode) GetKey() datastore.Key {
	return fmt.Sprintf("%s:%s:%s", ADDRESS, GlobalNodeType, gn.ID)
}

func (gn *GlobalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *GlobalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *GlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, gn)
	return err
}

func (gn *GlobalNode) Save(balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(gn.GetKey(), gn)
	return
}

type AuthorizerSignature struct {
	ID        string `json:"authorizer_id"`
	Signature string `json:"signature"`
}

// -----------  MintPayload -------------------

type MintPayload struct {
	EthereumTxnID     string                 `json:"ethereum_txn_id"`
	Amount            state.Balance          `json:"amount"`
	Nonce             int64                  `json:"nonce"`
	Signatures        []*AuthorizerSignature `json:"signatures"`
	ReceivingClientID string                 `json:"receiving_client_id"`
}

func (mp *MintPayload) Encode() []byte {
	buff, _ := json.Marshal(mp)
	return buff
}

func (mp *MintPayload) Decode(input []byte) error {
	const (
		fieldEthereumTxnId     = "ethereum_txn_id"
		fieldNonce             = "nonce"
		fieldAmount            = "amount"
		fieldReceivingClientId = "receiving_client_id"
		fieldSignatures        = "signatures"
	)

	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}

	id, ok := objMap[fieldEthereumTxnId]
	if ok {
		if id == nil {
			return errors.New("ethereum_txn_id is missing in the payload")
		}
		var value *string
		err = json.Unmarshal(*id, &value)
		if err != nil {
			return err
		}
		mp.EthereumTxnID = *value
	}

	id, ok = objMap[fieldNonce]
	if ok {
		if id == nil {
			return errors.New("nonce is missing in the payload")
		}
		var value *int64
		err = json.Unmarshal(*id, &value)
		if err != nil {
			return err
		}
		mp.Nonce = *value
	}

	id, ok = objMap[fieldAmount]
	if ok {
		if id == nil {
			return errors.New("amount is missing in the payload")
		}
		var value *int64
		err = json.Unmarshal(*id, &value)
		if err != nil {
			return err
		}
		mp.Amount = state.Balance(*value)
	}

	id, ok = objMap[fieldReceivingClientId]
	if ok {
		if id == nil {
			return errors.New("receiving_client_id is missing in the payload")
		}
		var value *string
		err = json.Unmarshal(*id, &value)
		if err != nil {
			return err
		}
		mp.ReceivingClientID = *value
	}

	id, ok = objMap[fieldSignatures]
	if ok {
		if id == nil {
			return errors.New("signatures entry is missing in payload")
		}
		var sigs []*json.RawMessage
		err = json.Unmarshal(*id, &sigs)
		if err != nil {
			return err
		}

		for _, raw := range sigs {
			sig := &AuthorizerSignature{}
			err = json.Unmarshal(*raw, sig)
			if err != nil {
				return err
			}
			mp.Signatures = append(mp.Signatures, sig)
		}
	}

	return err
}

func (mp *MintPayload) GetStringToSign() string {
	return encryption.Hash(fmt.Sprintf("%v:%v:%v:%v", mp.EthereumTxnID, mp.Amount, mp.Nonce, mp.ReceivingClientID))
}

func (mp *MintPayload) verifySignatures(state cstate.StateContextI) (err error) {
	toSign := mp.GetStringToSign()
	for _, v := range mp.Signatures {
		authorizerID := v.ID
		if authorizerID == "" {
			return errors.New("authorizer ID is empty in a signature")
		}

		node, err := GetAuthorizerNode(authorizerID, state)
		if err != nil {
			return errors.Wrapf(err, "failed to find authorizer by ID: %s", authorizerID)
		}

		if node.PublicKey == "" {
			return errors.New("authorizer public key is empty")
		}

		signatureScheme := state.GetSignatureScheme()
		err = signatureScheme.SetPublicKey(node.PublicKey)
		if err != nil {
			return errors.Wrap(err, "failed to set public key")
		}

		ok, err := signatureScheme.Verify(v.Signature, toSign)
		if !ok || err != nil {
			return errors.Wrap(err, "failed to verify signature")
		}
	}

	return
}

// ---- BurnPayloadResponse ----------

type BurnPayloadResponse struct {
	TxnID           string `json:"0chain_txn_id"`
	Nonce           int64  `json:"nonce"`
	Amount          int64  `json:"amount"`
	EthereumAddress string `json:"ethereum_address"`
}

func (bp *BurnPayloadResponse) Encode() []byte {
	buff, _ := json.Marshal(bp)
	return buff
}

func (bp *BurnPayloadResponse) Decode(input []byte) error {
	err := json.Unmarshal(input, bp)
	return err
}

// ------ BurnPayload ----------------

type BurnPayload struct {
	Nonce           int64  `json:"nonce"`
	EthereumAddress string `json:"ethereum_address"`
}

func (bp *BurnPayload) Encode() []byte {
	buff, _ := json.Marshal(bp)
	return buff
}

func (bp *BurnPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, bp)
	return err
}

// ------- AuthorizerParameter ------------

type AuthorizerParameter struct {
	PublicKey string `json:"public_key"`
	URL       string `json:"url"`
}

func (pk *AuthorizerParameter) Encode() (data []byte, err error) {
	data, err = json.Marshal(pk)
	return
}

func (pk *AuthorizerParameter) Decode(input []byte) error {
	err := json.Unmarshal(input, pk)
	return err
}

// ----- UserNode ------------------

type UserNode struct {
	ID    string `json:"id"`
	Nonce int64  `json:"nonce"`
}

func (un *UserNode) GetKey() datastore.Key {
	return fmt.Sprintf("%s:%s:%s", ADDRESS, UserNodeType, un.ID)
}

func (un *UserNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *UserNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	err := json.Unmarshal(input, un)
	return err
}

func (un *UserNode) Save(balances cstate.StateContextI) (err error) {
	_, err = balances.InsertTrieNode(un.GetKey(), un)
	return
}

// ----------  TokenLock -----------------

//type ZcnLockingPool struct {
//	tokenpool.ZcnPool `json:"pool"`
//	TokenLock         `json:"lock"`
//}

//func (zcn *ZcnLockingPool) Decode(data []byte) error {
//	return json.Unmarshal(data, zcn)
//}

type TokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  int64            `json:"duration"`
	Owner     string           `json:"owner"`
}

func (tl TokenLock) IsLocked(entity interface{}) bool {
	txn, ok := entity.(*transaction.Transaction)
	if txn.CreationDate == 0 {
		return false
	}
	if ok {
		return common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)) < time.Duration(tl.Duration)
	}
	return true
}

func (tl TokenLock) LockStats(entity interface{}) []byte {
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		p := &poolStat{
			StartTime: tl.StartTime,
			Duration:  time.Duration(tl.Duration),
			TimeLeft:  time.Duration(tl.Duration) - common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)),
			Locked:    tl.IsLocked(txn),
		}
		return p.encode()
	}
	return nil
}

type poolStat struct {
	ID        datastore.Key    `json:"pool_id"`
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
	TimeLeft  time.Duration    `json:"time_left"`
	Locked    bool             `json:"locked"`
}

func (ps *poolStat) encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *poolStat) decode(input []byte) error {
	err := json.Unmarshal(input, ps)
	return err
}
