package zrc20sc

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type tokenNode struct {
	tokenInfo
	TotalSupply state.Balance `json:"total_supply"`
	Available   state.Balance `json:"available"`
}

func (tn *tokenNode) Encode() []byte {
	buff, _ := json.Marshal(tn)
	return buff
}

func (tn *tokenNode) Decode(input []byte) error {
	err := json.Unmarshal(input, tn)
	return err
}

func (tn *tokenNode) GetHash() string {
	return util.ToHex(tn.GetHashBytes())
}

func (tn *tokenNode) GetHashBytes() []byte {
	return encryption.RawHash(tn.Encode())
}

func (tn *tokenNode) getKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + tn.TokenName)
}

func (tn *tokenNode) validate() bool {
	if !tn.validateInfo() {
		return false
	}
	if tn.TotalSupply <= 0 {
		return false
	}
	return true
}

type tokenInfo struct {
	ExchangeRate tokenRatio `json:"exchange_rate"`
	TokenName    string     `json:"token_name"`
}

type tokenRatio struct {
	ZCN   state.Balance `json:"zcn"`
	Other state.Balance `json:other`
}

func (ti *tokenInfo) validateInfo() bool {
	if ti.ExchangeRate.Other <= 0 {
		return false
	}
	if ti.ExchangeRate.ZCN <= 0 {
		return false
	}
	if ti.TokenName == "" {
		return false
	}
	return true
}

type zrc20PoolResponse struct {
	tokenpool.TokenPoolTransferResponse
	FromToken     tokenInfo     `json:"from_token,omitempty"`
	ToToken       tokenInfo     `json:"to_token,omitempty"` //only used in token to token exchange; if transfer between same tokens only FromToken is used
	FromPoolValue state.Balance `json:"from_pool_value,omitempty"`
}

func (zpr *zrc20PoolResponse) encode() []byte {
	buff, _ := json.Marshal(zpr)
	return buff
}

func (zpr *zrc20PoolResponse) decode(input []byte) error {
	err := json.Unmarshal(input, zpr)
	return err
}

type zrc20TransferRequest struct {
	tokenpool.TokenPoolTransferResponse
	FromToken string `json:"from_token_name"`
	ToToken   string `json:"to_token_name"`
}

func (zrc *zrc20TransferRequest) encode() []byte {
	buff, _ := json.Marshal(zrc)
	return buff
}

func (zrc *zrc20TransferRequest) decode(input []byte) error {
	err := json.Unmarshal(input, zrc)
	return err
}
