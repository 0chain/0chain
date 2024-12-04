package zcnsc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/core/sortedmap"
	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/pkg/errors"
)

//msgp:ignore MintPayload BurnPayloadResponse BurnPayload AddAuthorizerPayload UpdateAuthorizerStakePoolPayload poolStat AuthorizerSignature TokenLock
//go:generate msgp -io=false -tests=false -unexported=true -v

const (
	AuthorizerNodeType    = "authnode"
	AuthorizerNewNodeType = "create"
	GlobalNodeType        = "globalnode"
	StakePoolNodeType     = "stakepool"
	UserNodeType          = "usernode"
	Porvider              = "provider"
)

type (
	smartContractFunction func(t *transaction.Transaction, inputData []byte, balances cstate.StateContextI) (string, error)
)

// -----------  AuthorizerSignature -------------------

type AuthorizerSignature struct {
	ID        string `json:"authorizer_id"`
	Signature string `json:"signature"`
}

// -----------  MintPayload -------------------

type MintPayload struct {
	EthereumTxnID     string                 `json:"ethereum_txn_id"`
	Amount            currency.Coin          `json:"amount"`
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
		mp.Amount = currency.Coin(*value)
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

func (mp *MintPayload) verifySignatures(signatures []*AuthorizerSignature, state cstate.StateContextI) error {
	toSign := mp.GetStringToSign()
	if len(signatures) == 0 {
		return errors.New("signatures not found")
	}

	for _, v := range signatures {
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

	return nil
}

func (mp *MintPayload) getUniqueSignatures() []*AuthorizerSignature {
	sigsMap := sortedmap.New[string, *AuthorizerSignature]()
	for i, v := range mp.Signatures {
		sigsMap.Put(v.ID, mp.Signatures[i])
	}

	return sigsMap.GetValues()
}

// ---- BurnPayloadResponse ----------

type BurnPayloadResponse struct {
	TxnID           string        `json:"0chain_txn_id"`
	Nonce           int64         `json:"nonce"`
	Amount          currency.Coin `json:"amount"`
	EthereumAddress string        `json:"ethereum_address"`
}

func (bp *BurnPayloadResponse) Encode() []byte {
	buff, _ := json.Marshal(bp)
	return buff
}

func (bp *BurnPayloadResponse) Decode(input []byte) error {
	err := json.Unmarshal(input, bp)
	return err
}

// ------ RepairEthAddressPayload ----------------

type RepairEthAddressPayload struct {
	EthereumAddress string `json:"ethereum_address"`
	Nonce           int64  `json:"nonce"`
}

func (reap *RepairEthAddressPayload) Encode() []byte {
	buff, _ := json.Marshal(reap)
	return buff
}

func (reap *RepairEthAddressPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, reap)
	return err
}

// ------ BurnPayload ----------------

type BurnPayload struct {
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

// ------- UpdateAuthorizerStakePoolPayload ------------

type UpdateAuthorizerStakePoolPayload struct {
	StakePoolSettings stakepool.Settings `json:"stake_pool_settings"`
}

func (pk *UpdateAuthorizerStakePoolPayload) Encode() (data []byte) {
	data, _ = json.Marshal(pk)
	return
}

func (pk *UpdateAuthorizerStakePoolPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, pk)
	return err
}

// ------- AddAuthorizerPayload ------------

//	type addAuthorizerPayload struct {
//		URL           string
//		ClientID      string
//		ClientKey     string
//		NumDelegates  int
//		ServiceCharge float64
//	}
type AddAuthorizerPayload struct {
	PublicKey         string             `json:"public_key"`
	URL               string             `json:"url"`
	StakePoolSettings stakepool.Settings `json:"stake_pool_settings"` // Used to initially create stake pool
}

func (pk *AddAuthorizerPayload) Encode() (data []byte, err error) {
	data, err = json.Marshal(pk)
	return
}

func (pk *AddAuthorizerPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, pk)
	return err
}

// ----------  TokenLock -----------------

type TokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
	Owner     string           `json:"owner"`
}

func (tl TokenLock) IsLocked(entity interface{}) bool {
	txn, ok := entity.(*transaction.Transaction)
	if txn.CreationDate == 0 {
		return false
	}
	if ok {
		return common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl TokenLock) LockStats(entity interface{}) []byte {
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		p := &poolStat{
			StartTime: tl.StartTime,
			Duration:  tl.Duration,
			TimeLeft:  tl.Duration - common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)),
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

type AuthCount struct {
	Count int `json:"auth_count"`
}

func (ac *AuthCount) Encode() ([]byte, error) {
	return json.Marshal(ac)
}

func (ac *AuthCount) Decode(input []byte) error {
	return json.Unmarshal(input, ac)
}

type DeleteAuthorizerPayload struct {
	ID string `json:"id"`
}

func (dap *DeleteAuthorizerPayload) Encode() (data []byte, err error) {
	data, err = json.Marshal(dap)
	return
}

func (dap *DeleteAuthorizerPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, dap)
	return err
}

type AuthorizerHealthCheckPayload struct {
	ID string `json:"id"`
}

func (ahp *AuthorizerHealthCheckPayload) Encode() (data []byte, err error) {
	data, err = json.Marshal(ahp)
	return
}

func (ahp *AuthorizerHealthCheckPayload) Decode(input []byte) error {
	err := json.Unmarshal(input, ahp)
	return err
}
