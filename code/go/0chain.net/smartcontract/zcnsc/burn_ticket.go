package zcnsc

import (
	"encoding/json"

	"github.com/0chain/common/core/currency"
)

type BurnTicket struct {
	EthereumAddress string        `json:"ethereum_address"`
	Hash            string        `json:"hash"`
	Amount          currency.Coin `json:"amount"`
	Nonce           int64         `json:"nonce"`
}

func NewBurnTicket(ethereumAddress, hash string, amount currency.Coin, nonce int64) *BurnTicket {
	m := &BurnTicket{
		EthereumAddress: ethereumAddress,
		Hash:            hash,
		Amount:          amount,
		Nonce:           nonce,
	}
	return m
}

func (b *BurnTicket) Encode() []byte {
	buff, _ := json.Marshal(b)
	return buff
}

func (b *BurnTicket) Decode(input []byte) error {
	err := json.Unmarshal(input, b)
	return err
}
